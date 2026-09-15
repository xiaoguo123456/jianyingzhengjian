// Package ads implements rewarded-video sessions and claims (docs/GENERATION_PIPELINE.md §4).
package ads

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/service/credit"
)

type Service struct {
	db       *gorm.DB
	cfg      *config.Runtime
	credit   *credit.Service
	adUnitID string
}

func New(db *gorm.DB, cfg *config.Runtime, cr *credit.Service, adUnitID string) *Service {
	return &Service{db: db, cfg: cfg, credit: cr, adUnitID: adUnitID}
}

func (s *Service) CreateSession(ctx context.Context, userID string) (*domain.AdSession, error) {
	if !s.cfg.Bool(ctx, "ads_enabled") {
		return nil, apperr.Conflict("广告未开通")
	}
	bal, err := s.credit.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if bal.AdRewardsToday >= bal.AdRewardDailyCap {
		return nil, apperr.AdSessionInvalid("cap_reached")
	}
	sess := &domain.AdSession{ID: idgen.New(), UserID: userID, AdUnitID: s.adUnitID, Status: domain.AdPending, CreatedAt: clock.Now()}
	if err := s.db.WithContext(ctx).Create(sess).Error; err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Service) ExpiresAt(ctx context.Context, sess *domain.AdSession) time.Time {
	return sess.CreatedAt.Add(time.Duration(s.cfg.Int(ctx, "ad_session_ttl_minutes")) * time.Minute)
}

// Claim validates the session rules and grants one bonus credit.
func (s *Service) Claim(ctx context.Context, userID, sessionID string, isEnded bool) (int, error) {
	granted := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sess domain.AdSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sess, "id = ? AND user_id = ?", sessionID, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperr.NotFound("广告会话")
			}
			return err
		}
		reject := func(reason string) error {
			sess.Status = domain.AdRejected
			sess.RejectReason = &reason
			sess.ClientIsEnded = &isEnded
			tx.Save(&sess)
			return apperr.AdSessionInvalid(reason)
		}
		now := clock.Now()
		if sess.Status != domain.AdPending {
			return apperr.AdSessionInvalid("duplicate")
		}
		if now.Sub(sess.CreatedAt) < time.Duration(s.cfg.Int(ctx, "ad_session_min_seconds"))*time.Second {
			return reject("too_fast")
		}
		if now.After(s.ExpiresAt(ctx, &sess)) {
			return reject("expired")
		}
		if !isEnded {
			return reject("not_ended")
		}
		if _, err := s.credit.GrantAdReward(ctx, tx, userID, sess.ID); err != nil {
			if apperr.Is(err, "AD_SESSION_INVALID") {
				return reject("cap_reached")
			}
			return err
		}
		sess.Status = domain.AdClaimed
		sess.ClientIsEnded = &isEnded
		sess.ClaimedAt = &now
		granted = 1
		return tx.Save(&sess).Error
	})
	return granted, err
}

// ServerVerified marks the most recent pending/claimed session of the user as verified by WeChat's callback.
func (s *Service) ServerVerified(ctx context.Context, userID, transID string) error {
	var sess domain.AdSession
	err := s.db.WithContext(ctx).Where("user_id = ? AND status IN ('pending','claimed') AND created_at > ?", userID, clock.Now().Add(-time.Hour)).
		Order("created_at DESC").First(&sess).Error
	if err != nil {
		return err
	}
	now := clock.Now()
	return s.db.WithContext(ctx).Model(&sess).Updates(map[string]any{"server_verified_at": now, "trans_id": transID}).Error
}
