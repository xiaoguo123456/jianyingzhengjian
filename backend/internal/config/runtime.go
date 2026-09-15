package config

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Defaults for runtime configuration keys (docs/BACKEND_ARCHITECTURE.md §8).
var Defaults = map[string]any{
	"ads_enabled":               false,
	"daily_free_credits":        1,
	"daily_free_credits_no_ads": 3,
	"ad_reward_daily_cap":       10,
	"ad_session_min_seconds":    10,
	"ad_session_ttl_minutes":    30,
	"task_timeout_seconds":      300,
	"photo_retention_days":      30,
	"default_provider":          "mock",
	"provider_prices":           map[string]int{},
	"identity_threshold":        0.75,
	"share_reward_enabled":      true,
	"share_reward_daily_cap":    3,
	"share_preview_ttl_days":    90,
	"share_show_nickname":       false,
	"upload_max_bytes":          10485760,
	"photo_min_side_px":         600,
	"face_min_ratio":            0.08,
	"blur_min_var":              25.0,
	"dark_min_luma":             60.0,
	"share_reward_per":          1,
	"poster_slogan":             "上传自拍，生成可以直接用的照片",
}

type appConfigRow struct {
	Key   string          `gorm:"column:key;primaryKey"`
	Value json.RawMessage `gorm:"column:value"`
}

func (appConfigRow) TableName() string { return "app_configs" }

type cached struct {
	raw json.RawMessage
	at  time.Time
}

// Runtime reads app_configs with a two-level cache: in-process (10 s) and Redis (60 s).
type Runtime struct {
	db  *gorm.DB
	rdb *redis.Client
	mu  sync.Mutex
	mem map[string]cached
}

func NewRuntime(db *gorm.DB, rdb *redis.Client) *Runtime {
	return &Runtime{db: db, rdb: rdb, mem: map[string]cached{}}
}

const memTTL = 10 * time.Second
const redisTTL = 60 * time.Second

// Get returns the raw JSON value for key, falling back to Defaults.
func (r *Runtime) Get(ctx context.Context, key string) json.RawMessage {
	r.mu.Lock()
	if c, ok := r.mem[key]; ok && time.Since(c.at) < memTTL {
		r.mu.Unlock()
		return c.raw
	}
	r.mu.Unlock()

	var raw json.RawMessage
	if r.rdb != nil {
		if v, err := r.rdb.Get(ctx, "cfg:"+key).Bytes(); err == nil {
			raw = v
		}
	}
	if raw == nil {
		var row appConfigRow
		err := r.db.WithContext(ctx).First(&row, "`key` = ?", key).Error
		switch {
		case err == nil:
			raw = row.Value
		case errors.Is(err, gorm.ErrRecordNotFound):
			raw, _ = json.Marshal(Defaults[key])
		default:
			raw, _ = json.Marshal(Defaults[key])
		}
		if r.rdb != nil {
			_ = r.rdb.Set(ctx, "cfg:"+key, []byte(raw), redisTTL).Err()
		}
	}
	r.mu.Lock()
	r.mem[key] = cached{raw: raw, at: time.Now()}
	r.mu.Unlock()
	return raw
}

func (r *Runtime) Bool(ctx context.Context, key string) bool {
	var v bool
	_ = json.Unmarshal(r.Get(ctx, key), &v)
	return v
}

func (r *Runtime) Int(ctx context.Context, key string) int {
	var v float64
	_ = json.Unmarshal(r.Get(ctx, key), &v)
	return int(v)
}

func (r *Runtime) Float(ctx context.Context, key string) float64 {
	var v float64
	_ = json.Unmarshal(r.Get(ctx, key), &v)
	return v
}

func (r *Runtime) String(ctx context.Context, key string) string {
	var v string
	_ = json.Unmarshal(r.Get(ctx, key), &v)
	return v
}

func (r *Runtime) JSON(ctx context.Context, key string, out any) error {
	return json.Unmarshal(r.Get(ctx, key), out)
}

// Set writes a value and invalidates caches.
func (r *Runtime) Set(ctx context.Context, key string, value any, by string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = r.db.WithContext(ctx).Exec(
		"INSERT INTO app_configs (`key`, `value`, updated_by, updated_at) VALUES (?, ?, ?, NOW(3)) ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), updated_by = VALUES(updated_by), updated_at = NOW(3)",
		key, json.RawMessage(raw), by,
	).Error
	if err != nil {
		return err
	}
	r.mu.Lock()
	delete(r.mem, key)
	r.mu.Unlock()
	if r.rdb != nil {
		_ = r.rdb.Del(ctx, "cfg:"+key).Err()
	}
	return nil
}

// All returns every key with its effective value (defaults merged).
func (r *Runtime) All(ctx context.Context) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for k := range Defaults {
		out[k] = r.Get(ctx, k)
	}
	return out
}
