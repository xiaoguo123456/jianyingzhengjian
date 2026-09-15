// Package event records funnel events (docs/DECISIONS.md D-17).
package event

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/clock"
)

type Service struct{ db *gorm.DB }

func New(db *gorm.DB) *Service { return &Service{db: db} }

var allowed = map[string]bool{}

func init() {
	for _, n := range []string{
		"tab_view", "banner_click", "spec_click", "template_click", "template_detail_view", "upload_start", "upload_success",
		"photo_rejected", "confirm_view", "generate_click", "no_credits", "ad_show", "ad_ended", "ad_abandon", "ad_error",
		"task_created", "task_success", "task_failed", "result_view", "work_saved", "regenerate_click", "recolor_click",
		"share_click", "share_sent", "poster_saved",
	} {
		allowed[n] = true
	}
}

type Input struct {
	Name  string         `json:"name"`
	Props map[string]any `json:"props"`
	TS    string         `json:"ts"`
}

func (s *Service) Record(ctx context.Context, userID string, in []Input) error {
	rows := make([]domain.Event, 0, len(in))
	for _, e := range in {
		if !allowed[e.Name] {
			continue
		}
		row := domain.Event{Name: e.Name, Props: domain.MustJSON(e.Props), CreatedAt: clock.Now()}
		if userID != "" {
			uid := userID
			row.UserID = &uid
		}
		if t, err := time.Parse(time.RFC3339Nano, e.TS); err == nil {
			tt := t.UTC()
			row.ClientTS = &tt
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

// Server records a server-side event.
func (s *Service) Server(ctx context.Context, userID *string, name string, props map[string]any) {
	_ = s.db.WithContext(ctx).Create(&domain.Event{UserID: userID, Name: name, Props: domain.MustJSON(props), CreatedAt: clock.Now()}).Error
}
