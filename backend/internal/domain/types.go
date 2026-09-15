// Package domain holds the persistent entities and enums (docs/DATA_MODEL.md).
package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSON is a raw JSON column.
type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "null", nil
	}
	return string(j), nil
}

func (j *JSON) Scan(value any) error {
	if value == nil {
		*j = JSON("null")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = JSON(append([]byte{}, v...))
	case string:
		*j = JSON(v)
	default:
		return fmt.Errorf("domain.JSON: unsupported type %T", value)
	}
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("domain.JSON: nil receiver")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// MustJSON marshals v or panics; for constants and seeds.
func MustJSON(v any) JSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return JSON(b)
}

func (j JSON) Into(out any) error {
	if len(j) == 0 {
		return nil
	}
	return json.Unmarshal(j, out)
}

type Module string

const (
	ModuleIDPhoto  Module = "idphoto"
	ModulePro      Module = "pro"
	ModulePortrait Module = "portrait"
	ModuleAvatar   Module = "avatar"
)

var ModuleNames = map[Module]string{
	ModuleIDPhoto: "证件照", ModulePro: "职业照", ModulePortrait: "写真", ModuleAvatar: "头像",
}

func (m Module) Valid() bool { _, ok := ModuleNames[m]; return ok }

type TaskKind string

const (
	TaskKindIDPhoto  TaskKind = "idphoto"
	TaskKindTemplate TaskKind = "template"
)

type TaskStatus string

const (
	TaskWaiting    TaskStatus = "waiting"
	TaskProcessing TaskStatus = "processing"
	TaskSuccess    TaskStatus = "success"
	TaskFailed     TaskStatus = "failed"
)

type TaskStage string

const (
	StageQueued     TaskStage = "queued"
	StageProcessing TaskStage = "processing"
	StageFinishing  TaskStage = "finishing"
)

type LedgerKind string

const (
	LedgerDailyGrant  LedgerKind = "daily_grant"
	LedgerAdReward    LedgerKind = "ad_reward"
	LedgerConsume     LedgerKind = "consume"
	LedgerRefund      LedgerKind = "refund"
	LedgerShareReward LedgerKind = "share_reward"
	LedgerAdminAdjust LedgerKind = "admin_adjust"
)

type Bucket string

const (
	BucketDaily Bucket = "daily"
	BucketBonus Bucket = "bonus"
)

type AdSessionStatus string

const (
	AdPending  AdSessionStatus = "pending"
	AdClaimed  AdSessionStatus = "claimed"
	AdRejected AdSessionStatus = "rejected"
	AdExpired  AdSessionStatus = "expired"
)

type ShareType string

const (
	ShareTemplate ShareType = "template"
	ShareWork     ShareType = "work"
	SharePoster   ShareType = "poster"
	ShareTab      ShareType = "tab"
)

type CheckStatus string

const (
	CheckPending  CheckStatus = "pending"
	CheckPassed   CheckStatus = "passed"
	CheckRejected CheckStatus = "rejected"
)

type ModerationStatus string

const (
	ModPending ModerationStatus = "pending"
	ModPass    ModerationStatus = "pass"
	ModRisky   ModerationStatus = "risky"
)

const (
	StatusOnline  int8 = 1
	StatusOffline int8 = 0
)

// Error codes used in tasks.error_code.
const (
	ErrTimeout          = "TIMEOUT"
	ErrProvider         = "PROVIDER_ERROR"
	ErrContentRejected  = "CONTENT_REJECTED"
	ErrIdentityMismatch = "IDENTITY_MISMATCH"
	ErrNoFace           = "NO_FACE"
	ErrVision           = "VISION_ERROR"
	ErrStorage          = "STORAGE_ERROR"
)

// TaskErrorMessages are the user-facing messages (docs/GENERATION_PIPELINE.md §6).
var TaskErrorMessages = map[string]string{
	ErrTimeout:          "本次生成失败，生成次数已返还，请重新尝试。",
	ErrProvider:         "本次生成失败，生成次数已返还，请重新尝试。",
	ErrContentRejected:  "这张照片无法生成，请换一张照片。",
	ErrIdentityMismatch: "生成结果与本人差异较大，请换一张正脸照片。",
	ErrNoFace:           "未检测到清晰人脸，请换一张照片。",
	ErrVision:           "本次生成失败，生成次数已返还，请重新尝试。",
	ErrStorage:          "本次生成失败，生成次数已返还，请重新尝试。",
}
