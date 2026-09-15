package credit_test

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/service/credit"
	"yingji/backend/internal/testutil"
)

type fixture struct {
	db  *gorm.DB
	svc *credit.Service
	uid string
}

func setup(t *testing.T, overrides map[string]any) *fixture {
	t.Helper()
	db := testutil.DB(t)
	base := map[string]any{"ads_enabled": true, "daily_free_credits": 1, "daily_free_credits_no_ads": 3, "ad_reward_daily_cap": 10, "share_reward_daily_cap": 3, "share_reward_per": 1}
	for k, v := range overrides {
		base[k] = v
	}
	rt := testutil.Runtime(t, db, base)
	uid := idgen.New()
	if err := db.Create(&domain.User{ID: uid, Status: 1, CreatedAt: clock.Now(), UpdatedAt: clock.Now(), LastLoginAt: clock.Now()}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return &fixture{db: db, svc: credit.New(db, rt), uid: uid}
}

func (f *fixture) consume(t *testing.T, n uint, ref string) (*domain.CreditLedger, error) {
	t.Helper()
	var row *domain.CreditLedger
	err := f.db.Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = f.svc.Consume(t.Context(), tx, f.uid, n, "task", ref)
		return err
	})
	return row, err
}

func (f *fixture) refund(t *testing.T, consume *domain.CreditLedger) error {
	t.Helper()
	return f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.svc.Refund(t.Context(), tx, consume)
		return err
	})
}

func (f *fixture) balance(t *testing.T) credit.Balance {
	t.Helper()
	b, err := f.svc.Get(t.Context(), f.uid)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	return b
}

// Row 1: new user, ads enabled, first task -> daily grant applied and consumed from daily.
func TestDailyGrantAndConsume(t *testing.T) {
	f := setup(t, nil)
	if b := f.balance(t); b.DailyFreeRemaining != 1 || b.Total != 1 {
		t.Fatalf("after grant: %+v, want 1 daily", b)
	}
	row, err := f.consume(t, 1, "task-1")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if row.Bucket != domain.BucketDaily || row.Delta != -1 {
		t.Errorf("ledger = bucket %s delta %d, want daily -1", row.Bucket, row.Delta)
	}
	if b := f.balance(t); b.Total != 0 {
		t.Errorf("after consume: %+v, want 0", b)
	}
}

// Row 2: no credits -> NO_CREDITS and no balance change.
func TestConsumeWithoutCredits(t *testing.T) {
	f := setup(t, nil)
	if _, err := f.consume(t, 1, "task-1"); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	_, err := f.consume(t, 1, "task-2")
	if !apperr.Is(err, "NO_CREDITS") {
		t.Fatalf("err = %v, want NO_CREDITS", err)
	}
	if b := f.balance(t); b.Total != 0 {
		t.Errorf("balance changed: %+v", b)
	}
}

// Row 3+4: ad reward grants one bonus credit; a second claim on the same session is rejected upstream,
// but the ledger itself is unique per session.
func TestAdRewardUniquePerSession(t *testing.T) {
	f := setup(t, nil)
	sess := idgen.New()
	err := f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.svc.GrantAdReward(t.Context(), tx, f.uid, sess)
		return err
	})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if b := f.balance(t); b.BonusCredits != 1 || b.AdRewardsToday != 1 {
		t.Fatalf("after reward: %+v, want 1 bonus / 1 today", b)
	}
	// second grant with the same ref must violate the unique key
	err = f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.svc.GrantAdReward(t.Context(), tx, f.uid, sess)
		return err
	})
	if err == nil {
		t.Fatal("duplicate ad reward was accepted")
	}
	if b := f.balance(t); b.BonusCredits != 1 {
		t.Errorf("balance after duplicate: %+v, want 1 bonus", b)
	}
}

// Row 7: the daily ad cap is enforced.
func TestAdRewardCap(t *testing.T) {
	f := setup(t, map[string]any{"ad_reward_daily_cap": 2})
	for i := 0; i < 2; i++ {
		if err := f.db.Transaction(func(tx *gorm.DB) error {
			_, err := f.svc.GrantAdReward(t.Context(), tx, f.uid, idgen.New())
			return err
		}); err != nil {
			t.Fatalf("grant %d: %v", i, err)
		}
	}
	err := f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.svc.GrantAdReward(t.Context(), tx, f.uid, idgen.New())
		return err
	})
	if !apperr.Is(err, "AD_SESSION_INVALID") {
		t.Fatalf("err = %v, want AD_SESSION_INVALID (cap_reached)", err)
	}
}

// Row 10 + 18: consume splits across buckets and the refund restores the same buckets.
func TestConsumeSplitAcrossBucketsAndRefund(t *testing.T) {
	f := setup(t, nil) // 1 daily
	if err := f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.svc.GrantAdReward(t.Context(), tx, f.uid, idgen.New())
		return err
	}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	before := f.balance(t)
	if before.DailyFreeRemaining != 1 || before.BonusCredits != 1 {
		t.Fatalf("setup balance = %+v, want 1/1", before)
	}
	row, err := f.consume(t, 2, "task-2c")
	if err != nil {
		t.Fatalf("consume 2: %v", err)
	}
	if b := f.balance(t); b.DailyFreeRemaining != 0 || b.BonusCredits != 0 {
		t.Fatalf("after consume: %+v, want 0/0", b)
	}
	if err := f.refund(t, row); err != nil {
		t.Fatalf("refund: %v", err)
	}
	after := f.balance(t)
	if after.DailyFreeRemaining != before.DailyFreeRemaining || after.BonusCredits != before.BonusCredits {
		t.Errorf("refund restored %+v, want %+v", after, before)
	}
}

// Row 11: a double refund writes only one ledger row.
func TestRefundIdempotent(t *testing.T) {
	f := setup(t, nil)
	row, err := f.consume(t, 1, "task-r")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := f.refund(t, row); err != nil {
			t.Fatalf("refund %d: %v", i, err)
		}
	}
	var n int64
	f.db.Model(&domain.CreditLedger{}).Where("kind = ? AND ref_id = ?", domain.LedgerRefund, "task-r").Count(&n)
	if n != 1 {
		t.Errorf("refund rows = %d, want 1", n)
	}
	if b := f.balance(t); b.Total != 1 {
		t.Errorf("balance = %+v, want 1", b)
	}
}

// Row 8: consuming twice with the same ref returns the original row and charges once.
func TestConsumeIdempotentPerRef(t *testing.T) {
	f := setup(t, map[string]any{"daily_free_credits": 5})
	first, err := f.consume(t, 1, "task-same")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := f.consume(t, 1, "task-same")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("ids differ: %s vs %s", first.ID, second.ID)
	}
	if b := f.balance(t); b.Total != 4 {
		t.Errorf("balance = %+v, want 4 (charged once)", b)
	}
}

// Row 13: a new day resets the daily balance and the ad counter exactly once.
func TestDailyResetOncePerDay(t *testing.T) {
	f := setup(t, map[string]any{"daily_free_credits": 2})
	f.balance(t)
	// simulate yesterday: drain and backdate the reset date
	if _, err := f.consume(t, 2, "task-drain"); err != nil {
		t.Fatalf("drain: %v", err)
	}
	// backdate both the account and the first grant row so the two grants fall on different days
	yesterday := clock.Today().Add(-24 * time.Hour)
	f.db.Model(&domain.CreditAccount{}).Where("user_id = ?", f.uid).
		Updates(map[string]any{"daily_reset_date": yesterday, "ad_rewards_today": 5})
	f.db.Model(&domain.CreditLedger{}).Where("kind = ?", domain.LedgerDailyGrant).
		Updates(map[string]any{"ref_id": yesterday.Format("2006-01-02"), "created_at": yesterday})
	b := f.balance(t)
	if b.DailyFreeRemaining != 2 || b.AdRewardsToday != 0 {
		t.Fatalf("after rollover: %+v, want 2 daily / 0 ad rewards", b)
	}
	// calling again the same day must not grant more
	if b2 := f.balance(t); b2.DailyFreeRemaining != 2 {
		t.Errorf("second call granted again: %+v", b2)
	}
	var grants int64
	f.db.Model(&domain.CreditLedger{}).Where("kind = ?", domain.LedgerDailyGrant).Count(&grants)
	if grants != 2 {
		t.Errorf("daily_grant rows = %d, want 2 (one per day)", grants)
	}
}

// Row 14: with ads disabled the larger no-ads grant applies.
func TestNoAdsGrant(t *testing.T) {
	f := setup(t, map[string]any{"ads_enabled": false})
	b := f.balance(t)
	if b.DailyFreeRemaining != 3 {
		t.Errorf("daily = %d, want 3 (no-ads grant)", b.DailyFreeRemaining)
	}
	if b.AdsEnabled {
		t.Error("AdsEnabled should be false")
	}
}

// A refund issued after the day rolls over must not inflate the next day's free allowance.
func TestRefundAfterRolloverGoesToBonus(t *testing.T) {
	f := setup(t, map[string]any{"daily_free_credits": 1})
	row, err := f.consume(t, 1, "task-cross")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	// pretend the consume happened yesterday and the account has since reset
	f.db.Model(&domain.CreditLedger{}).Where("id = ?", row.ID).Update("created_at", clock.Now().Add(-24*time.Hour))
	row.CreatedAt = clock.Now().Add(-24 * time.Hour)
	if err := f.refund(t, row); err != nil {
		t.Fatalf("refund: %v", err)
	}
	b := f.balance(t)
	if b.DailyFreeRemaining != 0 || b.BonusCredits != 1 {
		t.Errorf("balance = %+v, want 0 daily / 1 bonus", b)
	}
}

// Row 21 + 23: a share reward is granted once per acquired user and capped per day.
func TestShareRewardOncePerUserAndCapped(t *testing.T) {
	f := setup(t, map[string]any{"share_reward_daily_cap": 2})
	acquired := idgen.New()
	grant := func(acq string) bool {
		var granted bool
		err := f.db.Transaction(func(tx *gorm.DB) error {
			var err error
			granted, err = f.svc.GrantShareReward(t.Context(), tx, f.uid, acq)
			return err
		})
		if err != nil {
			t.Fatalf("share reward: %v", err)
		}
		return granted
	}
	if !grant(acquired) {
		t.Fatal("first reward not granted")
	}
	if grant(acquired) {
		t.Error("second reward for the same acquired user was granted")
	}
	if !grant(idgen.New()) {
		t.Error("reward for a different acquired user was not granted")
	}
	if grant(idgen.New()) {
		t.Error("reward granted beyond the daily cap")
	}
}

func TestAdminAdjust(t *testing.T) {
	f := setup(t, nil)
	if _, err := f.svc.AdminAdjust(t.Context(), f.uid, 5, "compensation", idgen.New()); err != nil {
		t.Fatalf("adjust +5: %v", err)
	}
	if b := f.balance(t); b.BonusCredits != 5 {
		t.Fatalf("balance = %+v, want 5 bonus", b)
	}
	if _, err := f.svc.AdminAdjust(t.Context(), f.uid, -2, "correction", idgen.New()); err != nil {
		t.Fatalf("adjust -2: %v", err)
	}
	if b := f.balance(t); b.BonusCredits != 3 {
		t.Errorf("balance = %+v, want 3 bonus", b)
	}
	if _, err := f.svc.AdminAdjust(t.Context(), f.uid, -99, "too much", idgen.New()); err == nil {
		t.Error("over-deduction was accepted")
	}
}

var _ = config.Defaults
