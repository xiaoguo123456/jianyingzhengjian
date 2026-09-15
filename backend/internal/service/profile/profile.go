// Package profile covers the user's own record: nickname, avatar, privacy consent, deletion.
package profile

import (
	"bytes"
	"context"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/provider/storage"
)

type Service struct {
	db    *gorm.DB
	store storage.ObjectStore
}

func New(db *gorm.DB, store storage.ObjectStore) *Service { return &Service{db: db, store: store} }

func (s *Service) Get(ctx context.Context, userID string) (*domain.User, error) {
	var u domain.User
	if err := s.db.WithContext(ctx).First(&u, "id = ?", userID).Error; err != nil {
		return nil, apperr.NotFound("用户")
	}
	return &u, nil
}

func (s *Service) WorksCount(ctx context.Context, userID string) int64 {
	var n int64
	s.db.WithContext(ctx).Model(&domain.Work{}).Where("user_id = ?", userID).Count(&n)
	return n
}

func (s *Service) AvatarURL(ctx context.Context, u *domain.User) string {
	if u.AvatarKey == nil {
		return ""
	}
	return storage.URLFor(ctx, s.store, *u.AvatarKey, 24*time.Hour)
}

func (s *Service) UpdateNickname(ctx context.Context, userID, nickname string) error {
	if len([]rune(nickname)) > 20 {
		return apperr.BadRequest("昵称过长")
	}
	return s.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{"nickname": nickname, "updated_at": clock.Now()}).Error
}

func (s *Service) UploadAvatar(ctx context.Context, userID string, data []byte, contentType string) (string, error) {
	key := "avatars/" + userID + "/" + idgen.New() + ".jpg"
	if err := s.store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return "", apperr.Internal(err)
	}
	if err := s.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{"avatar_key": key, "updated_at": clock.Now()}).Error; err != nil {
		return "", err
	}
	return storage.URLFor(ctx, s.store, key, 24*time.Hour), nil
}

func (s *Service) AgreePrivacy(ctx context.Context, userID, version string) error {
	now := clock.Now()
	return s.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).
		Updates(map[string]any{"privacy_agreed_at": now, "privacy_version": version, "updated_at": now}).Error
}

// DeleteAccount anonymises the user and removes photos and works (docs/COMPLIANCE.md §2.4).
func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	var photos []domain.Photo
	s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&photos)
	for _, p := range photos {
		_ = s.store.Delete(ctx, p.ObjectKey)
	}
	var works []domain.Work
	s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&works)
	for _, w := range works {
		_ = s.store.Delete(ctx, w.ObjectKey)
		_ = s.store.Delete(ctx, w.ThumbKey)
		if w.AlphaKey != nil {
			_ = s.store.Delete(ctx, *w.AlphaKey)
		}
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("user_id = ?", userID).Delete(&domain.Photo{})
		tx.Where("user_id = ?", userID).Delete(&domain.Work{})
		tx.Where("user_id = ?", userID).Delete(&domain.Favorite{})
		tx.Model(&domain.Share{}).Where("user_id = ?", userID).Update("status", "revoked")
		tx.Where("user_id = ?", userID).Delete(&domain.UserIdentity{})
		return tx.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
			"nickname": nil, "avatar_key": nil, "status": 2, "updated_at": clock.Now(),
		}).Error
	})
}
