// Package public holds the Mini Program-facing handlers (docs/API.md §2–10).
package public

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"yingji/backend/internal/app"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/httpx"
	"yingji/backend/internal/service/catalogue"
	"yingji/backend/internal/service/event"
	"yingji/backend/internal/service/share"
	"yingji/backend/internal/service/task"
	"yingji/backend/internal/transport/http/dto"
)

type Handlers struct{ a *app.App }

func New(a *app.App) *Handlers { return &Handlers{a: a} }

func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 100 {
		s = 20
	}
	return p, s
}

// ---- auth & profile ----

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		Code     string `json:"code" binding:"required"`
		ShareID  string `json:"share_id"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	r, err := h.a.Auth.Login(c, req.Provider, req.Code, req.ShareID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"token": r.Token, "expires_in": r.ExpiresIn, "is_new": r.IsNew,
		"user": dto.UserFrom(r.User, h.a.Profile.AvatarURL(c, r.User), h.a.Profile.WorksCount(c, r.User.ID))})
}

func (h *Handlers) Me(c *gin.Context) {
	uid := httpx.UserID(c)
	u, err := h.a.Profile.Get(c, uid)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	bal, err := h.a.Credit.Get(c, uid)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	rt := h.a.Runtime
	cfg := gin.H{
		"ads_enabled":            rt.Bool(c, "ads_enabled"),
		"ad_unit_ids":            gin.H{"reward": h.a.Cfg.WechatRewardAdUnitID},
		"subscribe_template_ids": gin.H{"task_finished": h.a.Cfg.WechatSubscribeTmplID},
		"privacy_policy_url":     h.a.Cfg.PrivacyURL,
		"user_agreement_url":     h.a.Cfg.AgreementURL,
		"retention_days":         rt.Int(c, "photo_retention_days"),
		"share_reward":           gin.H{"enabled": rt.Bool(c, "share_reward_enabled"), "per_reward": max(1, rt.Int(c, "share_reward_per")), "daily_cap": rt.Int(c, "share_reward_daily_cap")},
	}
	httpx.OK(c, gin.H{"user": dto.UserFrom(u, h.a.Profile.AvatarURL(c, u), h.a.Profile.WorksCount(c, uid)), "credits": bal, "config": cfg})
}

func (h *Handlers) UpdateMe(c *gin.Context) {
	var req struct {
		Nickname *string `json:"nickname"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	if req.Nickname != nil {
		if err := h.a.Profile.UpdateNickname(c, httpx.UserID(c), *req.Nickname); err != nil {
			httpx.Fail(c, err)
			return
		}
	}
	u, _ := h.a.Profile.Get(c, httpx.UserID(c))
	httpx.OK(c, gin.H{"user": dto.UserFrom(u, h.a.Profile.AvatarURL(c, u), h.a.Profile.WorksCount(c, u.ID))})
}

func (h *Handlers) UploadAvatar(c *gin.Context) {
	data, ct, err := readUpload(c, 5<<20)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	url, err := h.a.Profile.UploadAvatar(c, httpx.UserID(c), data, ct)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"avatar_url": url})
}

func (h *Handlers) AgreePrivacy(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Version == "" {
		req.Version = "latest"
	}
	if err := h.a.Profile.AgreePrivacy(c, httpx.UserID(c), req.Version); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

func (h *Handlers) DeleteMe(c *gin.Context) {
	if err := h.a.Profile.DeleteAccount(c, httpx.UserID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Accepted(c, gin.H{})
}

func readUpload(c *gin.Context, max int64) ([]byte, string, error) {
	fh, err := c.FormFile("file")
	if err != nil {
		// Name the fields we did receive: the usual cause is a client sending the
		// file under a different part name.
		var got []string
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			for k := range form.File {
				got = append(got, k)
			}
			for k := range form.Value {
				got = append(got, k+"(value)")
			}
		}
		slog.Warn("upload: no file part", "request_id", httpx.RequestID(c), "parts", got, "content_type", c.ContentType(), "err", err)
		return nil, "", apperr.BadRequest("缺少文件").WithData(map[string]any{"expected_part": "file", "received_parts": got})
	}
	if fh.Size > max {
		return nil, "", apperr.PayloadTooLarge()
	}
	f, err := fh.Open()
	if err != nil {
		return nil, "", apperr.BadRequest("文件无法读取")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(data)) > max {
		return nil, "", apperr.PayloadTooLarge()
	}
	return data, http.DetectContentType(data), nil
}

// ---- credits & ads ----

func (h *Handlers) Credits(c *gin.Context) {
	bal, err := h.a.Credit.Get(c, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, bal)
}

func (h *Handlers) CreateAdSession(c *gin.Context) {
	s, err := h.a.Ads.CreateSession(c, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"session_id": s.ID, "ad_unit_id": s.AdUnitID, "expires_at": h.a.Ads.ExpiresAt(c, s).UTC().Format(time.RFC3339)})
}

func (h *Handlers) ClaimAd(c *gin.Context) {
	var req struct {
		IsEnded bool `json:"is_ended"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	granted, err := h.a.Ads.Claim(c, httpx.UserID(c), c.Param("id"), req.IsEnded)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	bal, _ := h.a.Credit.Get(c, httpx.UserID(c))
	httpx.OK(c, gin.H{"granted": granted, "credits": bal})
}

// ---- catalogue ----

func (h *Handlers) Home(c *gin.Context) {
	m := domain.Module(c.Param("module"))
	if !m.Valid() {
		httpx.Fail(c, apperr.NotFound("模块"))
		return
	}
	hd, err := h.a.Catalogue.Home(c, m)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	st := h.a.Store
	out := dto.Home{Banner: dto.BannerFrom(c, st, hd.Banner)}
	if m == domain.ModuleIDPhoto {
		var ids []string
		for _, s := range append(append([]domain.Spec{}, hd.HotSpecs...), hd.MoreSpecs...) {
			ids = append(ids, s.CategoryID)
		}
		cats := h.a.Catalogue.CategoriesByIDs(c, ids)
		out.HotSpecs = dto.SpecsFrom(hd.HotSpecs, cats)
		out.MoreSpecs = dto.SpecsFrom(hd.MoreSpecs, cats)
	} else {
		out.HotCategories = []dto.Category{}
		for _, ct := range hd.HotCategories {
			out.HotCategories = append(out.HotCategories, dto.CategoryFrom(c, st, ct))
		}
		if len(hd.Collections) > 0 {
			var ids []string
			for _, col := range hd.Collections {
				ids = append(ids, col.ID)
			}
			counts := h.a.Catalogue.CollectionTemplateCounts(c, ids)
			for _, col := range hd.Collections {
				out.Collections = append(out.Collections, dto.CollectionFrom(c, st, col, counts[col.ID]))
			}
		}
		var tids []string
		for _, t := range hd.HotTemplates {
			tids = append(tids, t.ID)
		}
		for _, r := range hd.Rails {
			for _, t := range r.Templates {
				tids = append(tids, t.ID)
			}
		}
		fav := h.a.Catalogue.FavoritedSet(c, httpx.UserID(c), tids)
		out.HotTemplates = dto.CardsFrom(c, st, hd.HotTemplates, fav)
		for _, r := range hd.Rails {
			out.Rails = append(out.Rails, dto.Rail{Title: r.Title, CategoryID: r.CategoryID, CollectionID: r.CollectionID, Templates: dto.CardsFrom(c, st, r.Templates, fav)})
		}
	}
	httpx.OK(c, out)
}

func (h *Handlers) SpecCategories(c *gin.Context) {
	cats, err := h.a.Catalogue.SpecCategories(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	out := []dto.IDName{}
	for _, ct := range cats {
		out = append(out, dto.IDName{ID: ct.ID, Name: ct.Name})
	}
	httpx.OK(c, out)
}

func (h *Handlers) Specs(c *gin.Context) {
	specs, err := h.a.Catalogue.Specs(c, c.Query("category_id"), c.Query("hot") == "1")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var ids []string
	for _, s := range specs {
		ids = append(ids, s.CategoryID)
	}
	items := dto.SpecsFrom(specs, h.a.Catalogue.CategoriesByIDs(c, ids))
	httpx.OK(c, dto.Paged[dto.Spec]{Items: items, Total: int64(len(items))})
}

func (h *Handlers) Spec(c *gin.Context) {
	s, err := h.a.Catalogue.Spec(c, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, dto.SpecFrom(*s, h.a.Catalogue.CategoriesByIDs(c, []string{s.CategoryID})))
}

func (h *Handlers) TemplateCategories(c *gin.Context) {
	m := domain.Module(c.Query("module"))
	cats, err := h.a.Catalogue.TemplateCategories(c, m)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	out := []dto.Category{}
	for _, ct := range cats {
		out = append(out, dto.CategoryFrom(c, h.a.Store, ct))
	}
	httpx.OK(c, out)
}

func (h *Handlers) Templates(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Catalogue.Templates(c, catalogue.TemplateQuery{Module: domain.Module(c.Query("module")), CategoryID: c.Query("category_id"),
		CollectionID: c.Query("collection_id"), Hot: c.Query("hot") == "1", Page: p, PageSize: s})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var ids []string
	for _, t := range rows {
		ids = append(ids, t.ID)
	}
	httpx.OK(c, dto.NewPaged(dto.CardsFrom(c, h.a.Store, rows, h.a.Catalogue.FavoritedSet(c, httpx.UserID(c), ids)), total, p, s))
}

func (h *Handlers) Template(c *gin.Context) {
	t, err := h.a.Catalogue.Template(c, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var cat *domain.Category
	if t.CategoryID != nil {
		if ct, ok := h.a.Catalogue.CategoriesByIDs(c, []string{*t.CategoryID})[*t.CategoryID]; ok {
			cat = &ct
		}
	}
	fav := h.a.Catalogue.FavoritedSet(c, httpx.UserID(c), []string{t.ID})[t.ID]
	httpx.OK(c, dto.TemplateFrom(c, h.a.Store, *t, fav, cat))
}

func (h *Handlers) Collections(c *gin.Context) {
	cols, err := h.a.Catalogue.Collections(c, domain.Module(c.Query("module")))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var ids []string
	for _, col := range cols {
		ids = append(ids, col.ID)
	}
	counts := h.a.Catalogue.CollectionTemplateCounts(c, ids)
	out := []dto.Collection{}
	for _, col := range cols {
		out = append(out, dto.CollectionFrom(c, h.a.Store, col, counts[col.ID]))
	}
	httpx.OK(c, out)
}

func (h *Handlers) Collection(c *gin.Context) {
	col, rows, err := h.a.Catalogue.Collection(c, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var ids []string
	for _, t := range rows {
		ids = append(ids, t.ID)
	}
	httpx.OK(c, gin.H{"collection": dto.CollectionFrom(c, h.a.Store, *col, int64(len(rows))),
		"templates": dto.CardsFrom(c, h.a.Store, rows, h.a.Catalogue.FavoritedSet(c, httpx.UserID(c), ids))})
}

func (h *Handlers) ClothingOptions(c *gin.Context) { httpx.OK(c, domain.ClothingOptions) }

// ---- photos ----

func (h *Handlers) UploadPhoto(c *gin.Context) {
	data, _, err := readUpload(c, h.a.Cfg.UploadMaxBytes)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	u, err := h.a.Profile.Get(c, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if u.PrivacyAgreedAt == nil {
		httpx.Fail(c, apperr.Forbidden().WithMessage("请先同意隐私政策"))
		return
	}
	p, check, err := h.a.Photo.Upload(c, u.ID, domain.Module(c.PostForm("module")), data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, gin.H{"photo": dto.PhotoFrom(p, h.a.Photo.URL(c, p)), "check": check})
}

func (h *Handlers) Photos(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Photo.List(c, httpx.UserID(c), p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	items := []dto.Photo{}
	for i := range rows {
		items = append(items, dto.PhotoFrom(&rows[i], h.a.Photo.URL(c, &rows[i])))
	}
	httpx.OK(c, dto.NewPaged(items, total, p, s))
}

func (h *Handlers) DeletePhoto(c *gin.Context) {
	if err := h.a.Photo.Delete(c, httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

// ---- tasks ----

func (h *Handlers) taskDTO(c *gin.Context, t *domain.Task) dto.Task {
	var w *dto.Work
	if t.WorkID != nil {
		if wk, err := h.a.Work.Load(c, *t.WorkID); err == nil {
			w = h.workDTO(c, wk)
		}
	}
	names := h.a.Task.TargetNames(c, []domain.Task{*t})
	return dto.TaskFrom(t, w, names[t.ID])
}

func (h *Handlers) CreateTask(c *gin.Context) {
	var req struct {
		Kind         string               `json:"kind" binding:"required"`
		SpecID       string               `json:"spec_id"`
		TemplateID   string               `json:"template_id"`
		PhotoID      string               `json:"photo_id" binding:"required"`
		Params       domain.IDPhotoParams `json:"params"`
		Notify       bool                 `json:"notify"`
		ParentTaskID *string              `json:"parent_task_id"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	in := task.CreateInput{Kind: domain.TaskKind(req.Kind), SpecID: req.SpecID, TemplateID: req.TemplateID, PhotoID: req.PhotoID,
		Params: req.Params, Notify: req.Notify, IdempotencyKey: c.GetHeader("Idempotency-Key")}
	if req.ParentTaskID != nil {
		in.ParentTaskID = *req.ParentTaskID
	}
	t, existing, err := h.a.Task.Create(c, httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	bal, _ := h.a.Credit.Get(c, httpx.UserID(c))
	body := gin.H{"task": h.taskDTO(c, t), "credits": bal}
	if existing {
		httpx.OK(c, body)
		return
	}
	httpx.Created(c, body)
}

func (h *Handlers) Task(c *gin.Context) {
	t, err := h.a.Task.Get(c, httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, h.taskDTO(c, t))
}

func (h *Handlers) Tasks(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Task.List(c, httpx.UserID(c), c.Query("status"), p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	names := h.a.Task.TargetNames(c, rows)
	items := []dto.Task{}
	for i := range rows {
		var w *dto.Work
		if rows[i].WorkID != nil {
			if wk, err := h.a.Work.Load(c, *rows[i].WorkID); err == nil {
				w = h.workDTO(c, wk)
			}
		}
		items = append(items, dto.TaskFrom(&rows[i], w, names[rows[i].ID]))
	}
	httpx.OK(c, dto.NewPaged(items, total, p, s))
}

// Regenerate accepts an optional {"bg": "#RRGGBB"} to redraw an ID photo on another background.
func (h *Handlers) Regenerate(c *gin.Context) {
	var req struct {
		Bg string `json:"bg"`
	}
	if c.Request.ContentLength > 0 {
		if err := httpx.Bind(c, &req); err != nil {
			httpx.Fail(c, err)
			return
		}
	}
	t, existing, err := h.a.Task.Regenerate(c, httpx.UserID(c), c.Param("id"), c.GetHeader("Idempotency-Key"), req.Bg)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	bal, _ := h.a.Credit.Get(c, httpx.UserID(c))
	body := gin.H{"task": h.taskDTO(c, t), "credits": bal}
	if existing {
		httpx.OK(c, body)
		return
	}
	httpx.Created(c, body)
}

// ---- works ----

func (h *Handlers) workDTO(c *gin.Context, w *domain.Work) *dto.Work {
	url, thumb := h.a.Work.URLs(c, w)
	tpls, specs := h.a.Work.Names(c, []domain.Work{*w})
	out := dto.WorkFrom(w, url, thumb, tpls, specs)
	return &out
}

func (h *Handlers) Works(c *gin.Context) {
	p, s := page(c)
	m := domain.Module(c.Query("module"))
	if !m.Valid() {
		m = ""
	}
	rows, total, err := h.a.Work.List(c, httpx.UserID(c), m, p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	tpls, specs := h.a.Work.Names(c, rows)
	items := []dto.Work{}
	for i := range rows {
		url, thumb := h.a.Work.URLs(c, &rows[i])
		items = append(items, dto.WorkFrom(&rows[i], url, thumb, tpls, specs))
	}
	httpx.OK(c, dto.NewPaged(items, total, p, s))
}

func (h *Handlers) WorksSummary(c *gin.Context) {
	sum, err := h.a.Work.Summary(c, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	tpls, specs := h.a.Work.Names(c, sum.Recent)
	recent := []dto.Work{}
	for i := range sum.Recent {
		url, thumb := h.a.Work.URLs(c, &sum.Recent[i])
		recent = append(recent, dto.WorkFrom(&sum.Recent[i], url, thumb, tpls, specs))
	}
	httpx.OK(c, gin.H{"total": sum.Total, "by_module": sum.ByModule, "recent": recent})
}

func (h *Handlers) Work(c *gin.Context) {
	w, err := h.a.Work.Get(c, httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, h.workDTO(c, w))
}

func (h *Handlers) DownloadWork(c *gin.Context) {
	url, exp, err := h.a.Work.DownloadURL(c, httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"url": url, "expires_at": exp.UTC().Format(time.RFC3339)})
}

func (h *Handlers) DeleteWork(c *gin.Context) {
	if err := h.a.Work.Delete(c, httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	h.a.Share.RevokeByWork(c, c.Param("id"))
	httpx.OK(c, gin.H{})
}

// ---- favorites ----

func (h *Handlers) Favorites(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Favorite.List(c, httpx.UserID(c), p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	favs := map[string]bool{}
	for _, t := range rows {
		favs[t.ID] = true
	}
	httpx.OK(c, dto.NewPaged(dto.CardsFrom(c, h.a.Store, rows, favs), total, p, s))
}

func (h *Handlers) Favorite(c *gin.Context) {
	if _, err := h.a.Catalogue.Template(c, c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.a.Favorite.Add(c, httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

func (h *Handlers) Unfavorite(c *gin.Context) {
	if err := h.a.Favorite.Remove(c, httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

// ---- shares ----

func (h *Handlers) shareDTO(sh *domain.Share) dto.Share {
	return dto.Share{ID: sh.ID, Type: string(sh.Type), Path: sh.Path, Title: sh.Title, ImageURL: h.a.Share.PreviewURL(sh), PosterURL: h.a.Share.PosterURL(sh)}
}

func (h *Handlers) CreateShare(c *gin.Context) {
	var req struct {
		Type       string `json:"type" binding:"required"`
		TemplateID string `json:"template_id"`
		WorkID     string `json:"work_id"`
		Module     string `json:"module"`
		Surface    string `json:"surface"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	sh, err := h.a.Share.Create(c, httpx.UserID(c), share.CreateInput{Type: domain.ShareType(req.Type), TemplateID: req.TemplateID, WorkID: req.WorkID, Module: domain.Module(req.Module), Surface: req.Surface})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, gin.H{"share": h.shareDTO(sh)})
}

func (h *Handlers) OpenShare(c *gin.Context) {
	var req struct {
		Platform string `json:"platform"`
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&req)
	sh, err := h.a.Share.Open(c, c.Param("id"), req.Platform, req.DeviceID, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	out := dto.ShareLanding{Type: string(sh.Type), TemplateID: sh.TemplateID, SpecID: sh.SpecID, Title: sh.Title}
	if sh.Status == "active" && sh.Type != domain.ShareTemplate && sh.Type != domain.ShareTab {
		out.PreviewURL = h.a.Share.PreviewURL(sh)
	}
	httpx.OK(c, gin.H{"share": out})
}

func (h *Handlers) Shares(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Share.ListMine(c, httpx.UserID(c), p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	items := []gin.H{}
	for i := range rows {
		items = append(items, gin.H{"share": h.shareDTO(&rows[i]), "opens": rows[i].Opens, "status": rows[i].Status, "created_at": rows[i].CreatedAt})
	}
	httpx.OK(c, dto.NewPaged(items, total, p, s))
}

func (h *Handlers) ShareRewards(c *gin.Context) {
	r, err := h.a.Share.Rewards(c, httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, r)
}

func (h *Handlers) DeleteShare(c *gin.Context) {
	if err := h.a.Share.Revoke(c, httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

// ---- events ----

func (h *Handlers) Events(c *gin.Context) {
	var req struct {
		Events []event.Input `json:"events"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	if len(req.Events) > 100 {
		req.Events = req.Events[:100]
	}
	if err := h.a.Event.Record(c, httpx.UserID(c), req.Events); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"accepted": len(req.Events)})
}
