// Package work manages generated outputs: listing, download URLs, free recolor, deletion.
package work

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pipeline/idphoto"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/provider/storage"
	"yingji/backend/internal/service/catalogue"
)

type Service struct {
	db        *gorm.DB
	store     storage.ObjectStore
	catalogue *catalogue.Service
}

func New(db *gorm.DB, store storage.ObjectStore, cat *catalogue.Service) *Service {
	return &Service{db: db, store: store, catalogue: cat}
}

const urlTTL = 10 * time.Minute

func (s *Service) URLs(ctx context.Context, w *domain.Work) (string, string) {
	return storage.URLFor(ctx, s.store, w.ObjectKey, urlTTL), storage.URLFor(ctx, s.store, w.ThumbKey, urlTTL)
}

func (s *Service) List(ctx context.Context, userID string, module domain.Module, page, size int) ([]domain.Work, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&domain.Work{}).Where("user_id = ? AND moderation_status <> 'risky'", userID)
	if module != "" {
		q = q.Where("module = ?", module)
	}
	var total int64
	q.Count(&total)
	var rows []domain.Work
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

type Summary struct {
	Total    int64
	ByModule map[domain.Module]int64
	Recent   []domain.Work
}

func (s *Service) Summary(ctx context.Context, userID string) (*Summary, error) {
	out := &Summary{ByModule: map[domain.Module]int64{domain.ModuleIDPhoto: 0, domain.ModulePro: 0, domain.ModulePortrait: 0, domain.ModuleAvatar: 0}}
	type row struct {
		Module domain.Module
		N      int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&domain.Work{}).Select("module, COUNT(*) AS n").Where("user_id = ? AND moderation_status <> 'risky'", userID).Group("module").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out.ByModule[r.Module] = r.N
		out.Total += r.N
	}
	// most recent per module
	for m := range out.ByModule {
		var w domain.Work
		if s.db.WithContext(ctx).Where("user_id = ? AND module = ? AND moderation_status <> 'risky'", userID, m).Order("created_at DESC").First(&w).Error == nil {
			out.Recent = append(out.Recent, w)
		}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*domain.Work, error) {
	var w domain.Work
	if err := s.db.WithContext(ctx).First(&w, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, apperr.NotFound("作品")
	}
	if w.ModerationStatus == domain.ModRisky {
		return nil, apperr.NotFound("作品")
	}
	return &w, nil
}

func (s *Service) Load(ctx context.Context, id string) (*domain.Work, error) {
	var w domain.Work
	if err := s.db.WithContext(ctx).First(&w, "id = ?", id).Error; err != nil {
		return nil, apperr.NotFound("作品")
	}
	return &w, nil
}

func (s *Service) Bytes(ctx context.Context, key string) ([]byte, error) {
	rc, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (s *Service) DownloadURL(ctx context.Context, userID, id string) (string, time.Time, error) {
	w, err := s.Get(ctx, userID, id)
	if err != nil {
		return "", time.Time{}, err
	}
	u, err := s.store.SignedURL(ctx, w.ObjectKey, urlTTL)
	return u, clock.Now().Add(urlTTL), err
}

// Recolor composites the stored alpha over a new background; free and synchronous (D-05).
func (s *Service) Recolor(ctx context.Context, userID, id, bg string) (*domain.Work, error) {
	src, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if src.Module != domain.ModuleIDPhoto || src.AlphaKey == nil || src.SpecID == nil {
		return nil, apperr.BadRequest("该作品不支持换背景")
	}
	sp, err := s.catalogue.Spec(ctx, *src.SpecID)
	if err != nil {
		return nil, err
	}
	var allowed []string
	_ = sp.BgAllowed.Into(&allowed)
	ok := false
	for _, a := range allowed {
		if strings.EqualFold(a, bg) {
			ok = true
			bg = a
		}
	}
	if !ok {
		return nil, apperr.BadRequest("该规格不支持此背景色")
	}
	img, err := s.Bytes(ctx, src.ObjectKey)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	alpha, err := s.Bytes(ctx, *src.AlphaKey)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	data, thumb, err := idphoto.Recolor(img, alpha, bg)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var meta map[string]any
	_ = src.Meta.Into(&meta)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["bg"] = bg
	meta["recolored_from"] = src.ID
	w := &domain.Work{ID: idgen.New(), UserID: userID, Module: src.Module, SpecID: src.SpecID,
		Width: src.Width, Height: src.Height, Meta: domain.MustJSON(meta), AILabel: src.AILabel, ModerationStatus: src.ModerationStatus,
		AlphaKey: src.AlphaKey, CreatedAt: clock.Now()}
	w.ObjectKey = "works/" + userID + "/" + w.ID + ".png"
	w.ThumbKey = "works/" + userID + "/" + w.ID + "_thumb.jpg"
	if err := s.store.Put(ctx, w.ObjectKey, bytes.NewReader(data), int64(len(data)), "image/png"); err != nil {
		return nil, apperr.Internal(err)
	}
	if err := s.store.Put(ctx, w.ThumbKey, bytes.NewReader(thumb), int64(len(thumb)), "image/jpeg"); err != nil {
		return nil, apperr.Internal(err)
	}
	if err := s.db.WithContext(ctx).Create(w).Error; err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	w, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	_ = s.store.Delete(ctx, w.ObjectKey)
	_ = s.store.Delete(ctx, w.ThumbKey)
	return s.db.WithContext(ctx).Delete(w).Error
}

func (s *Service) SetModeration(ctx context.Context, id string, status domain.ModerationStatus) error {
	return s.db.WithContext(ctx).Model(&domain.Work{}).Where("id = ?", id).Update("moderation_status", status).Error
}

// Names resolves template names and spec rows for DTO mapping.
func (s *Service) Names(ctx context.Context, works []domain.Work) (map[string]domain.Template, map[string]domain.Spec) {
	var tids, sids []string
	for _, w := range works {
		if w.TemplateID != nil {
			tids = append(tids, *w.TemplateID)
		}
		if w.SpecID != nil {
			sids = append(sids, *w.SpecID)
		}
	}
	return s.catalogue.TemplatesByIDs(ctx, tids), s.catalogue.SpecsByIDs(ctx, sids)
}
