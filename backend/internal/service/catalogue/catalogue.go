// Package catalogue serves specs, templates, collections and the per-tab home payload.
package catalogue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
)

type Service struct {
	db  *gorm.DB
	rdb *redis.Client
}

func New(db *gorm.DB, rdb *redis.Client) *Service { return &Service{db: db, rdb: rdb} }

// Rail is a horizontally scrolling row of templates on a tab home (docs/API.md GET /v1/home).
// V1 derives rails from the hot categories; an admin-curated rails table can replace this later.
type Rail struct {
	Title        string
	CategoryID   string
	CollectionID string
	Templates    []domain.Template
}

const (
	homeHotTemplates = 6 // one rail on the tab
	homeRails        = 2 // secondary rails
	railTemplates    = 6
)

type HomeData struct {
	Banner        *domain.Banner
	HotSpecs      []domain.Spec
	MoreSpecs     []domain.Spec
	HotCategories []domain.Category
	Collections   []domain.Collection
	HotTemplates  []domain.Template
	Rails         []Rail
}

func (s *Service) Home(ctx context.Context, m domain.Module) (*HomeData, error) {
	key := "home:" + string(m)
	if s.rdb != nil {
		if b, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var h HomeData
			if json.Unmarshal(b, &h) == nil {
				return &h, nil
			}
		}
	}
	h := &HomeData{}
	now := time.Now().UTC()
	var banners []domain.Banner
	s.db.WithContext(ctx).Where("module = ? AND status = 1 AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", m, now, now).
		Order("sort ASC").Limit(1).Find(&banners)
	if len(banners) > 0 {
		h.Banner = &banners[0]
	}
	if m == domain.ModuleIDPhoto {
		s.db.WithContext(ctx).Where("status = 1 AND is_hot = 1").Order("sort ASC").Limit(4).Find(&h.HotSpecs)
		s.db.WithContext(ctx).Where("status = 1 AND is_hot = 0").Order("sort ASC").Limit(4).Find(&h.MoreSpecs)
	} else {
		s.db.WithContext(ctx).Where("module = ? AND kind = 'template' AND status = 1", m).Order("sort ASC").Limit(4).Find(&h.HotCategories)
		if m == domain.ModulePortrait {
			s.db.WithContext(ctx).Where("module = ? AND status = 1", m).Order("sort ASC").Limit(4).Find(&h.Collections)
		}
		s.db.WithContext(ctx).Where("module = ? AND status = 1", m).Order("is_hot DESC, sort ASC").Limit(homeHotTemplates).Find(&h.HotTemplates)
		for _, ct := range h.HotCategories {
			if len(h.Rails) >= homeRails {
				break
			}
			var ts []domain.Template
			s.db.WithContext(ctx).Where("module = ? AND category_id = ? AND status = 1", m, ct.ID).Order("is_hot DESC, sort ASC").Limit(railTemplates).Find(&ts)
			if len(ts) >= 2 {
				h.Rails = append(h.Rails, Rail{Title: ct.Name, CategoryID: ct.ID, Templates: ts})
			}
		}
	}
	if s.rdb != nil {
		if b, err := json.Marshal(h); err == nil {
			_ = s.rdb.Set(ctx, key, b, 60*time.Second).Err()
		}
	}
	return h, nil
}

// InvalidateHome drops the cached home payloads (admin writes).
func (s *Service) InvalidateHome(ctx context.Context) {
	if s.rdb == nil {
		return
	}
	for m := range domain.ModuleNames {
		_ = s.rdb.Del(ctx, "home:"+string(m)).Err()
	}
}

func (s *Service) SpecCategories(ctx context.Context) ([]domain.Category, error) {
	var cats []domain.Category
	err := s.db.WithContext(ctx).Where("kind = 'spec' AND status = 1").Order("sort ASC").Find(&cats).Error
	return cats, err
}

func (s *Service) Specs(ctx context.Context, categoryID string, hot bool) ([]domain.Spec, error) {
	q := s.db.WithContext(ctx).Where("status = 1")
	if categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}
	if hot {
		q = q.Where("is_hot = 1")
	}
	var specs []domain.Spec
	err := q.Order("sort ASC").Find(&specs).Error
	return specs, err
}

func (s *Service) Spec(ctx context.Context, id string) (*domain.Spec, error) {
	var sp domain.Spec
	if err := s.db.WithContext(ctx).First(&sp, "id = ? AND status = 1", id).Error; err != nil {
		return nil, apperr.NotFound("规格")
	}
	return &sp, nil
}

func (s *Service) CategoriesByIDs(ctx context.Context, ids []string) map[string]domain.Category {
	out := map[string]domain.Category{}
	if len(ids) == 0 {
		return out
	}
	var cats []domain.Category
	s.db.WithContext(ctx).Where("id IN ?", ids).Find(&cats)
	for _, c := range cats {
		out[c.ID] = c
	}
	return out
}

func (s *Service) TemplateCategories(ctx context.Context, m domain.Module) ([]domain.Category, error) {
	var cats []domain.Category
	err := s.db.WithContext(ctx).Where("module = ? AND kind = 'template' AND status = 1", m).Order("sort ASC").Find(&cats).Error
	return cats, err
}

type TemplateQuery struct {
	Module       domain.Module
	CategoryID   string
	CollectionID string
	Hot          bool
	Page         int
	PageSize     int
}

func (s *Service) Templates(ctx context.Context, q TemplateQuery) ([]domain.Template, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	db := s.db.WithContext(ctx).Model(&domain.Template{}).Where("templates.status = 1")
	if q.Module != "" {
		db = db.Where("templates.module = ?", q.Module)
	}
	if q.CategoryID != "" {
		db = db.Where("templates.category_id = ?", q.CategoryID)
	}
	if q.Hot {
		db = db.Where("templates.is_hot = 1")
	}
	if q.CollectionID != "" {
		db = db.Joins("JOIN collection_templates ct ON ct.template_id = templates.id AND ct.collection_id = ?", q.CollectionID).Order("ct.sort ASC")
	} else {
		db = db.Order("templates.is_hot DESC, templates.sort ASC")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []domain.Template
	err := db.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}

func (s *Service) Template(ctx context.Context, id string) (*domain.Template, error) {
	var t domain.Template
	if err := s.db.WithContext(ctx).First(&t, "id = ? AND status = 1", id).Error; err != nil {
		return nil, apperr.NotFound("模板")
	}
	return &t, nil
}

func (s *Service) TemplatesByIDs(ctx context.Context, ids []string) map[string]domain.Template {
	out := map[string]domain.Template{}
	if len(ids) == 0 {
		return out
	}
	var rows []domain.Template
	s.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows)
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

func (s *Service) SpecsByIDs(ctx context.Context, ids []string) map[string]domain.Spec {
	out := map[string]domain.Spec{}
	if len(ids) == 0 {
		return out
	}
	var rows []domain.Spec
	s.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows)
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

func (s *Service) Collections(ctx context.Context, m domain.Module) ([]domain.Collection, error) {
	var cols []domain.Collection
	err := s.db.WithContext(ctx).Where("module = ? AND status = 1", m).Order("sort ASC").Find(&cols).Error
	return cols, err
}

func (s *Service) CollectionTemplateCounts(ctx context.Context, ids []string) map[string]int64 {
	type row struct {
		CollectionID string
		N            int64
	}
	var rows []row
	out := map[string]int64{}
	if len(ids) == 0 {
		return out
	}
	s.db.WithContext(ctx).Table("collection_templates").Select("collection_id, COUNT(*) AS n").Where("collection_id IN ?", ids).Group("collection_id").Scan(&rows)
	for _, r := range rows {
		out[r.CollectionID] = r.N
	}
	return out
}

func (s *Service) Collection(ctx context.Context, id string) (*domain.Collection, []domain.Template, error) {
	var c domain.Collection
	if err := s.db.WithContext(ctx).First(&c, "id = ? AND status = 1", id).Error; err != nil {
		return nil, nil, apperr.NotFound("专题")
	}
	rows, _, err := s.Templates(ctx, TemplateQuery{CollectionID: id, PageSize: 100})
	return &c, rows, err
}

// FavoritedSet returns which of the template ids the user favourited.
func (s *Service) FavoritedSet(ctx context.Context, userID string, ids []string) map[string]bool {
	out := map[string]bool{}
	if userID == "" || len(ids) == 0 {
		return out
	}
	var favs []domain.Favorite
	s.db.WithContext(ctx).Where("user_id = ? AND template_id IN ?", userID, ids).Find(&favs)
	for _, f := range favs {
		out[f.TemplateID] = true
	}
	return out
}
