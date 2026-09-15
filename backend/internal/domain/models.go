package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID              string     `gorm:"primaryKey;size:26" json:"id"`
	Nickname        *string    `gorm:"size:64" json:"nickname"`
	AvatarKey       *string    `gorm:"size:255" json:"avatar_key"`
	Status          int8       `gorm:"default:1" json:"status"`
	PrivacyAgreedAt *time.Time `gorm:"type:datetime(3)" json:"privacy_agreed_at"`
	PrivacyVersion  *string    `gorm:"size:32" json:"privacy_version"`
	AcquiredShareID *string    `gorm:"size:26" json:"acquired_share_id"`
	LastLoginAt     time.Time  `gorm:"type:datetime(3)" json:"last_login_at"`
	CreatedAt       time.Time  `gorm:"type:datetime(3)" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"type:datetime(3)" json:"updated_at"`
}

type UserIdentity struct {
	ID          string    `gorm:"primaryKey;size:26" json:"id"`
	UserID      string    `gorm:"size:26;index" json:"user_id"`
	Provider    string    `gorm:"size:16;uniqueIndex:uk_provider_uid" json:"provider"`
	ProviderUID string    `gorm:"size:64;uniqueIndex:uk_provider_uid" json:"provider_uid"`
	UnionID     *string   `gorm:"size:64;index" json:"union_id"`
	CreatedAt   time.Time `gorm:"type:datetime(3)" json:"created_at"`
}

type CreditAccount struct {
	UserID             string    `gorm:"primaryKey;size:26" json:"user_id"`
	DailyFreeRemaining uint      `gorm:"not null" json:"daily_free_remaining"`
	DailyResetDate     time.Time `gorm:"type:date" json:"daily_reset_date"`
	BonusCredits       uint      `gorm:"not null" json:"bonus_credits"`
	AdRewardsToday     uint      `gorm:"not null" json:"ad_rewards_today"`
	UpdatedAt          time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

type CreditLedger struct {
	ID                string     `gorm:"primaryKey;size:26" json:"id"`
	UserID            string     `gorm:"size:26;index" json:"user_id"`
	Kind              LedgerKind `gorm:"size:16;uniqueIndex:uk_ledger_ref" json:"kind"`
	Bucket            Bucket     `gorm:"size:8" json:"bucket"`
	Delta             int        `gorm:"not null" json:"delta"`
	BalanceDailyAfter uint       `gorm:"not null" json:"balance_daily_after"`
	BalanceBonusAfter uint       `gorm:"not null" json:"balance_bonus_after"`
	RefType           string     `gorm:"size:32;uniqueIndex:uk_ledger_ref" json:"ref_type"`
	RefID             string     `gorm:"size:64;uniqueIndex:uk_ledger_ref" json:"ref_id"`
	Note              *string    `gorm:"size:255" json:"note"`
	CreatedAt         time.Time  `gorm:"type:datetime(3)" json:"created_at"`
}

func (CreditLedger) TableName() string { return "credit_ledger" }

type AdSession struct {
	ID               string          `gorm:"primaryKey;size:26" json:"id"`
	UserID           string          `gorm:"size:26;index" json:"user_id"`
	AdUnitID         string          `gorm:"size:64" json:"ad_unit_id"`
	Status           AdSessionStatus `gorm:"size:16" json:"status"`
	RejectReason     *string         `gorm:"size:64" json:"reject_reason"`
	ClientIsEnded    *bool           `json:"client_is_ended"`
	ServerVerifiedAt *time.Time      `gorm:"type:datetime(3)" json:"server_verified_at"`
	TransID          *string         `gorm:"size:128;uniqueIndex" json:"trans_id"`
	CreatedAt        time.Time       `gorm:"type:datetime(3)" json:"created_at"`
	ClaimedAt        *time.Time      `gorm:"type:datetime(3)" json:"claimed_at"`
}

type Photo struct {
	ID               string           `gorm:"primaryKey;size:26" json:"id"`
	UserID           string           `gorm:"size:26;index" json:"user_id"`
	ObjectKey        string           `gorm:"size:255" json:"object_key"`
	Width            int              `gorm:"not null" json:"width"`
	Height           int              `gorm:"not null" json:"height"`
	Bytes            int              `gorm:"not null" json:"bytes"`
	SHA256           string           `gorm:"column:sha256;size:64;index" json:"sha256"`
	CheckStatus      CheckStatus      `gorm:"size:16" json:"check_status"`
	CheckResult      JSON             `gorm:"type:json" json:"check_result"`
	ModerationStatus ModerationStatus `gorm:"size:16;default:pending" json:"moderation_status"`
	ExpiresAt        time.Time        `gorm:"type:datetime(3)" json:"expires_at"`
	CreatedAt        time.Time        `gorm:"type:datetime(3)" json:"created_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"type:datetime(3);index" json:"deleted_at"`
}

type Category struct {
	ID       string  `gorm:"primaryKey;size:26" json:"id"`
	Module   Module  `gorm:"size:16;index" json:"module"`
	Kind     string  `gorm:"size:16" json:"kind"` // spec | template
	Name     string  `gorm:"size:32" json:"name"`
	Icon     *string `gorm:"size:64" json:"icon"`
	CoverKey *string `gorm:"size:255" json:"cover_key"`
	Desc     *string `gorm:"size:64" json:"desc"`
	Sort     int     `gorm:"default:0" json:"sort"`
	Status   int8    `gorm:"default:1" json:"status"`
}

type Spec struct {
	ID         string    `gorm:"primaryKey;size:26" json:"id"`
	CategoryID string    `gorm:"size:26;index" json:"category_id"`
	Name       string    `gorm:"size:32" json:"name"`
	WidthMM    float64   `gorm:"type:decimal(5,1)" json:"width_mm"`
	HeightMM   float64   `gorm:"type:decimal(5,1)" json:"height_mm"`
	WidthPx    int       `gorm:"not null" json:"width_px"`
	HeightPx   int       `gorm:"not null" json:"height_px"`
	DPI        int       `gorm:"default:300" json:"dpi"`
	BgDefault  string    `gorm:"size:7" json:"bg_default"`
	BgAllowed  JSON      `gorm:"type:json" json:"bg_allowed"`
	CropRule   JSON      `gorm:"type:json" json:"crop_rule"`
	SourceNote *string   `gorm:"size:255" json:"source_note"`
	Note       *string   `gorm:"size:64" json:"note"`
	IsHot      bool      `gorm:"default:false" json:"is_hot"`
	Sort       int       `gorm:"default:0" json:"sort"`
	Status     int8      `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `gorm:"type:datetime(3)" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

type Template struct {
	ID         string    `gorm:"primaryKey;size:26" json:"id"`
	Module     Module    `gorm:"size:16;index:idx_tpl_list,priority:1" json:"module"`
	CategoryID *string   `gorm:"size:26;index" json:"category_id"`
	Name       string    `gorm:"size:32" json:"name"`
	Subtitle   *string   `gorm:"size:64" json:"subtitle"`
	CoverKey   string    `gorm:"size:255" json:"cover_key"`
	SampleKeys JSON      `gorm:"type:json" json:"sample_keys"`
	CreditCost uint      `gorm:"default:1" json:"credit_cost"`
	GenConfig  JSON      `gorm:"type:json" json:"gen_config"`
	Tags       JSON      `gorm:"type:json" json:"tags"`
	IsHot      bool      `gorm:"default:false;index:idx_tpl_list,priority:3" json:"is_hot"`
	Sort       int       `gorm:"default:0;index:idx_tpl_list,priority:4" json:"sort"`
	Status     int8      `gorm:"default:1;index:idx_tpl_list,priority:2" json:"status"`
	CreatedAt  time.Time `gorm:"type:datetime(3)" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

type Collection struct {
	ID          string  `gorm:"primaryKey;size:26" json:"id"`
	Module      Module  `gorm:"size:16;index" json:"module"`
	Name        string  `gorm:"size:32" json:"name"`
	CoverKey    string  `gorm:"size:255" json:"cover_key"`
	Description *string `gorm:"size:255" json:"description"`
	Sort        int     `gorm:"default:0" json:"sort"`
	Status      int8    `gorm:"default:1" json:"status"`
}

type CollectionTemplate struct {
	CollectionID string `gorm:"primaryKey;size:26" json:"collection_id"`
	TemplateID   string `gorm:"primaryKey;size:26" json:"template_id"`
	Sort         int    `gorm:"default:0" json:"sort"`
}

type Banner struct {
	ID       string     `gorm:"primaryKey;size:26" json:"id"`
	Module   Module     `gorm:"size:16;index" json:"module"`
	Title    string     `gorm:"size:64" json:"title"`
	Subtitle *string    `gorm:"size:64" json:"subtitle"`
	ImageKey string     `gorm:"size:255" json:"image_key"`
	Link     JSON       `gorm:"type:json" json:"link"`
	Sort     int        `gorm:"default:0" json:"sort"`
	Status   int8       `gorm:"default:1" json:"status"`
	StartAt  *time.Time `gorm:"type:datetime(3)" json:"start_at"`
	EndAt    *time.Time `gorm:"type:datetime(3)" json:"end_at"`
}

type Task struct {
	ID              string     `gorm:"primaryKey;size:26" json:"id"`
	UserID          string     `gorm:"size:26;index:idx_task_user,priority:1;uniqueIndex:uk_task_idem,priority:1" json:"user_id"`
	IdempotencyKey  string     `gorm:"size:36;uniqueIndex:uk_task_idem,priority:2" json:"idempotency_key"`
	Module          Module     `gorm:"size:16" json:"module"`
	Kind            TaskKind   `gorm:"size:16" json:"kind"`
	SpecID          *string    `gorm:"size:26" json:"spec_id"`
	TemplateID      *string    `gorm:"size:26" json:"template_id"`
	PhotoID         string     `gorm:"size:26" json:"photo_id"`
	ParentTaskID    *string    `gorm:"size:26" json:"parent_task_id"`
	Params          JSON       `gorm:"type:json" json:"params"`
	Status          TaskStatus `gorm:"size:16;index:idx_task_status,priority:1" json:"status"`
	Stage           *TaskStage `gorm:"size:16" json:"stage"`
	Provider        *string    `gorm:"size:32" json:"provider"`
	ProviderRef     *string    `gorm:"size:128" json:"provider_ref"`
	WorkID          *string    `gorm:"size:26" json:"work_id"`
	ErrorCode       *string    `gorm:"size:32" json:"error_code"`
	ErrorMessage    *string    `gorm:"size:255" json:"error_message"`
	CostCents       int        `gorm:"default:0" json:"cost_cents"`
	UsesGenmodel    bool       `gorm:"default:false" json:"uses_genmodel"`
	CreditsConsumed uint       `gorm:"default:0" json:"credits_consumed"`
	ConsumeLedgerID *string    `gorm:"size:26" json:"consume_ledger_id"`
	RefundLedgerID  *string    `gorm:"size:26" json:"refund_ledger_id"`
	NotifyRequested bool       `gorm:"default:false" json:"notify_requested"`
	CreatedAt       time.Time  `gorm:"type:datetime(3);index:idx_task_user,priority:2" json:"created_at"`
	StartedAt       *time.Time `gorm:"type:datetime(3);index:idx_task_status,priority:2" json:"started_at"`
	FinishedAt      *time.Time `gorm:"type:datetime(3)" json:"finished_at"`
}

type Work struct {
	ID               string           `gorm:"primaryKey;size:26" json:"id"`
	UserID           string           `gorm:"size:26;index" json:"user_id"`
	TaskID           *string          `gorm:"size:26;uniqueIndex" json:"task_id"` // nil for free recolors, which create no task
	Module           Module           `gorm:"size:16" json:"module"`
	SpecID           *string          `gorm:"size:26" json:"spec_id"`
	TemplateID       *string          `gorm:"size:26" json:"template_id"`
	ObjectKey        string           `gorm:"size:255" json:"object_key"`
	ThumbKey         string           `gorm:"size:255" json:"thumb_key"`
	AlphaKey         *string          `gorm:"size:255" json:"alpha_key"`
	Width            int              `gorm:"not null" json:"width"`
	Height           int              `gorm:"not null" json:"height"`
	Meta             JSON             `gorm:"type:json" json:"meta"`
	AILabel          bool             `gorm:"default:false" json:"ai_label"`
	ModerationStatus ModerationStatus `gorm:"size:16;default:pending" json:"moderation_status"`
	CreatedAt        time.Time        `gorm:"type:datetime(3)" json:"created_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"type:datetime(3);index" json:"deleted_at"`
}

type Favorite struct {
	UserID     string    `gorm:"primaryKey;size:26" json:"user_id"`
	TemplateID string    `gorm:"primaryKey;size:26" json:"template_id"`
	CreatedAt  time.Time `gorm:"type:datetime(3)" json:"created_at"`
}

type Share struct {
	ID           string     `gorm:"primaryKey;size:26" json:"id"`
	UserID       string     `gorm:"size:26;index" json:"user_id"`
	Type         ShareType  `gorm:"size:16" json:"type"`
	Module       Module     `gorm:"size:16" json:"module"`
	TemplateID   *string    `gorm:"size:26" json:"template_id"`
	SpecID       *string    `gorm:"size:26" json:"spec_id"`
	WorkID       *string    `gorm:"size:26;index" json:"work_id"`
	PreviewKey   *string    `gorm:"size:255" json:"preview_key"`
	PosterKey    *string    `gorm:"size:255" json:"poster_key"`
	Path         string     `gorm:"size:255" json:"path"`
	Title        string     `gorm:"size:128" json:"title"`
	Status       string     `gorm:"size:16;default:active" json:"status"`
	Opens        uint       `gorm:"default:0" json:"opens"`
	LastOpenedAt *time.Time `gorm:"type:datetime(3)" json:"last_opened_at"`
	CreatedAt    time.Time  `gorm:"type:datetime(3)" json:"created_at"`
}

type ShareOpen struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID      string    `gorm:"size:26;index" json:"share_id"`
	OpenerUserID *string   `gorm:"size:26" json:"opener_user_id"`
	DeviceID     *string   `gorm:"size:64;index" json:"device_id"`
	Platform     string    `gorm:"size:16" json:"platform"`
	CreatedAt    time.Time `gorm:"type:datetime(3)" json:"created_at"`
}

type Event struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *string    `gorm:"size:26" json:"user_id"`
	Name      string     `gorm:"size:48;index:idx_event_name,priority:1" json:"name"`
	Props     JSON       `gorm:"type:json" json:"props"`
	ClientTS  *time.Time `gorm:"type:datetime(3)" json:"client_ts"`
	CreatedAt time.Time  `gorm:"type:datetime(3);index:idx_event_name,priority:2" json:"created_at"`
}

type AppConfig struct {
	Key         string    `gorm:"primaryKey;size:64;column:key" json:"key"`
	Value       JSON      `gorm:"type:json" json:"value"`
	Description *string   `gorm:"size:255" json:"description"`
	UpdatedBy   *string   `gorm:"size:64" json:"updated_by"`
	UpdatedAt   time.Time `gorm:"type:datetime(3)" json:"updated_at"`
}

type AdminUser struct {
	ID           string     `gorm:"primaryKey;size:26" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:255" json:"-"` // never serialised
	Role         string     `gorm:"size:16;default:admin" json:"role"`
	Status       int8       `gorm:"default:1" json:"status"`
	LastLoginAt  *time.Time `gorm:"type:datetime(3)" json:"last_login_at"`
	CreatedAt    time.Time  `gorm:"type:datetime(3)" json:"created_at"`
}

type AuditLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    string    `gorm:"size:26;index" json:"admin_id"`
	Action     string    `gorm:"size:64" json:"action"`
	TargetType string    `gorm:"size:32" json:"target_type"`
	TargetID   string    `gorm:"size:64" json:"target_id"`
	Before     JSON      `gorm:"type:json" json:"before"`
	After      JSON      `gorm:"type:json" json:"after"`
	IP         string    `gorm:"size:64" json:"ip"`
	CreatedAt  time.Time `gorm:"type:datetime(3)" json:"created_at"`
}

type DailyStat struct {
	Date            time.Time `gorm:"type:date;primaryKey" json:"date"`
	Module          Module    `gorm:"size:16;primaryKey" json:"module"`
	TemplateID      string    `gorm:"size:26;primaryKey;default:''" json:"template_id"`
	SpecID          string    `gorm:"size:26;primaryKey;default:''" json:"spec_id"`
	TasksCreated    uint      `gorm:"default:0" json:"tasks_created"`
	TasksSuccess    uint      `gorm:"default:0" json:"tasks_success"`
	TasksFailed     uint      `gorm:"default:0" json:"tasks_failed"`
	WorksSaved      uint      `gorm:"default:0" json:"works_saved"`
	CreditsConsumed uint      `gorm:"default:0" json:"credits_consumed"`
	CreditsRefunded uint      `gorm:"default:0" json:"credits_refunded"`
	AdClaims        uint      `gorm:"default:0" json:"ad_claims"`
	CostCents       uint      `gorm:"default:0" json:"cost_cents"`
}

// AllModels lists every table for AutoMigrate in dev/tests.
func AllModels() []any {
	return []any{
		&User{}, &UserIdentity{}, &CreditAccount{}, &CreditLedger{}, &AdSession{}, &Photo{},
		&Category{}, &Spec{}, &Template{}, &Collection{}, &CollectionTemplate{}, &Banner{},
		&Task{}, &Work{}, &Favorite{}, &Share{}, &ShareOpen{}, &Event{}, &AppConfig{},
		&AdminUser{}, &AuditLog{}, &DailyStat{},
	}
}
