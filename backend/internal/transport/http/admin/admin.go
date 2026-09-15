// Package admin holds the operations console handlers (docs/API.md §12).
package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"yingji/backend/internal/app"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/httpx"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/provider/storage"
	adminsvc "yingji/backend/internal/service/admin"
	"yingji/backend/internal/transport/http/dto"
)

type Handlers struct{ a *app.App }

func New(a *app.App) *Handlers { return &Handlers{a: a} }

func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	s, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 200 {
		s = 50
	}
	return p, s
}

func dateRange(c *gin.Context) (time.Time, time.Time) {
	to := time.Now().UTC()
	from := to.Add(-7 * 24 * time.Hour)
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t.Add(24 * time.Hour)
		}
	}
	return from, to
}

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	tok, u, err := h.a.Admin.Login(c, req.Username, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"token": tok, "username": u.Username, "role": u.Role})
}

func (h *Handlers) List(c *gin.Context) {
	p, s := page(c)
	filters := map[string]string{}
	for _, k := range []string{"module", "status", "kind", "category_id"} {
		if v := c.Query(k); v != "" {
			filters[k] = v
		}
	}
	rows, total, err := h.a.Admin.List(c, c.Param("resource"), filters, p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": rows, "total": total, "has_more": int64(p*s) < total})
}

func (h *Handlers) Upsert(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		httpx.Fail(c, apperr.BadRequest("读取失败"))
		return
	}
	row, err := h.a.Admin.Upsert(c, c.Param("resource"), body, httpx.AdminID(c), c.ClientIP())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	h.a.Catalogue.InvalidateHome(c)
	httpx.OK(c, row)
}

func (h *Handlers) Delete(c *gin.Context) {
	if err := h.a.Admin.Delete(c, c.Param("resource"), c.Param("id"), httpx.AdminID(c), c.ClientIP()); err != nil {
		httpx.Fail(c, err)
		return
	}
	h.a.Catalogue.InvalidateHome(c)
	httpx.OK(c, gin.H{})
}

func (h *Handlers) ValidateTemplate(c *gin.Context) {
	var t domain.Template
	if err := c.ShouldBindJSON(&t); err != nil {
		httpx.Fail(c, apperr.BadRequest("JSON 无效"))
		return
	}
	if err := h.a.Admin.ValidateTemplate(c, &t); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"valid": true})
}

func (h *Handlers) UploadAsset(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, apperr.BadRequest("缺少文件"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Fail(c, apperr.BadRequest("文件无法读取"))
		return
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, 20<<20))
	ct := http.DetectContentType(data)
	ext := ".jpg"
	if ct == "image/png" {
		ext = ".png"
	}
	key := "assets/" + time.Now().Format("2006/01") + "/" + idgen.New() + ext
	if err := h.a.Store.Put(c, key, bytes.NewReader(data), int64(len(data)), ct); err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.Created(c, gin.H{"key": key, "url": storage.URLFor(c, h.a.Store, key, time.Hour)})
}

func (h *Handlers) Configs(c *gin.Context) { httpx.OK(c, h.a.Runtime.All(c)) }

func (h *Handlers) SetConfigs(c *gin.Context) {
	var body map[string]json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, apperr.BadRequest("JSON 无效"))
		return
	}
	for k, v := range body {
		if err := h.a.Admin.SetConfig(c, k, v, httpx.AdminID(c), c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
	}
	httpx.OK(c, h.a.Runtime.All(c))
}

func (h *Handlers) User(c *gin.Context) {
	v, err := h.a.Admin.FindUser(c, c.Param("id"), c.Query("openid"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, v)
}

func (h *Handlers) AdjustCredits(c *gin.Context) {
	var req struct {
		Delta int    `json:"delta" binding:"required"`
		Note  string `json:"note"`
	}
	if err := httpx.Bind(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	row, err := h.a.Credit.AdminAdjust(c, c.Param("id"), req.Delta, req.Note, idgen.New())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handlers) Tasks(c *gin.Context) {
	p, s := page(c)
	from, to := dateRange(c)
	rows, total, err := h.a.Task.AdminList(c, c.Query("status"), c.Query("module"), c.Query("user_id"), &from, &to, p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": rows, "total": total, "has_more": int64(p*s) < total})
}

func (h *Handlers) RefundTask(c *gin.Context) {
	t, err := h.a.Task.Load(c, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.a.Task.Fail(c, t.ID, domain.ErrProvider, t.CostCents); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{})
}

func (h *Handlers) Funnel(c *gin.Context) {
	from, to := dateRange(c)
	rows, err := h.a.Admin.Funnel(c, from, to, c.Query("module"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h *Handlers) TemplateStats(c *gin.Context) {
	from, to := dateRange(c)
	rows, err := h.a.Admin.TemplateStats(c, from, to)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h *Handlers) ShareStats(c *gin.Context) {
	from, to := dateRange(c)
	rows, err := h.a.Admin.ShareStats(c, from, to, c.Query("module"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rows)
}

func (h *Handlers) AuditLogs(c *gin.Context) {
	p, s := page(c)
	rows, total, err := h.a.Admin.AuditLogs(c, p, s)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": rows, "total": total, "has_more": int64(p*s) < total})
}

var _ = adminsvc.Resources
var _ = dto.IDName{}
