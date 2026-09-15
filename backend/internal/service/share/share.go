// Package share implements share creation, landing, attribution, rewards and posters (docs/SHARING.md).
package share

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/engine/local"
	"yingji/backend/internal/pipeline/poster"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/provider/storage"
	"yingji/backend/internal/provider/wechat"
	"yingji/backend/internal/service/catalogue"
	"yingji/backend/internal/service/credit"
	"yingji/backend/internal/service/event"
	"yingji/backend/internal/service/work"
)

type Service struct {
	db        *gorm.DB
	cfg       *config.Runtime
	store     storage.ObjectStore
	wx        *wechat.Client
	catalogue *catalogue.Service
	work      *work.Service
	credit    *credit.Service
	events    *event.Service
	fontPath  string
	log       *slog.Logger
}

func New(db *gorm.DB, cfg *config.Runtime, store storage.ObjectStore, wx *wechat.Client, cat *catalogue.Service, wk *work.Service, cr *credit.Service, ev *event.Service, fontPath string, log *slog.Logger) *Service {
	return &Service{db: db, cfg: cfg, store: store, wx: wx, catalogue: cat, work: wk, credit: cr, events: ev, fontPath: fontPath, log: log}
}

type CreateInput struct {
	Type       domain.ShareType
	TemplateID string
	WorkID     string
	Module     domain.Module
	Surface    string
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*domain.Share, error) {
	sh := &domain.Share{ID: idgen.New(), UserID: userID, Type: in.Type, Module: in.Module, Status: "active", CreatedAt: clock.Now()}
	switch in.Type {
	case domain.ShareTemplate:
		t, err := s.catalogue.Template(ctx, in.TemplateID)
		if err != nil {
			return nil, err
		}
		sh.Module = t.Module
		sh.TemplateID = &t.ID
		sh.Title = fmt.Sprintf("「%s」上传照片就能生成同款", t.Name)
		sh.PreviewKey = &t.CoverKey
		sh.Path = fmt.Sprintf("/pages/template-detail/index?id=%s&s=%s", t.ID, sh.ID)
	case domain.ShareWork, domain.SharePoster:
		w, err := s.work.Get(ctx, userID, in.WorkID)
		if err != nil {
			return nil, err
		}
		sh.Module = w.Module
		sh.WorkID = &w.ID
		sh.TemplateID = w.TemplateID
		sh.SpecID = w.SpecID
		tpls, specs := s.work.Names(ctx, []domain.Work{*w})
		name := ""
		if w.TemplateID != nil {
			name = tpls[*w.TemplateID].Name
		} else if w.SpecID != nil {
			name = specs[*w.SpecID].Name
		}
		sh.Title = fmt.Sprintf("我用「%s」生成了这张，试试同款", name)
		if w.TemplateID != nil {
			sh.Path = fmt.Sprintf("/pages/template-detail/index?id=%s&s=%s", *w.TemplateID, sh.ID)
		} else {
			sh.Path = fmt.Sprintf("/pages/spec-library/index?spec=%s&s=%s", derefStr(w.SpecID), sh.ID)
		}
		// public copy of the thumbnail with the explicit label burned in
		raw, err := s.work.Bytes(ctx, w.ThumbKey)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		img, _, err := local.Decode(raw)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		card := local.Fill(img, 1000, 800)
		local.DrawBadge(card, s.fontPath)
		jpg, _ := local.EncodeJPEG(card, 88)
		key := "shares/" + sh.ID + ".jpg"
		if err := s.store.Put(ctx, key, bytes.NewReader(jpg), int64(len(jpg)), "image/jpeg"); err != nil {
			return nil, apperr.Internal(err)
		}
		sh.PreviewKey = &key
		if in.Type == domain.SharePoster {
			pk, err := s.renderPoster(ctx, sh, w, img, name)
			if err != nil {
				s.log.Warn("poster render failed", "share", sh.ID, "err", err)
			} else {
				sh.PosterKey = &pk
			}
		}
	case domain.ShareTab:
		if !in.Module.Valid() {
			in.Module = domain.ModuleIDPhoto
		}
		sh.Module = in.Module
		sh.Title = "上传自拍，生成" + domain.ModuleNames[in.Module]
		// 头像 is a segment of the 写真 tab (docs/DECISIONS.md D-24)
		if in.Module == domain.ModuleAvatar {
			sh.Path = fmt.Sprintf("/pages/portrait/index?seg=avatar&s=%s", sh.ID)
		} else {
			sh.Path = fmt.Sprintf("/pages/%s/index?s=%s", in.Module, sh.ID)
		}
		h, _ := s.catalogue.Home(ctx, in.Module)
		if h != nil && h.Banner != nil {
			sh.PreviewKey = &h.Banner.ImageKey
		}
	default:
		return nil, apperr.BadRequest("unknown share type")
	}
	if err := s.db.WithContext(ctx).Create(sh).Error; err != nil {
		return nil, err
	}
	uid := userID
	s.events.Server(ctx, &uid, "share_created", map[string]any{"share_id": sh.ID, "type": in.Type, "surface": in.Surface, "module": sh.Module})
	return sh, nil
}

func (s *Service) renderPoster(ctx context.Context, sh *domain.Share, w *domain.Work, img image.Image, name string) (string, error) {
	var qr image.Image
	if s.wx.Configured() {
		if data, err := s.wx.GetUnlimitedQRCode(ctx, "s="+sh.ID, "pages/template-detail/index", 430); err == nil {
			if q, err := png.Decode(bytes.NewReader(data)); err == nil {
				qr = q
			} else if q, _, err := image.Decode(bytes.NewReader(data)); err == nil {
				qr = q
			}
		}
	}
	full, err := s.work.Bytes(ctx, w.ObjectKey)
	if err == nil {
		if fimg, _, err := local.Decode(full); err == nil {
			img = fimg
		}
	}
	out := poster.Render(poster.Input{Image: img, Square: w.Module == domain.ModuleAvatar, Title: name,
		Subtitle: s.cfg.String(ctx, "poster_slogan"), Brand: "映己证件照写真馆", QR: qr, FontPath: s.fontPath})
	jpg, err := local.EncodeJPEG(out, 90)
	if err != nil {
		return "", err
	}
	key := "shares/" + sh.ID + "-poster.jpg"
	if err := s.store.Put(ctx, key, bytes.NewReader(jpg), int64(len(jpg)), "image/jpeg"); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Share, error) {
	var sh domain.Share
	if err := s.db.WithContext(ctx).First(&sh, "id = ?", id).Error; err != nil {
		return nil, apperr.NotFound("分享")
	}
	return &sh, nil
}

func (s *Service) PreviewURL(sh *domain.Share) string {
	if sh.PreviewKey == nil {
		return ""
	}
	return storage.URLFor(context.Background(), s.store, *sh.PreviewKey, 24*time.Hour)
}

func (s *Service) PosterURL(sh *domain.Share) string {
	if sh.PosterKey == nil {
		return ""
	}
	return storage.URLFor(context.Background(), s.store, *sh.PosterKey, 24*time.Hour)
}

// Open records a landing and returns the share for the hero (preview only while active).
func (s *Service) Open(ctx context.Context, id, platform, deviceID string, userID string) (*domain.Share, error) {
	sh, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	now := clock.Now()
	open := domain.ShareOpen{ShareID: sh.ID, Platform: platform, CreatedAt: now}
	if deviceID != "" {
		open.DeviceID = &deviceID
	}
	if userID != "" {
		open.OpenerUserID = &userID
	}
	_ = s.db.WithContext(ctx).Create(&open).Error
	_ = s.db.WithContext(ctx).Model(sh).Updates(map[string]any{"opens": gorm.Expr("opens + 1"), "last_opened_at": now}).Error
	s.events.Server(ctx, nil, "share_open", map[string]any{"share_id": sh.ID, "platform": platform})
	return sh, nil
}

func (s *Service) ListMine(ctx context.Context, userID string, page, size int) ([]domain.Share, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&domain.Share{}).Where("user_id = ?", userID)
	var total int64
	q.Count(&total)
	var rows []domain.Share
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (s *Service) Revoke(ctx context.Context, userID, id string) error {
	sh, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if sh.UserID != userID {
		return apperr.Forbidden()
	}
	return s.revoke(ctx, sh)
}

func (s *Service) revoke(ctx context.Context, sh *domain.Share) error {
	if sh.Type == domain.ShareWork || sh.Type == domain.SharePoster {
		if sh.PreviewKey != nil {
			_ = s.store.Delete(ctx, *sh.PreviewKey)
		}
		if sh.PosterKey != nil {
			_ = s.store.Delete(ctx, *sh.PosterKey)
		}
	}
	return s.db.WithContext(ctx).Model(sh).Updates(map[string]any{"status": "revoked", "preview_key": nil, "poster_key": nil}).Error
}

// RevokeByWork revokes every share referencing a deleted work.
func (s *Service) RevokeByWork(ctx context.Context, workID string) {
	var rows []domain.Share
	s.db.WithContext(ctx).Where("work_id = ? AND status = 'active'", workID).Find(&rows)
	for i := range rows {
		_ = s.revoke(ctx, &rows[i])
	}
}

type Rewards struct {
	Enabled     bool `json:"enabled"`
	PerReward   int  `json:"per_reward"`
	DailyCap    int  `json:"daily_cap"`
	EarnedToday int  `json:"earned_today"`
	EarnedTotal int  `json:"earned_total"`
}

func (s *Service) Rewards(ctx context.Context, userID string) (Rewards, error) {
	r := Rewards{Enabled: s.cfg.Bool(ctx, "share_reward_enabled"), PerReward: s.cfg.Int(ctx, "share_reward_per"), DailyCap: s.cfg.Int(ctx, "share_reward_daily_cap")}
	if r.PerReward == 0 {
		r.PerReward = 1
	}
	var today, total int64
	s.db.WithContext(ctx).Model(&domain.CreditLedger{}).Where("user_id = ? AND kind = ?", userID, domain.LedgerShareReward).Count(&total)
	s.db.WithContext(ctx).Model(&domain.CreditLedger{}).Where("user_id = ? AND kind = ? AND created_at >= ?", userID, domain.LedgerShareReward, clock.Today()).Count(&today)
	r.EarnedToday, r.EarnedTotal = int(today), int(total)
	return r, nil
}

// GrantForTask rewards the sharer when an acquired user completes their first successful gen task (D-22).
func (s *Service) GrantForTask(ctx context.Context, t *domain.Task) {
	if !t.UsesGenmodel || t.Status != domain.TaskSuccess || !s.cfg.Bool(ctx, "share_reward_enabled") {
		return
	}
	var u domain.User
	if s.db.WithContext(ctx).First(&u, "id = ?", t.UserID).Error != nil || u.AcquiredShareID == nil {
		return
	}
	sh, err := s.Get(ctx, *u.AcquiredShareID)
	if err != nil || sh.UserID == u.ID {
		return
	}
	// self-referral check by shared device ids
	var n int64
	s.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM share_opens o JOIN shares sh ON sh.id = o.share_id
		WHERE o.share_id = ? AND o.device_id IS NOT NULL AND o.device_id IN (SELECT device_id FROM share_opens WHERE opener_user_id = ? AND device_id IS NOT NULL)`, sh.ID, sh.UserID).Scan(&n)
	if n > 0 {
		s.log.Info("share reward skipped: self-referral", "share", sh.ID)
		return
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		granted, err := s.credit.GrantShareReward(ctx, tx, sh.UserID, u.ID)
		if err != nil {
			return err
		}
		if granted {
			sid := sh.UserID
			s.events.Server(ctx, &sid, "share_reward_granted", map[string]any{"share_id": sh.ID, "acquired_user_id": u.ID})
		}
		return nil
	})
	if err != nil {
		s.log.Warn("share reward failed", "share", sh.ID, "err", err)
	}
}

// Cleanup removes previews of shares not opened for share_preview_ttl_days.
func (s *Service) Cleanup(ctx context.Context) (int, error) {
	cutoff := clock.Now().Add(-time.Duration(s.cfg.Int(ctx, "share_preview_ttl_days")) * 24 * time.Hour)
	var rows []domain.Share
	if err := s.db.WithContext(ctx).Where("status = 'active' AND type IN ('work','poster') AND COALESCE(last_opened_at, created_at) < ?", cutoff).Limit(200).Find(&rows).Error; err != nil {
		return 0, err
	}
	for i := range rows {
		_ = s.revoke(ctx, &rows[i])
	}
	return len(rows), nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
