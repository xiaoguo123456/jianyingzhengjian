// Package favorite stores template favourites (templates only, PRD 32).
package favorite

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/clock"
)

type Service struct{ db *gorm.DB }

func New(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) List(ctx context.Context, userID string, page, size int) ([]domain.Template, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&domain.Template{}).
		Joins("JOIN favorites f ON f.template_id = templates.id AND f.user_id = ?", userID).
		Where("templates.status = 1")
	var total int64
	q.Count(&total)
	var rows []domain.Template
	err := q.Order("f.created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (s *Service) Add(ctx context.Context, userID, templateID string) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&domain.Favorite{UserID: userID, TemplateID: templateID, CreatedAt: clock.Now()}).Error
}

func (s *Service) Remove(ctx context.Context, userID, templateID string) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND template_id = ?", userID, templateID).Delete(&domain.Favorite{}).Error
}
