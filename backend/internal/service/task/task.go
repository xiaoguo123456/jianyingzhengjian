// Package task owns task creation (credit reservation), state transitions, refunds and the schedulers
// (docs/GENERATION_PIPELINE.md §5–6).
package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	gmrouter "yingji/backend/internal/engine/genmodel"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	gm "yingji/backend/internal/provider/genmodel"
	"yingji/backend/internal/service/catalogue"
	"yingji/backend/internal/service/credit"
	"yingji/backend/internal/service/photo"
)

// Enqueuer is implemented by the queue transport.
type Enqueuer interface {
	EnqueueGeneration(ctx context.Context, taskID string) error
	EnqueueNotify(ctx context.Context, taskID string) error
	EnqueueModeration(ctx context.Context, kind, id string) error
}

type Service struct {
	db        *gorm.DB
	cfg       *config.Runtime
	credit    *credit.Service
	catalogue *catalogue.Service
	photo     *photo.Service
	gen       *gmrouter.Router
	enq       Enqueuer
}

func New(db *gorm.DB, cfg *config.Runtime, cr *credit.Service, cat *catalogue.Service, ph *photo.Service, gen *gmrouter.Router, enq Enqueuer) *Service {
	return &Service{db: db, cfg: cfg, credit: cr, catalogue: cat, photo: ph, gen: gen, enq: enq}
}

func (s *Service) SetEnqueuer(e Enqueuer) { s.enq = e }

type CreateInput struct {
	Kind           domain.TaskKind
	SpecID         string
	TemplateID     string
	PhotoID        string
	Params         domain.IDPhotoParams
	Notify         bool
	ParentTaskID   string
	IdempotencyKey string
}

// Create reserves credits and inserts the task in one transaction, then enqueues it.
// Returns existing=true when the idempotency key was seen before.
func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*domain.Task, bool, error) {
	if in.IdempotencyKey == "" {
		return nil, false, apperr.BadRequest("Idempotency-Key required")
	}
	var existing domain.Task
	if err := s.db.WithContext(ctx).First(&existing, "user_id = ? AND idempotency_key = ?", userID, in.IdempotencyKey).Error; err == nil {
		return &existing, true, nil
	}
	ph, err := s.photo.Get(ctx, userID, in.PhotoID)
	if err != nil {
		return nil, false, err
	}
	if ph.CheckStatus != domain.CheckPassed || ph.ExpiresAt.Before(clock.Now()) {
		return nil, false, apperr.PhotoRejected([]string{"expired"})
	}

	t := &domain.Task{ID: idgen.New(), UserID: userID, IdempotencyKey: in.IdempotencyKey, Kind: in.Kind, PhotoID: in.PhotoID,
		Status: domain.TaskWaiting, NotifyRequested: in.Notify, CreatedAt: clock.Now()}
	stage := domain.StageQueued
	t.Stage = &stage
	if in.ParentTaskID != "" {
		t.ParentTaskID = &in.ParentTaskID
	}
	var cost uint
	switch in.Kind {
	case domain.TaskKindIDPhoto:
		sp, err := s.catalogue.Spec(ctx, in.SpecID)
		if err != nil {
			return nil, false, err
		}
		if !allowedBg(sp, in.Params.Bg) {
			in.Params.Bg = sp.BgDefault
		}
		if in.Params.Clothing == "" {
			in.Params.Clothing = "keep"
		}
		if in.Params.Beauty == "" {
			in.Params.Beauty = "natural"
		}
		t.Module = domain.ModuleIDPhoto
		t.SpecID = &sp.ID
		t.Params = domain.MustJSON(in.Params)
		t.UsesGenmodel = true // every ID photo is drawn by the gen model (D-26)
		cost = 1
	case domain.TaskKindTemplate:
		tp, err := s.catalogue.Template(ctx, in.TemplateID)
		if err != nil {
			return nil, false, err
		}
		t.Module = tp.Module
		t.TemplateID = &tp.ID
		t.Params = domain.JSON("{}")
		t.UsesGenmodel = true
		cost = tp.CreditCost
		if cost == 0 {
			cost = 1
		}
	default:
		return nil, false, apperr.BadRequest("unknown kind")
	}
	if t.UsesGenmodel && !s.gen.AnyAvailable(gm.ModeImg2Img) && !s.gen.AnyAvailable(gm.ModeEdit) {
		return nil, false, apperr.GenerationUnavailable()
	}
	t.CreditsConsumed = cost

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if cost > 0 {
			row, err := s.credit.Consume(ctx, tx, userID, cost, "task", t.ID)
			if err != nil {
				return err
			}
			t.ConsumeLedgerID = &row.ID
		}
		if err := tx.Create(t).Error; err != nil {
			if isDuplicate(err) {
				return errDuplicateKey
			}
			return err
		}
		return nil
	})
	if errors.Is(err, errDuplicateKey) {
		if err := s.db.WithContext(ctx).First(&existing, "user_id = ? AND idempotency_key = ?", userID, in.IdempotencyKey).Error; err == nil {
			return &existing, true, nil
		}
	}
	if err != nil {
		return nil, false, err
	}
	if s.enq != nil {
		if err := s.enq.EnqueueGeneration(ctx, t.ID); err != nil {
			// requeue-stuck will pick it up (docs/BACKEND_ARCHITECTURE.md §5)
			_ = err
		}
	}
	return t, false, nil
}

var errDuplicateKey = errors.New("duplicate key")

func isDuplicate(err error) bool {
	return err != nil && (contains(err.Error(), "Duplicate entry") || contains(err.Error(), "23505"))
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func allowedBg(sp *domain.Spec, bg string) bool {
	var allowed []string
	_ = sp.BgAllowed.Into(&allowed)
	for _, a := range allowed {
		if a == bg {
			return true
		}
	}
	return false
}

func (s *Service) Get(ctx context.Context, userID, id string) (*domain.Task, error) {
	var t domain.Task
	if err := s.db.WithContext(ctx).First(&t, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, apperr.NotFound("任务")
	}
	return &t, nil
}

func (s *Service) Load(ctx context.Context, id string) (*domain.Task, error) {
	var t domain.Task
	if err := s.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, apperr.NotFound("任务")
	}
	return &t, nil
}

func (s *Service) List(ctx context.Context, userID, status string, page, size int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&domain.Task{}).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []domain.Task
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

// Regenerate creates a new task with the same inputs (new seed happens in the pipeline).
// Regenerate creates a new task with the same photo and target. For ID photos a non-empty bg replaces the
// background colour, which is how a user changes the background (a new gen task, D-26).
func (s *Service) Regenerate(ctx context.Context, userID, taskID, idemKey, bg string) (*domain.Task, bool, error) {
	src, err := s.Get(ctx, userID, taskID)
	if err != nil {
		return nil, false, err
	}
	in := CreateInput{Kind: src.Kind, PhotoID: src.PhotoID, Notify: src.NotifyRequested, ParentTaskID: src.ID, IdempotencyKey: idemKey}
	if src.SpecID != nil {
		in.SpecID = *src.SpecID
	}
	if src.TemplateID != nil {
		in.TemplateID = *src.TemplateID
	}
	_ = src.Params.Into(&in.Params)
	if bg != "" && src.Kind == domain.TaskKindIDPhoto {
		in.Params.Bg = bg
	}
	return s.Create(ctx, userID, in)
}

// TargetNames resolves template/spec names for a list of tasks.
func (s *Service) TargetNames(ctx context.Context, tasks []domain.Task) map[string]string {
	var tids, sids []string
	for _, t := range tasks {
		if t.TemplateID != nil {
			tids = append(tids, *t.TemplateID)
		}
		if t.SpecID != nil {
			sids = append(sids, *t.SpecID)
		}
	}
	tpls := s.catalogue.TemplatesByIDs(ctx, tids)
	specs := s.catalogue.SpecsByIDs(ctx, sids)
	out := map[string]string{}
	for _, t := range tasks {
		if t.TemplateID != nil {
			out[t.ID] = tpls[*t.TemplateID].Name
		} else if t.SpecID != nil {
			out[t.ID] = specs[*t.SpecID].Name
		}
	}
	return out
}

// --- worker-side transitions ---

// Start moves waiting → processing. Returns false if the task was not waiting (idempotent handlers).
func (s *Service) Start(ctx context.Context, id string) (bool, error) {
	now := clock.Now()
	res := s.db.WithContext(ctx).Model(&domain.Task{}).Where("id = ? AND status = ?", id, domain.TaskWaiting).
		Updates(map[string]any{"status": domain.TaskProcessing, "stage": domain.StageProcessing, "started_at": now})
	return res.RowsAffected == 1, res.Error
}

func (s *Service) SetStage(ctx context.Context, id string, stage domain.TaskStage) {
	s.db.WithContext(ctx).Model(&domain.Task{}).Where("id = ? AND status = ?", id, domain.TaskProcessing).Update("stage", stage)
}

// Succeed stores the work and marks the task successful.
func (s *Service) Succeed(ctx context.Context, t *domain.Task, w *domain.Work, cost int, provider, ref string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(w).Error; err != nil {
			return err
		}
		now := clock.Now()
		upd := map[string]any{"status": domain.TaskSuccess, "stage": nil, "work_id": w.ID, "cost_cents": cost, "finished_at": now}
		if provider != "" {
			upd["provider"] = provider
			upd["provider_ref"] = ref
		}
		res := tx.Model(&domain.Task{}).Where("id = ? AND status = ?", t.ID, domain.TaskProcessing).Updates(upd)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("task %s not processing", t.ID)
		}
		return nil
	})
}

// Fail marks the task failed with a code and refunds the credit (idempotent).
// A task that already succeeded is left alone; see RejectContent for outputs removed by moderation.
func (s *Service) Fail(ctx context.Context, taskID, code string, cost int) error {
	_, err := s.fail(ctx, taskID, code, &cost, func(st domain.TaskStatus) bool { return st != domain.TaskSuccess })
	return err
}

// RejectContent fails a task whose output was flagged by moderation and refunds its credit,
// also after the task succeeded (D-15). The provider cost already recorded is kept.
func (s *Service) RejectContent(ctx context.Context, taskID string) error {
	_, err := s.fail(ctx, taskID, domain.ErrContentRejected, nil, func(st domain.TaskStatus) bool { return st != domain.TaskFailed })
	return err
}

// fail moves the task to failed and refunds a consumed credit in one transaction, if allow accepts the
// current status. A nil cost keeps the recorded cost. Reports whether the task changed.
func (s *Service) fail(ctx context.Context, taskID, code string, cost *int, allow func(domain.TaskStatus) bool) (bool, error) {
	msg := domain.TaskErrorMessages[code]
	if msg == "" {
		msg = domain.TaskErrorMessages[domain.ErrProvider]
	}
	changed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t domain.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t, "id = ?", taskID).Error; err != nil {
			return err
		}
		if !allow(t.Status) {
			return nil
		}
		now := clock.Now()
		upd := map[string]any{"status": domain.TaskFailed, "stage": nil, "error_code": code, "error_message": msg, "finished_at": now}
		if cost != nil {
			upd["cost_cents"] = *cost
		}
		if t.ConsumeLedgerID != nil && t.RefundLedgerID == nil {
			var consume domain.CreditLedger
			if err := tx.First(&consume, "id = ?", *t.ConsumeLedgerID).Error; err != nil {
				return err
			}
			refund, err := s.credit.Refund(ctx, tx, &consume)
			if err != nil {
				return err
			}
			upd["refund_ledger_id"] = refund.ID
		}
		changed = true
		return tx.Model(&domain.Task{}).Where("id = ?", taskID).Updates(upd).Error
	})
	return changed && err == nil, err
}

// ExpireProcessing fails tasks past the deadline (docs/DECISIONS.md D-06).
func (s *Service) ExpireProcessing(ctx context.Context) (int, error) {
	deadline := clock.Now().Add(-time.Duration(s.cfg.Int(ctx, "task_timeout_seconds")) * time.Second)
	var rows []domain.Task
	if err := s.db.WithContext(ctx).Where("status = ? AND started_at < ?", domain.TaskProcessing, deadline).Limit(200).Find(&rows).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, t := range rows {
		if err := s.Fail(ctx, t.ID, domain.ErrTimeout, t.CostCents); err == nil {
			n++
		}
	}
	return n, nil
}

// ExpireWaiting fails and refunds tasks that never started within task_queue_timeout_seconds, e.g. because
// their queue job was lost or no worker is running. The bound is never shorter than task_timeout_seconds.
// A job that starts later finds the task no longer waiting and skips it.
func (s *Service) ExpireWaiting(ctx context.Context) (int, error) {
	sec := max(s.cfg.Int(ctx, "task_queue_timeout_seconds"), s.cfg.Int(ctx, "task_timeout_seconds"))
	deadline := clock.Now().Add(-time.Duration(sec) * time.Second)
	var rows []domain.Task
	if err := s.db.WithContext(ctx).Where("status = ? AND created_at < ?", domain.TaskWaiting, deadline).Limit(200).Find(&rows).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, t := range rows {
		if ok, err := s.fail(ctx, t.ID, domain.ErrTimeout, nil, func(st domain.TaskStatus) bool { return st == domain.TaskWaiting }); err == nil && ok {
			n++
		}
	}
	return n, nil
}

// RequeueStuck re-enqueues waiting tasks older than a minute.
func (s *Service) RequeueStuck(ctx context.Context) (int, error) {
	var rows []domain.Task
	if err := s.db.WithContext(ctx).Where("status = ? AND created_at < ?", domain.TaskWaiting, clock.Now().Add(-time.Minute)).Limit(200).Find(&rows).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, t := range rows {
		if s.enq != nil && s.enq.EnqueueGeneration(ctx, t.ID) == nil {
			n++
		}
	}
	return n, nil
}

// RefundMissing refunds failed tasks that consumed but never refunded, and rejects successful tasks whose
// work moderation flagged as risky but whose rejection was missed (consistency job).
func (s *Service) RefundMissing(ctx context.Context) (int, error) {
	var rows []domain.Task
	if err := s.db.WithContext(ctx).Where("status = ? AND consume_ledger_id IS NOT NULL AND refund_ledger_id IS NULL", domain.TaskFailed).Limit(200).Find(&rows).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, t := range rows {
		code := domain.ErrProvider
		if t.ErrorCode != nil {
			code = *t.ErrorCode
		}
		if err := s.Fail(ctx, t.ID, code, t.CostCents); err == nil {
			n++
		}
	}
	var rejected []string
	if err := s.db.WithContext(ctx).Model(&domain.Task{}).Joins("JOIN works ON works.id = tasks.work_id").
		Where("tasks.status = ? AND works.moderation_status = ?", domain.TaskSuccess, domain.ModRisky).
		Limit(200).Pluck("tasks.id", &rejected).Error; err != nil {
		return n, err
	}
	for _, id := range rejected {
		if err := s.RejectContent(ctx, id); err == nil {
			n++
		}
	}
	return n, nil
}

// AdminList lists tasks with filters.
func (s *Service) AdminList(ctx context.Context, status, module, userID string, from, to *time.Time, page, size int) ([]domain.Task, int64, error) {
	q := s.db.WithContext(ctx).Model(&domain.Task{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if module != "" {
		q = q.Where("module = ?", module)
	}
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at < ?", *to)
	}
	var total int64
	q.Count(&total)
	var rows []domain.Task
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}
