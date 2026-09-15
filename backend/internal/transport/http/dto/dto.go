// Package dto maps domain rows to the JSON shapes the client expects (client/src/types).
package dto

import (
	"context"
	"fmt"
	"time"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/provider/storage"
)

const assetTTL = 24 * time.Hour

type Paged[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
}

func NewPaged[T any](items []T, total int64, page, size int) Paged[T] {
	if items == nil {
		items = []T{}
	}
	return Paged[T]{Items: items, Total: total, HasMore: int64(page*size) < total}
}

type User struct {
	ID            string `json:"id"`
	Nickname      string `json:"nickname"`
	AvatarURL     string `json:"avatar_url"`
	WorksCount    int64  `json:"works_count"`
	PrivacyAgreed bool   `json:"privacy_agreed"`
}

func UserFrom(u *domain.User, avatarURL string, works int64) User {
	out := User{ID: u.ID, AvatarURL: avatarURL, WorksCount: works, PrivacyAgreed: u.PrivacyAgreedAt != nil}
	if u.Nickname != nil {
		out.Nickname = *u.Nickname
	}
	return out
}

type Spec struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	WidthMM   float64  `json:"width_mm"`
	HeightMM  float64  `json:"height_mm"`
	WidthPx   int      `json:"width_px"`
	HeightPx  int      `json:"height_px"`
	DPI       int      `json:"dpi"`
	BgDefault string   `json:"bg_default"`
	BgAllowed []string `json:"bg_allowed"`
	Category  *IDName  `json:"category,omitempty"`
	Note      string   `json:"note,omitempty"`
	IsHot     bool     `json:"is_hot"`
}

type IDName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func SpecFrom(s domain.Spec, cats map[string]domain.Category) Spec {
	out := Spec{ID: s.ID, Name: s.Name, WidthMM: s.WidthMM, HeightMM: s.HeightMM, WidthPx: s.WidthPx, HeightPx: s.HeightPx, DPI: s.DPI, BgDefault: s.BgDefault, IsHot: s.IsHot}
	_ = s.BgAllowed.Into(&out.BgAllowed)
	if out.BgAllowed == nil {
		out.BgAllowed = []string{s.BgDefault}
	}
	if s.Note != nil {
		out.Note = *s.Note
	}
	if c, ok := cats[s.CategoryID]; ok {
		out.Category = &IDName{ID: c.ID, Name: c.Name}
	}
	return out
}

func SpecsFrom(specs []domain.Spec, cats map[string]domain.Category) []Spec {
	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		out = append(out, SpecFrom(s, cats))
	}
	return out
}

type Category struct {
	ID       string `json:"id"`
	Module   string `json:"module"`
	Name     string `json:"name"`
	Icon     string `json:"icon,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

func CategoryFrom(ctx context.Context, st storage.ObjectStore, c domain.Category) Category {
	out := Category{ID: c.ID, Module: string(c.Module), Name: c.Name}
	if c.Icon != nil {
		out.Icon = *c.Icon
	}
	if c.CoverKey != nil {
		out.CoverURL = storage.URLFor(ctx, st, *c.CoverKey, assetTTL)
	}
	if c.Desc != nil {
		out.Desc = *c.Desc
	}
	return out
}

type TemplateCard struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Name        string   `json:"name"`
	CoverURL    string   `json:"cover_url"`
	Tags        []string `json:"tags"`
	IsFavorited bool     `json:"is_favorited"`
	CreditCost  uint     `json:"credit_cost"`
}

func CardFrom(ctx context.Context, st storage.ObjectStore, t domain.Template, fav bool) TemplateCard {
	out := TemplateCard{ID: t.ID, Module: string(t.Module), Name: t.Name, CoverURL: storage.URLFor(ctx, st, t.CoverKey, assetTTL), IsFavorited: fav, CreditCost: t.CreditCost}
	_ = t.Tags.Into(&out.Tags)
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if out.CreditCost == 0 {
		out.CreditCost = 1
	}
	return out
}

func CardsFrom(ctx context.Context, st storage.ObjectStore, rows []domain.Template, favs map[string]bool) []TemplateCard {
	out := make([]TemplateCard, 0, len(rows))
	for _, t := range rows {
		out = append(out, CardFrom(ctx, st, t, favs[t.ID]))
	}
	return out
}

type Template struct {
	TemplateCard
	Subtitle   string   `json:"subtitle"`
	SampleURLs []string `json:"sample_urls"`
	Output     struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Aspect string `json:"aspect"`
	} `json:"output"`
	Style    string  `json:"style"`
	Category *IDName `json:"category,omitempty"`
}

func TemplateFrom(ctx context.Context, st storage.ObjectStore, t domain.Template, fav bool, cat *domain.Category) Template {
	out := Template{TemplateCard: CardFrom(ctx, st, t, fav)}
	if t.Subtitle != nil {
		out.Subtitle = *t.Subtitle
	}
	var keys []string
	_ = t.SampleKeys.Into(&keys)
	out.SampleURLs = []string{}
	for _, k := range keys {
		out.SampleURLs = append(out.SampleURLs, storage.URLFor(ctx, st, k, assetTTL))
	}
	var cfg domain.GenConfig
	_ = t.GenConfig.Into(&cfg)
	out.Style = cfg.Style
	if out.Style == "" {
		out.Style = "photo"
	}
	w, h := 1200, 1600
	if cfg.Output != nil && cfg.Output.Width > 0 {
		w, h = cfg.Output.Width, cfg.Output.Height
	}
	if t.Module == domain.ModuleAvatar {
		w, h = 1024, 1024
	}
	out.Output.Width, out.Output.Height = w, h
	out.Output.Aspect = aspect(w, h)
	if cat != nil {
		out.Category = &IDName{ID: cat.ID, Name: cat.Name}
	}
	return out
}

func aspect(w, h int) string {
	if w == h {
		return "1:1"
	}
	if w*4 == h*3 {
		return "3:4"
	}
	return fmt.Sprintf("%d:%d", w, h)
}

type Collection struct {
	ID            string `json:"id"`
	Module        string `json:"module"`
	Name          string `json:"name"`
	CoverURL      string `json:"cover_url"`
	Desc          string `json:"desc"`
	TemplateCount int64  `json:"template_count"`
}

func CollectionFrom(ctx context.Context, st storage.ObjectStore, c domain.Collection, count int64) Collection {
	out := Collection{ID: c.ID, Module: string(c.Module), Name: c.Name, CoverURL: storage.URLFor(ctx, st, c.CoverKey, assetTTL), TemplateCount: count}
	if c.Description != nil {
		out.Desc = *c.Description
	}
	return out
}

type Banner struct {
	Title    string         `json:"title"`
	Subtitle string         `json:"subtitle"`
	ImageURL string         `json:"image_url"`
	Link     map[string]any `json:"link"`
}

func BannerFrom(ctx context.Context, st storage.ObjectStore, b *domain.Banner) *Banner {
	if b == nil {
		return nil
	}
	out := &Banner{Title: b.Title, ImageURL: storage.URLFor(ctx, st, b.ImageKey, assetTTL), Link: map[string]any{"type": "upload"}}
	if b.Subtitle != nil {
		out.Subtitle = *b.Subtitle
	}
	_ = b.Link.Into(&out.Link)
	return out
}

type Rail struct {
	Title        string         `json:"title"`
	CategoryID   string         `json:"category_id,omitempty"`
	CollectionID string         `json:"collection_id,omitempty"`
	Templates    []TemplateCard `json:"templates"`
}

type Home struct {
	Banner        *Banner        `json:"banner"`
	HotSpecs      []Spec         `json:"hot_specs,omitempty"`
	MoreSpecs     []Spec         `json:"more_specs,omitempty"`
	HotCategories []Category     `json:"hot_categories,omitempty"`
	Collections   []Collection   `json:"collections,omitempty"`
	HotTemplates  []TemplateCard `json:"hot_templates,omitempty"`
	Rails         []Rail         `json:"rails,omitempty"`
}

type Photo struct {
	ID         string `json:"id"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	PreviewURL string `json:"preview_url"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
}

func PhotoFrom(p *domain.Photo, url string) Photo {
	return Photo{ID: p.ID, Width: p.Width, Height: p.Height, PreviewURL: url, ExpiresAt: ts(p.ExpiresAt), CreatedAt: ts(p.CreatedAt)}
}

type WorkSpec struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	WidthMM  float64 `json:"width_mm"`
	HeightMM float64 `json:"height_mm"`
	WidthPx  int     `json:"width_px"`
	HeightPx int     `json:"height_px"`
}

type Work struct {
	ID        string         `json:"id"`
	Module    string         `json:"module"`
	URL       string         `json:"url"`
	ThumbURL  string         `json:"thumb_url"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	CreatedAt string         `json:"created_at"`
	Spec      *WorkSpec      `json:"spec,omitempty"`
	Template  *IDName        `json:"template,omitempty"`
	Meta      map[string]any `json:"meta"`
	AILabel   bool           `json:"ai_label"`
	TaskID    string         `json:"task_id"`
}

func WorkFrom(w *domain.Work, url, thumb string, tpls map[string]domain.Template, specs map[string]domain.Spec) Work {
	out := Work{ID: w.ID, Module: string(w.Module), URL: url, ThumbURL: thumb, Width: w.Width, Height: w.Height, CreatedAt: ts(w.CreatedAt), AILabel: w.AILabel}
	if w.TaskID != nil {
		out.TaskID = *w.TaskID
	}
	_ = w.Meta.Into(&out.Meta)
	if out.Meta == nil {
		out.Meta = map[string]any{}
	}
	if w.TemplateID != nil {
		if t, ok := tpls[*w.TemplateID]; ok {
			out.Template = &IDName{ID: t.ID, Name: t.Name}
		} else {
			out.Template = &IDName{ID: *w.TemplateID}
		}
	}
	if w.SpecID != nil {
		if s, ok := specs[*w.SpecID]; ok {
			out.Spec = &WorkSpec{ID: s.ID, Name: s.Name, WidthMM: s.WidthMM, HeightMM: s.HeightMM, WidthPx: s.WidthPx, HeightPx: s.HeightPx}
		}
	}
	return out
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Task struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	Stage           *string        `json:"stage"`
	Module          string         `json:"module"`
	Kind            string         `json:"kind"`
	UsesGenmodel    bool           `json:"uses_genmodel"`
	CreditsConsumed uint           `json:"credits_consumed"`
	CreatedAt       string         `json:"created_at"`
	StartedAt       *string        `json:"started_at"`
	FinishedAt      *string        `json:"finished_at"`
	Work            *Work          `json:"work"`
	Error           *TaskError     `json:"error"`
	Refunded        bool           `json:"refunded"`
	TemplateID      *string        `json:"template_id,omitempty"`
	SpecID          *string        `json:"spec_id,omitempty"`
	PhotoID         string         `json:"photo_id"`
	Params          map[string]any `json:"params"`
	TargetName      string         `json:"target_name"`
}

func TaskFrom(t *domain.Task, work *Work, targetName string) Task {
	out := Task{ID: t.ID, Status: string(t.Status), Module: string(t.Module), Kind: string(t.Kind), UsesGenmodel: t.UsesGenmodel,
		CreditsConsumed: t.CreditsConsumed, CreatedAt: ts(t.CreatedAt), Work: work, Refunded: t.RefundLedgerID != nil,
		TemplateID: t.TemplateID, SpecID: t.SpecID, PhotoID: t.PhotoID, TargetName: targetName}
	if t.Stage != nil {
		s := string(*t.Stage)
		out.Stage = &s
	}
	if t.StartedAt != nil {
		s := ts(*t.StartedAt)
		out.StartedAt = &s
	}
	if t.FinishedAt != nil {
		s := ts(*t.FinishedAt)
		out.FinishedAt = &s
	}
	if t.ErrorCode != nil {
		msg := ""
		if t.ErrorMessage != nil {
			msg = *t.ErrorMessage
		}
		out.Error = &TaskError{Code: *t.ErrorCode, Message: msg}
	}
	_ = t.Params.Into(&out.Params)
	if out.Params == nil {
		out.Params = map[string]any{}
	}
	return out
}

type Share struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url"`
	PosterURL string `json:"poster_url,omitempty"`
}

type ShareLanding struct {
	Type       string  `json:"type"`
	TemplateID *string `json:"template_id,omitempty"`
	SpecID     *string `json:"spec_id,omitempty"`
	PreviewURL string  `json:"preview_url,omitempty"`
	Title      string  `json:"title"`
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
