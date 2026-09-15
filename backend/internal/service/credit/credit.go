// Package credit is the transactional core of the credit ledger (docs/BACKEND_ARCHITECTURE.md §5).
package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
)

type Service struct {
	db  *gorm.DB
	cfg *config.Runtime
}

func New(db *gorm.DB, cfg *config.Runtime) *Service { return &Service{db: db, cfg: cfg} }

type Balance struct {
	DailyFreeRemaining uint `json:"daily_free_remaining"`
	BonusCredits       uint `json:"bonus_credits"`
	Total              uint `json:"total"`
	AdRewardsToday     uint `json:"ad_rewards_today"`
	AdRewardDailyCap   uint `json:"ad_reward_daily_cap"`
	AdsEnabled         bool `json:"ads_enabled"`
}

type split struct {
	Daily uint `json:"daily"`
	Bonus uint `json:"bonus"`
}

// Get returns the balance, applying the lazy daily grant.
func (s *Service) Get(ctx context.Context, userID string) (Balance, error) {
	var bal Balance
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acct, err := s.EnsureAccount(ctx, tx, userID)
		if err != nil {
			return err
		}
		bal = s.balance(ctx, acct)
		return nil
	})
	return bal, err
}

func (s *Service) balance(ctx context.Context, a *domain.CreditAccount) Balance {
	return Balance{
		DailyFreeRemaining: a.DailyFreeRemaining, BonusCredits: a.BonusCredits, Total: a.DailyFreeRemaining + a.BonusCredits,
		AdRewardsToday: a.AdRewardsToday, AdRewardDailyCap: uint(s.cfg.Int(ctx, "ad_reward_daily_cap")), AdsEnabled: s.cfg.Bool(ctx, "ads_enabled"),
	}
}

// EnsureAccount locks the account row (creating it if needed) and applies the daily reset.
func (s *Service) EnsureAccount(ctx context.Context, tx *gorm.DB, userID string) (*domain.CreditAccount, error) {
	now := clock.Now()
	var acct domain.CreditAccount
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acct, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		acct = domain.CreditAccount{UserID: userID, DailyResetDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: now}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&acct).Error; err != nil {
			return nil, err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&acct, "user_id = ?", userID).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if !clock.SameDay(acct.DailyResetDate, now) {
		grant := s.cfg.Int(ctx, "daily_free_credits")
		if !s.cfg.Bool(ctx, "ads_enabled") {
			grant = s.cfg.Int(ctx, "daily_free_credits_no_ads")
		}
		acct.DailyFreeRemaining = uint(grant)
		acct.AdRewardsToday = 0
		acct.DailyResetDate = clock.Today()
		acct.UpdatedAt = now
		if err := tx.Save(&acct).Error; err != nil {
			return nil, err
		}
		day := clock.DateString(now)
		_ = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domain.CreditLedger{
			ID: idgen.New(), UserID: userID, Kind: domain.LedgerDailyGrant, Bucket: domain.BucketDaily, Delta: grant,
			BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: "day", RefID: day, CreatedAt: now,
		}).Error
	}
	return &acct, nil
}

// Consume takes n credits (daily first, then bonus). Idempotent per (task) ref.
func (s *Service) Consume(ctx context.Context, tx *gorm.DB, userID string, n uint, refType, refID string) (*domain.CreditLedger, error) {
	var existing domain.CreditLedger
	if err := tx.First(&existing, "kind = ? AND ref_type = ? AND ref_id = ?", domain.LedgerConsume, refType, refID).Error; err == nil {
		return &existing, nil
	}
	acct, err := s.EnsureAccount(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if acct.DailyFreeRemaining+acct.BonusCredits < n {
		return nil, apperr.NoCredits(s.balance(ctx, acct))
	}
	sp := split{}
	sp.Daily = min(acct.DailyFreeRemaining, n)
	sp.Bonus = n - sp.Daily
	acct.DailyFreeRemaining -= sp.Daily
	acct.BonusCredits -= sp.Bonus
	acct.UpdatedAt = clock.Now()
	if err := tx.Save(acct).Error; err != nil {
		return nil, err
	}
	bucket := domain.BucketDaily
	if sp.Bonus > 0 && sp.Daily > 0 {
		bucket = "mixed"
	} else if sp.Bonus > 0 {
		bucket = domain.BucketBonus
	}
	noteB, _ := json.Marshal(sp)
	note := string(noteB)
	row := &domain.CreditLedger{ID: idgen.New(), UserID: userID, Kind: domain.LedgerConsume, Bucket: bucket, Delta: -int(n),
		BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: refType, RefID: refID, Note: &note, CreatedAt: clock.Now()}
	if err := tx.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Refund returns what a consume row took, to the same buckets. Idempotent per ref.
func (s *Service) Refund(ctx context.Context, tx *gorm.DB, consume *domain.CreditLedger) (*domain.CreditLedger, error) {
	var existing domain.CreditLedger
	if err := tx.First(&existing, "kind = ? AND ref_type = ? AND ref_id = ?", domain.LedgerRefund, consume.RefType, consume.RefID).Error; err == nil {
		return &existing, nil
	}
	acct, err := s.EnsureAccount(ctx, tx, consume.UserID)
	if err != nil {
		return nil, err
	}
	var sp split
	if consume.Note != nil {
		_ = json.Unmarshal([]byte(*consume.Note), &sp)
	}
	if sp.Daily+sp.Bonus == 0 {
		sp.Bonus = uint(-consume.Delta)
	}
	// daily credits refunded on a later day would exceed the day's grant; give them as bonus instead
	if !clock.SameDay(acct.DailyResetDate, consume.CreatedAt) {
		sp.Bonus += sp.Daily
		sp.Daily = 0
	}
	acct.DailyFreeRemaining += sp.Daily
	acct.BonusCredits += sp.Bonus
	acct.UpdatedAt = clock.Now()
	if err := tx.Save(acct).Error; err != nil {
		return nil, err
	}
	row := &domain.CreditLedger{ID: idgen.New(), UserID: consume.UserID, Kind: domain.LedgerRefund, Bucket: consume.Bucket, Delta: -consume.Delta,
		BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: consume.RefType, RefID: consume.RefID, CreatedAt: clock.Now()}
	if err := tx.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// GrantAdReward adds one bonus credit for a claimed ad session (unique per session).
func (s *Service) GrantAdReward(ctx context.Context, tx *gorm.DB, userID, sessionID string) (*domain.CreditLedger, error) {
	acct, err := s.EnsureAccount(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if acct.AdRewardsToday >= uint(s.cfg.Int(ctx, "ad_reward_daily_cap")) {
		return nil, apperr.AdSessionInvalid("cap_reached")
	}
	acct.BonusCredits++
	acct.AdRewardsToday++
	acct.UpdatedAt = clock.Now()
	if err := tx.Save(acct).Error; err != nil {
		return nil, err
	}
	row := &domain.CreditLedger{ID: idgen.New(), UserID: userID, Kind: domain.LedgerAdReward, Bucket: domain.BucketBonus, Delta: 1,
		BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: "ad_session", RefID: sessionID, CreatedAt: clock.Now()}
	if err := tx.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// GrantShareReward rewards a sharer once per acquired user, capped per day. Returns granted=false when skipped.
func (s *Service) GrantShareReward(ctx context.Context, tx *gorm.DB, sharerID, acquiredUserID string) (bool, error) {
	var n int64
	tx.Model(&domain.CreditLedger{}).Where("kind = ? AND ref_type = ? AND ref_id = ?", domain.LedgerShareReward, "user", acquiredUserID).Count(&n)
	if n > 0 {
		return false, nil
	}
	dayStart := clock.Today()
	tx.Model(&domain.CreditLedger{}).Where("kind = ? AND user_id = ? AND created_at >= ?", domain.LedgerShareReward, sharerID, dayStart).Count(&n)
	if n >= int64(s.cfg.Int(ctx, "share_reward_daily_cap")) {
		return false, nil
	}
	acct, err := s.EnsureAccount(ctx, tx, sharerID)
	if err != nil {
		return false, err
	}
	per := uint(s.cfg.Int(ctx, "share_reward_per"))
	if per == 0 {
		per = 1
	}
	acct.BonusCredits += per
	acct.UpdatedAt = clock.Now()
	if err := tx.Save(acct).Error; err != nil {
		return false, err
	}
	row := &domain.CreditLedger{ID: idgen.New(), UserID: sharerID, Kind: domain.LedgerShareReward, Bucket: domain.BucketBonus, Delta: int(per),
		BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: "user", RefID: acquiredUserID, CreatedAt: clock.Now()}
	if err := tx.Create(row).Error; err != nil {
		return false, err
	}
	return true, nil
}

// AdminAdjust changes the bonus balance by delta (may be negative) with an audit note.
func (s *Service) AdminAdjust(ctx context.Context, userID string, delta int, note, opID string) (*domain.CreditLedger, error) {
	var row *domain.CreditLedger
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acct, err := s.EnsureAccount(ctx, tx, userID)
		if err != nil {
			return err
		}
		if delta < 0 && acct.BonusCredits < uint(-delta) {
			return apperr.BadRequest("余额不足以扣减")
		}
		acct.BonusCredits = uint(int(acct.BonusCredits) + delta)
		acct.UpdatedAt = clock.Now()
		if err := tx.Save(acct).Error; err != nil {
			return err
		}
		row = &domain.CreditLedger{ID: idgen.New(), UserID: userID, Kind: domain.LedgerAdminAdjust, Bucket: domain.BucketBonus, Delta: delta,
			BalanceDailyAfter: acct.DailyFreeRemaining, BalanceBonusAfter: acct.BonusCredits, RefType: "admin", RefID: opID, Note: &note, CreatedAt: clock.Now()}
		return tx.Create(row).Error
	})
	return row, err
}

// Ledger lists a user's ledger rows, newest first.
func (s *Service) Ledger(ctx context.Context, userID string, limit int) ([]domain.CreditLedger, error) {
	var rows []domain.CreditLedger
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) String() string { return fmt.Sprintf("credit.Service") }
