package task_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	gmrouter "yingji/backend/internal/engine/genmodel"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	gm "yingji/backend/internal/provider/genmodel"
	"yingji/backend/internal/provider/storage"
	"yingji/backend/internal/service/catalogue"
	"yingji/backend/internal/service/credit"
	"yingji/backend/internal/service/photo"
	"yingji/backend/internal/service/task"
	"yingji/backend/internal/testutil"
)

// recordingEnqueuer captures enqueued task ids instead of talking to Redis.
type recordingEnqueuer struct {
	mu  sync.Mutex
	ids []string
}

func (e *recordingEnqueuer) EnqueueGeneration(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ids = append(e.ids, id)
	return nil
}
func (e *recordingEnqueuer) EnqueueNotify(context.Context, string) error             { return nil }
func (e *recordingEnqueuer) EnqueueModeration(context.Context, string, string) error { return nil }
func (e *recordingEnqueuer) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.ids)
}

// unavailableModel reports no capabilities, so the router cannot serve any mode.
type unavailableModel struct{}

func (unavailableModel) Name() string          { return "down" }
func (unavailableModel) Capabilities() gm.Caps { return gm.Caps{} }
func (unavailableModel) Run(context.Context, gm.Request) (gm.Result, error) {
	return gm.Result{}, apperr.Transient("PROVIDER_ERROR", context.DeadlineExceeded)
}

type fixture struct {
	db      *gorm.DB
	svc     *task.Service
	credit  *credit.Service
	enq     *recordingEnqueuer
	uid     string
	photoID string
}

func setup(t *testing.T, dailyCredits int, models ...gm.Model) *fixture {
	t.Helper()
	db := testutil.DB(t)
	rt := testutil.Runtime(t, db, map[string]any{
		"ads_enabled": true, "daily_free_credits": dailyCredits, "ad_reward_daily_cap": 10,
		"photo_retention_days": 30, "upload_max_bytes": 10485760,
	})
	if len(models) == 0 {
		models = []gm.Model{gm.Mock{}}
	}
	router := gmrouter.NewRouter(models[0].Name(), 2, models...)
	cr := credit.New(db, rt)
	cat := catalogue.New(db, nil)
	var store storage.ObjectStore
	ph := photo.New(db, rt, store, nil)
	enq := &recordingEnqueuer{}
	svc := task.New(db, rt, cr, cat, ph, router, enq)

	uid := idgen.New()
	now := clock.Now()
	if err := db.Create(&domain.User{ID: uid, Status: 1, CreatedAt: now, UpdatedAt: now, LastLoginAt: now}).Error; err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := db.Create(&domain.Category{ID: "cat1", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "常用", Status: 1}).Error; err != nil {
		t.Fatalf("category: %v", err)
	}
	if err := db.Create(&domain.Spec{ID: "sp1", CategoryID: "cat1", Name: "一寸", WidthMM: 25, HeightMM: 35, WidthPx: 295, HeightPx: 413,
		DPI: 300, BgDefault: "#438EDB", BgAllowed: domain.MustJSON([]string{"#FFFFFF", "#438EDB"}), CropRule: domain.JSON("{}"),
		Status: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("spec: %v", err)
	}
	if err := db.Create(&domain.Template{ID: "tp1", Module: domain.ModulePro, Name: "面试职业照", CoverKey: "assets/x.jpg",
		SampleKeys: domain.MustJSON([]string{}), CreditCost: 1, Tags: domain.MustJSON([]string{}),
		GenConfig: domain.MustJSON(domain.GenConfig{Mode: "img2img", Prompt: "p"}), Status: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("template: %v", err)
	}
	pid := idgen.New()
	if err := db.Create(&domain.Photo{ID: pid, UserID: uid, ObjectKey: "originals/x.jpg", Width: 900, Height: 1200, Bytes: 1000,
		SHA256: "abc", CheckStatus: domain.CheckPassed, CheckResult: domain.JSON("{}"), ModerationStatus: domain.ModPass,
		ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now}).Error; err != nil {
		t.Fatalf("photo: %v", err)
	}
	return &fixture{db: db, svc: svc, credit: cr, enq: enq, uid: uid, photoID: pid}
}

func (f *fixture) idPhoto(clothing, beauty string) task.CreateInput {
	return task.CreateInput{Kind: domain.TaskKindIDPhoto, SpecID: "sp1", PhotoID: f.photoID,
		Params: domain.IDPhotoParams{Bg: "#438EDB", Clothing: clothing, Beauty: beauty}, IdempotencyKey: idgen.New()}
}

func (f *fixture) template() task.CreateInput {
	return task.CreateInput{Kind: domain.TaskKindTemplate, TemplateID: "tp1", PhotoID: f.photoID, IdempotencyKey: idgen.New()}
}

func (f *fixture) balance(t *testing.T) credit.Balance {
	t.Helper()
	b, err := f.credit.Get(t.Context(), f.uid)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	return b
}

// Row 3 (D-26): every ID photo is drawn by the gen model and costs one credit, whatever the options.
func TestIDPhotoAlwaysCostsOneCredit(t *testing.T) {
	for _, opt := range [][2]string{{"keep", "natural"}, {"m_black_suit", "natural"}, {"keep", "light"}} {
		t.Run(opt[0]+"/"+opt[1], func(t *testing.T) {
			f := setup(t, 1)
			tk, existing, err := f.svc.Create(t.Context(), f.uid, f.idPhoto(opt[0], opt[1]))
			if err != nil || existing {
				t.Fatalf("create: %v existing=%v", err, existing)
			}
			if !tk.UsesGenmodel || tk.CreditsConsumed != 1 || tk.ConsumeLedgerID == nil {
				t.Fatalf("task = gen %v / credits %d / ledger %v, want true / 1 / set", tk.UsesGenmodel, tk.CreditsConsumed, tk.ConsumeLedgerID)
			}
			if b := f.balance(t); b.Total != 0 {
				t.Errorf("balance = %+v, want 0", b)
			}
			if f.enq.count() != 1 {
				t.Errorf("enqueued %d, want 1", f.enq.count())
			}
		})
	}
}

// Row 2 for ID photos: without credits nothing is created.
func TestIDPhotoWithoutCredits(t *testing.T) {
	f := setup(t, 0)
	if _, _, err := f.svc.Create(t.Context(), f.uid, f.idPhoto("keep", "natural")); !apperr.Is(err, "NO_CREDITS") {
		t.Fatalf("err = %v, want NO_CREDITS", err)
	}
}

// Row 2: a template task without credits is refused and creates no task row.
func TestTemplateTaskWithoutCredits(t *testing.T) {
	f := setup(t, 0)
	_, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if !apperr.Is(err, "NO_CREDITS") {
		t.Fatalf("err = %v, want NO_CREDITS", err)
	}
	var n int64
	f.db.Model(&domain.Task{}).Count(&n)
	if n != 0 {
		t.Errorf("task rows = %d, want 0", n)
	}
	if f.enq.count() != 0 {
		t.Errorf("enqueued %d, want 0", f.enq.count())
	}
}

// Row 9: the same idempotency key returns the same task and charges once.
func TestIdempotentCreate(t *testing.T) {
	f := setup(t, 5)
	in := f.template()
	first, existing, err := f.svc.Create(t.Context(), f.uid, in)
	if err != nil || existing {
		t.Fatalf("first: %v existing=%v", err, existing)
	}
	second, existing, err := f.svc.Create(t.Context(), f.uid, in)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !existing {
		t.Error("second call not reported as existing")
	}
	if first.ID != second.ID {
		t.Errorf("ids differ: %s vs %s", first.ID, second.ID)
	}
	if b := f.balance(t); b.Total != 4 {
		t.Errorf("balance = %+v, want 4 (charged once)", b)
	}
	if f.enq.count() != 1 {
		t.Errorf("enqueued %d times, want 1", f.enq.count())
	}
}

// Rows 11 + 12: failure refunds once, even when Fail is called twice.
func TestFailRefundsOnceAndIsIdempotent(t *testing.T) {
	f := setup(t, 2)
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if b := f.balance(t); b.Total != 1 {
		t.Fatalf("after create: %+v, want 1", b)
	}
	if _, err := f.svc.Start(t.Context(), tk.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := f.svc.Fail(t.Context(), tk.ID, domain.ErrProvider, 0); err != nil {
			t.Fatalf("fail %d: %v", i, err)
		}
	}
	if b := f.balance(t); b.Total != 2 {
		t.Errorf("balance = %+v, want 2 (refunded once)", b)
	}
	var refunds int64
	f.db.Model(&domain.CreditLedger{}).Where("kind = ?", domain.LedgerRefund).Count(&refunds)
	if refunds != 1 {
		t.Errorf("refund rows = %d, want 1", refunds)
	}
	var got domain.Task
	f.db.First(&got, "id = ?", tk.ID)
	if got.Status != domain.TaskFailed || got.RefundLedgerID == nil {
		t.Errorf("task = %s refund=%v, want failed with a refund", got.Status, got.RefundLedgerID)
	}
	if got.ErrorMessage == nil || *got.ErrorMessage != domain.TaskErrorMessages[domain.ErrProvider] {
		t.Errorf("error message = %v, want the documented copy", got.ErrorMessage)
	}
}

// Rows 16–17: with the gen provider unavailable, every task (ID photos included) is refused before charging.
func TestUnavailableGenRefusesAllTasks(t *testing.T) {
	f := setup(t, 5, unavailableModel{})
	if _, _, err := f.svc.Create(t.Context(), f.uid, f.template()); !apperr.Is(err, "GENERATION_UNAVAILABLE") {
		t.Fatalf("template err = %v, want GENERATION_UNAVAILABLE", err)
	}
	if _, _, err := f.svc.Create(t.Context(), f.uid, f.idPhoto("keep", "natural")); !apperr.Is(err, "GENERATION_UNAVAILABLE") {
		t.Fatalf("ID photo err = %v, want GENERATION_UNAVAILABLE", err)
	}
	if b := f.balance(t); b.Total != 5 {
		t.Errorf("balance = %+v, want 5 (nothing consumed)", b)
	}
}

// Row 12 (state machine): a finished task cannot be restarted.
func TestStartIsIdempotent(t *testing.T) {
	f := setup(t, 5)
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	first, err := f.svc.Start(t.Context(), tk.ID)
	if err != nil || !first {
		t.Fatalf("first start = %v, %v; want true", first, err)
	}
	second, err := f.svc.Start(t.Context(), tk.ID)
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if second {
		t.Error("second start returned true; a duplicate worker would run the pipeline twice")
	}
}

// Row 12 (timeout): tasks past the deadline are failed and refunded by the scheduler.
func TestExpireProcessingRefunds(t *testing.T) {
	f := setup(t, 2)
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.Start(t.Context(), tk.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	f.db.Model(&domain.Task{}).Where("id = ?", tk.ID).Update("started_at", clock.Now().Add(-10*time.Minute))
	n, err := f.svc.ExpireProcessing(t.Context())
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired %d, want 1", n)
	}
	var got domain.Task
	f.db.First(&got, "id = ?", tk.ID)
	if got.Status != domain.TaskFailed || got.ErrorCode == nil || *got.ErrorCode != domain.ErrTimeout {
		t.Errorf("task = %s / %v, want failed TIMEOUT", got.Status, got.ErrorCode)
	}
	if b := f.balance(t); b.Total != 2 {
		t.Errorf("balance = %+v, want 2 (refunded)", b)
	}
}

// The consistency job refunds a failed task whose refund was missed.
func TestRefundMissing(t *testing.T) {
	f := setup(t, 2)
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// simulate a crash between marking failed and refunding
	f.db.Model(&domain.Task{}).Where("id = ?", tk.ID).
		Updates(map[string]any{"status": domain.TaskFailed, "error_code": domain.ErrProvider, "finished_at": clock.Now()})
	if b := f.balance(t); b.Total != 1 {
		t.Fatalf("pre-condition balance = %+v, want 1", b)
	}
	n, err := f.svc.RefundMissing(t.Context())
	if err != nil {
		t.Fatalf("refund missing: %v", err)
	}
	if n != 1 {
		t.Fatalf("refunded %d, want 1", n)
	}
	if b := f.balance(t); b.Total != 2 {
		t.Errorf("balance = %+v, want 2", b)
	}
}

// succeeded creates a paid template task and completes it with a work, as the worker would.
func (f *fixture) succeeded(t *testing.T) (*domain.Task, *domain.Work) {
	t.Helper()
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.Start(t.Context(), tk.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	w := &domain.Work{ID: idgen.New(), UserID: f.uid, TaskID: &tk.ID, Module: domain.ModulePro, ObjectKey: "works/x.jpg",
		ThumbKey: "works/x_thumb.jpg", Width: 1, Height: 1, Meta: domain.JSON("{}"), ModerationStatus: domain.ModPending, CreatedAt: clock.Now()}
	if err := f.svc.Succeed(t.Context(), tk, w, 30, "mock", "ref"); err != nil {
		t.Fatalf("succeed: %v", err)
	}
	return tk, w
}

func (f *fixture) refunds(t *testing.T) int64 {
	t.Helper()
	var n int64
	f.db.Model(&domain.CreditLedger{}).Where("kind = ?", domain.LedgerRefund).Count(&n)
	return n
}

// Row 9 of the failure matrix (D-15): an output flagged after success still fails the task and refunds once.
func TestRejectContentRefundsSucceededTask(t *testing.T) {
	f := setup(t, 2)
	tk, _ := f.succeeded(t)
	if b := f.balance(t); b.Total != 1 {
		t.Fatalf("after success: %+v, want 1", b)
	}
	if err := f.svc.Fail(t.Context(), tk.ID, domain.ErrProvider, 0); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if b := f.balance(t); b.Total != 1 {
		t.Fatalf("Fail refunded a successful task: %+v", b)
	}
	for i := 0; i < 2; i++ {
		if err := f.svc.RejectContent(t.Context(), tk.ID); err != nil {
			t.Fatalf("reject %d: %v", i, err)
		}
	}
	if b := f.balance(t); b.Total != 2 {
		t.Errorf("balance = %+v, want 2 (refunded)", b)
	}
	if n := f.refunds(t); n != 1 {
		t.Errorf("refund rows = %d, want 1", n)
	}
	var got domain.Task
	f.db.First(&got, "id = ?", tk.ID)
	if got.Status != domain.TaskFailed || got.ErrorCode == nil || *got.ErrorCode != domain.ErrContentRejected || got.RefundLedgerID == nil {
		t.Errorf("task = %s / %v refund=%v, want failed CONTENT_REJECTED with a refund", got.Status, got.ErrorCode, got.RefundLedgerID)
	}
	if got.CostCents != 30 {
		t.Errorf("cost = %d, want the recorded provider cost 30", got.CostCents)
	}
}

// A missed moderation rejection is caught by the consistency job.
func TestRefundMissingRejectsRiskyWork(t *testing.T) {
	f := setup(t, 2)
	tk, w := f.succeeded(t)
	f.succeeded(t) // a second, clean success must stay untouched
	f.db.Model(&domain.Work{}).Where("id = ?", w.ID).Update("moderation_status", domain.ModRisky)
	n, err := f.svc.RefundMissing(t.Context())
	if err != nil {
		t.Fatalf("refund missing: %v", err)
	}
	if n != 1 {
		t.Fatalf("handled %d, want 1", n)
	}
	var got domain.Task
	f.db.First(&got, "id = ?", tk.ID)
	if got.Status != domain.TaskFailed || got.ErrorCode == nil || *got.ErrorCode != domain.ErrContentRejected {
		t.Errorf("task = %s / %v, want failed CONTENT_REJECTED", got.Status, got.ErrorCode)
	}
	if b := f.balance(t); b.Total != 1 {
		t.Errorf("balance = %+v, want 1 (one of two tasks refunded)", b)
	}
}

// A task that never leaves the queue is failed and refunded, and a late job no longer runs it.
func TestExpireWaitingRefunds(t *testing.T) {
	f := setup(t, 3)
	stale, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	fresh, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	f.db.Model(&domain.Task{}).Where("id = ?", stale.ID).Update("created_at", clock.Now().Add(-31*time.Minute))
	n, err := f.svc.ExpireWaiting(t.Context())
	if err != nil {
		t.Fatalf("expire waiting: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired %d, want 1", n)
	}
	var got domain.Task
	f.db.First(&got, "id = ?", stale.ID)
	if got.Status != domain.TaskFailed || got.ErrorCode == nil || *got.ErrorCode != domain.ErrTimeout || got.RefundLedgerID == nil {
		t.Errorf("stale task = %s / %v refund=%v, want failed TIMEOUT with a refund", got.Status, got.ErrorCode, got.RefundLedgerID)
	}
	var other domain.Task
	f.db.First(&other, "id = ?", fresh.ID)
	if other.Status != domain.TaskWaiting {
		t.Errorf("fresh task = %s, want waiting", other.Status)
	}
	if b := f.balance(t); b.Total != 2 {
		t.Errorf("balance = %+v, want 2 (stale task refunded)", b)
	}
	if started, err := f.svc.Start(t.Context(), stale.ID); err != nil || started {
		t.Errorf("late start = %v, %v; want false", started, err)
	}
}

// An unknown background colour falls back to the spec default rather than being accepted.
func TestInvalidBackgroundFallsBackToDefault(t *testing.T) {
	f := setup(t, 1)
	in := f.idPhoto("keep", "natural")
	in.Params.Bg = "#123456"
	tk, _, err := f.svc.Create(t.Context(), f.uid, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var params domain.IDPhotoParams
	if err := tk.Params.Into(&params); err != nil {
		t.Fatalf("params: %v", err)
	}
	if params.Bg != "#438EDB" {
		t.Errorf("bg = %s, want the spec default #438EDB", params.Bg)
	}
}

// Regenerate copies the inputs of the source task and charges again.
func TestRegenerate(t *testing.T) {
	f := setup(t, 2)
	src, _, err := f.svc.Create(t.Context(), f.uid, f.template())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	again, existing, err := f.svc.Regenerate(t.Context(), f.uid, src.ID, idgen.New(), "")
	if err != nil || existing {
		t.Fatalf("regenerate: %v existing=%v", err, existing)
	}
	if again.ID == src.ID {
		t.Error("regenerate reused the original task id")
	}
	if again.ParentTaskID == nil || *again.ParentTaskID != src.ID {
		t.Errorf("parent = %v, want %s", again.ParentTaskID, src.ID)
	}
	if again.TemplateID == nil || *again.TemplateID != "tp1" {
		t.Errorf("template = %v, want tp1", again.TemplateID)
	}
	if b := f.balance(t); b.Total != 0 {
		t.Errorf("balance = %+v, want 0 (both charged)", b)
	}
}

// Changing an ID photo background is a regenerate with a new bg: same photo and spec, new colour, one credit.
func TestRegenerateWithBackground(t *testing.T) {
	f := setup(t, 2)
	src, _, err := f.svc.Create(t.Context(), f.uid, f.idPhoto("m_black_suit", "natural"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	again, _, err := f.svc.Regenerate(t.Context(), f.uid, src.ID, idgen.New(), "#FFFFFF")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	var params domain.IDPhotoParams
	if err := again.Params.Into(&params); err != nil {
		t.Fatalf("params: %v", err)
	}
	if params.Bg != "#FFFFFF" || params.Clothing != "m_black_suit" {
		t.Errorf("params = %+v, want white background and the original clothing", params)
	}
	if again.SpecID == nil || *again.SpecID != "sp1" || again.PhotoID != src.PhotoID {
		t.Errorf("spec/photo = %v/%s, want sp1/%s", again.SpecID, again.PhotoID, src.PhotoID)
	}
	if b := f.balance(t); b.Total != 0 {
		t.Errorf("balance = %+v, want 0 (both charged)", b)
	}
}

// A task belonging to another user must not be readable.
func TestGetIsScopedToOwner(t *testing.T) {
	f := setup(t, 1)
	tk, _, err := f.svc.Create(t.Context(), f.uid, f.idPhoto("keep", "natural"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.Get(t.Context(), idgen.New(), tk.ID); !apperr.Is(err, "NOT_FOUND") {
		t.Errorf("err = %v, want NOT_FOUND for another user", err)
	}
}
