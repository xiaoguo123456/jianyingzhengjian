// Package queue is the Asynq transport: enqueuing, handlers and the scheduler (docs/BACKEND_ARCHITECTURE.md §6).
package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"yingji/backend/internal/app"
	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pipeline/idphoto"
	"yingji/backend/internal/pipeline/steps"
	tplpipe "yingji/backend/internal/pipeline/template"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/pkg/redisx"
	"yingji/backend/internal/provider/storage"
)

const (
	TypeGeneration   = "generation:run"
	TypeModeration   = "moderation:check"
	TypeNotify       = "notify:task-finished"
	TypePhotoCleanup = "photo:cleanup"
	TypeRequeueStuck = "task:requeue-stuck"
	TypeExpire       = "task:expire-processing"
	TypeShareCleanup = "share:cleanup"
	TypeRollup       = "stats:daily-rollup"
	TypeConsistency  = "consistency:refund"
)

func redisOpt(cfg *config.Config) redisx.QueueOpt { return redisx.QueueOpt{Cfg: cfg} }

// Enqueuer implements task.Enqueuer.
type Enqueuer struct{ c *asynq.Client }

func NewEnqueuer(cfg *config.Config) *Enqueuer { return &Enqueuer{c: asynq.NewClient(redisOpt(cfg))} }
func (e *Enqueuer) Close()                     { _ = e.c.Close() }

type idPayload struct {
	ID   string `json:"id"`
	Kind string `json:"kind,omitempty"`
}

func (e *Enqueuer) enqueue(ctx context.Context, typ string, p any, opts ...asynq.Option) error {
	b, _ := json.Marshal(p)
	_, err := e.c.EnqueueContext(ctx, asynq.NewTask(typ, b), opts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func (e *Enqueuer) EnqueueGeneration(ctx context.Context, taskID string) error {
	return e.enqueue(ctx, TypeGeneration, idPayload{ID: taskID}, asynq.Queue("generation"), asynq.Timeout(5*time.Minute),
		asynq.MaxRetry(1), asynq.TaskID("gen:"+taskID), asynq.Retention(time.Hour))
}

func (e *Enqueuer) EnqueueNotify(ctx context.Context, taskID string) error {
	return e.enqueue(ctx, TypeNotify, idPayload{ID: taskID}, asynq.Queue("default"), asynq.Timeout(30*time.Second), asynq.MaxRetry(3))
}

func (e *Enqueuer) EnqueueModeration(ctx context.Context, kind, id string) error {
	return e.enqueue(ctx, TypeModeration, idPayload{ID: id, Kind: kind}, asynq.Queue("default"), asynq.Timeout(60*time.Second), asynq.MaxRetry(3))
}

// NewServer builds the Asynq server and the periodic scheduler.
func NewServer(a *app.App, enq *Enqueuer) (*asynq.Server, *asynq.Scheduler) {
	conc := a.Cfg.GenConcurrency
	if conc < 1 {
		conc = 4
	}
	srv := asynq.NewServer(redisOpt(a.Cfg), asynq.Config{
		Concurrency: conc + 4,
		Queues:      map[string]int{"generation": 6, "default": 3, "low": 1},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
			a.Log.Warn("asynq task failed", "type", t.Type(), "err", err)
		}),
	})
	sched := asynq.NewScheduler(redisOpt(a.Cfg), &asynq.SchedulerOpts{Location: clock.Shanghai})
	reg := func(spec, typ string, opts ...asynq.Option) {
		if _, err := sched.Register(spec, asynq.NewTask(typ, nil), append(opts, asynq.Queue("low"))...); err != nil {
			a.Log.Error("schedule", "type", typ, "err", err)
		}
	}
	reg("* * * * *", TypeRequeueStuck, asynq.Timeout(time.Minute))
	reg("* * * * *", TypeExpire, asynq.Timeout(time.Minute))
	reg("0 4 * * *", TypePhotoCleanup, asynq.Timeout(10*time.Minute))
	reg("30 4 * * *", TypeShareCleanup, asynq.Timeout(10*time.Minute))
	reg("10 0 * * *", TypeRollup, asynq.Timeout(10*time.Minute))
	reg("*/30 * * * *", TypeConsistency, asynq.Timeout(5*time.Minute))
	return srv, sched
}

// Mux registers handlers.
func Mux(a *app.App, enq *Enqueuer) *asynq.ServeMux {
	h := &handlers{a: a, enq: enq}
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeGeneration, h.generation)
	mux.HandleFunc(TypeNotify, h.notify)
	mux.HandleFunc(TypeModeration, h.moderation)
	mux.HandleFunc(TypePhotoCleanup, func(ctx context.Context, t *asynq.Task) error {
		n, err := a.Photo.Cleanup(ctx)
		a.Log.Info("photo cleanup", "n", n)
		return err
	})
	mux.HandleFunc(TypeRequeueStuck, func(ctx context.Context, t *asynq.Task) error { _, err := a.Task.RequeueStuck(ctx); return err })
	mux.HandleFunc(TypeExpire, func(ctx context.Context, t *asynq.Task) error {
		n, err := a.Task.ExpireProcessing(ctx)
		if n > 0 {
			a.Log.Warn("expired tasks", "n", n)
		}
		return err
	})
	mux.HandleFunc(TypeShareCleanup, func(ctx context.Context, t *asynq.Task) error { _, err := a.Share.Cleanup(ctx); return err })
	mux.HandleFunc(TypeConsistency, func(ctx context.Context, t *asynq.Task) error {
		n, err := a.Task.RefundMissing(ctx)
		if n > 0 {
			a.Log.Error("consistency: refunded missing", "n", n)
		}
		return err
	})
	mux.HandleFunc(TypeRollup, h.rollup)
	return mux
}

type handlers struct {
	a   *app.App
	enq *Enqueuer
}

func (h *handlers) generation(ctx context.Context, t *asynq.Task) error {
	var p idPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload: %w", err)
	}
	a := h.a
	task, err := a.Task.Load(ctx, p.ID)
	if err != nil {
		return nil
	}
	started, err := a.Task.Start(ctx, task.ID)
	if err != nil {
		return err
	}
	if !started {
		return nil // already processed
	}
	log := a.Log.With("task", task.ID, "module", task.Module)
	photo, err := a.Photo.Load(ctx, task.PhotoID)
	if err != nil {
		return a.Task.Fail(ctx, task.ID, domain.ErrStorage, 0)
	}
	raw, err := a.Photo.Bytes(ctx, photo)
	if err != nil {
		return a.Task.Fail(ctx, task.ID, domain.ErrStorage, 0)
	}
	st := &steps.State{Task: task, Raw: raw, OnStage: func(s domain.TaskStage) { a.Task.SetStage(ctx, task.ID, s) }}
	_ = task.Params.Into(&st.Params)
	if task.SpecID != nil {
		if st.Spec, err = a.Catalogue.Spec(ctx, *task.SpecID); err != nil {
			return a.Task.Fail(ctx, task.ID, domain.ErrProvider, 0)
		}
	}
	if task.TemplateID != nil {
		if st.Template, err = a.Catalogue.Template(ctx, *task.TemplateID); err != nil {
			return a.Task.Fail(ctx, task.ID, domain.ErrProvider, 0)
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 270*time.Second)
	defer cancel()
	var out *steps.Output
	switch task.Kind {
	case domain.TaskKindIDPhoto:
		out, err = idphoto.Run(runCtx, a.Pipeline, st)
	default:
		out, err = tplpipe.Run(runCtx, a.Pipeline, st)
	}
	if err != nil {
		code := apperr.From(err).Code
		if _, known := domain.TaskErrorMessages[code]; !known {
			code = domain.ErrProvider
		}
		log.Warn("pipeline failed", "code", code, "err", err)
		return a.Task.Fail(ctx, task.ID, code, st.Cost)
	}

	w := &domain.Work{ID: idgen.New(), UserID: task.UserID, TaskID: &task.ID, Module: task.Module, SpecID: task.SpecID, TemplateID: task.TemplateID,
		Width: out.Width, Height: out.Height, Meta: domain.MustJSON(out.Meta), AILabel: out.AILabel, ModerationStatus: domain.ModPending, CreatedAt: clock.Now()}
	ext := "jpg"
	if out.Format == "png" {
		ext = "png"
	}
	w.ObjectKey = "works/" + task.UserID + "/" + w.ID + "." + ext
	w.ThumbKey = "works/" + task.UserID + "/" + w.ID + "_thumb.jpg"
	if err := a.Store.Put(ctx, w.ObjectKey, bytes.NewReader(out.Image), int64(len(out.Image)), storage.ContentTypeFor(out.Format)); err != nil {
		return a.Task.Fail(ctx, task.ID, domain.ErrStorage, st.Cost)
	}
	if err := a.Store.Put(ctx, w.ThumbKey, bytes.NewReader(out.Thumb), int64(len(out.Thumb)), "image/jpeg"); err != nil {
		return a.Task.Fail(ctx, task.ID, domain.ErrStorage, st.Cost)
	}
	if out.Alpha != nil {
		key := "works/" + task.UserID + "/" + w.ID + "_alpha.png"
		if err := a.Store.Put(ctx, key, bytes.NewReader(out.Alpha), int64(len(out.Alpha)), "image/png"); err == nil {
			w.AlphaKey = &key
		}
	}
	if err := a.Task.Succeed(ctx, task, w, out.Cost, out.Provider, out.ProviderRef); err != nil {
		log.Error("succeed failed", "err", err)
		return err
	}
	task.Status = domain.TaskSuccess
	task.WorkID = &w.ID
	_ = h.enq.EnqueueModeration(ctx, "work", w.ID)
	_ = h.enq.EnqueueNotify(ctx, task.ID)
	a.Share.GrantForTask(ctx, task)
	log.Info("task success", "work", w.ID, "cost_cents", out.Cost, "provider", out.Provider)
	return nil
}

func (h *handlers) notify(ctx context.Context, t *asynq.Task) error {
	var p idPayload
	_ = json.Unmarshal(t.Payload(), &p)
	task, err := h.a.Task.Load(ctx, p.ID)
	if err != nil {
		return nil
	}
	names := h.a.Task.TargetNames(ctx, []domain.Task{*task})
	return h.a.Notify.TaskFinished(ctx, task, names[task.ID])
}

// moderation submits to WeChat when configured and reachable; otherwise marks pass.
func (h *handlers) moderation(ctx context.Context, t *asynq.Task) error {
	var p idPayload
	_ = json.Unmarshal(t.Payload(), &p)
	a := h.a
	if !a.WX.Configured() || a.LocalStore != nil {
		if p.Kind == "work" {
			return a.Work.SetModeration(ctx, p.ID, domain.ModPass)
		}
		return a.Photo.SetModeration(ctx, p.ID, domain.ModPass)
	}
	var key, userID string
	if p.Kind == "work" {
		w, err := a.Work.Load(ctx, p.ID)
		if err != nil {
			return nil
		}
		key, userID = w.ThumbKey, w.UserID
	} else {
		ph, err := a.Photo.Load(ctx, p.ID)
		if err != nil {
			return nil
		}
		key, userID = ph.ObjectKey, ph.UserID
	}
	url, err := a.Store.SignedURL(ctx, key, time.Hour)
	if err != nil {
		return err
	}
	openid, _ := a.Auth.OpenID(ctx, userID)
	trace, err := a.WX.MediaCheckAsync(ctx, url, openid, 1)
	if err != nil {
		return err
	}
	return a.RDB.Set(ctx, "mod:"+trace, p.Kind+":"+p.ID, 48*time.Hour).Err()
}

// rollup aggregates yesterday into daily_stats.
func (h *handlers) rollup(ctx context.Context, t *asynq.Task) error {
	a := h.a
	end := clock.Today()
	start := end.Add(-24 * time.Hour)
	type row struct {
		Module          domain.Module
		TemplateID      *string
		SpecID          *string
		TasksCreated    uint
		TasksSuccess    uint
		TasksFailed     uint
		CreditsConsumed uint
		CreditsRefunded uint
		CostCents       uint
	}
	var rows []row
	err := a.DB.WithContext(ctx).Raw(`SELECT module, template_id, spec_id, COUNT(*) AS tasks_created,
		SUM(CASE WHEN status='success' THEN 1 ELSE 0 END) AS tasks_success, SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END) AS tasks_failed,
		SUM(credits_consumed) AS credits_consumed, SUM(CASE WHEN refund_ledger_id IS NOT NULL THEN 1 ELSE 0 END) AS credits_refunded, SUM(cost_cents) AS cost_cents
		FROM tasks WHERE created_at >= ? AND created_at < ? GROUP BY module, template_id, spec_id`, start, end).Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, r := range rows {
		ds := domain.DailyStat{Date: start, Module: r.Module, TasksCreated: r.TasksCreated, TasksSuccess: r.TasksSuccess, TasksFailed: r.TasksFailed,
			CreditsConsumed: r.CreditsConsumed, CreditsRefunded: r.CreditsRefunded, CostCents: r.CostCents}
		if r.TemplateID != nil {
			ds.TemplateID = *r.TemplateID
		}
		if r.SpecID != nil {
			ds.SpecID = *r.SpecID
		}
		a.DB.WithContext(ctx).Save(&ds)
	}
	a.Log.Info("daily rollup", "date", start.Format("2006-01-02"), "rows", len(rows))
	return nil
}
