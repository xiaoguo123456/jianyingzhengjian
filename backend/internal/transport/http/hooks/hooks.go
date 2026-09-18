// Package hooks receives WeChat server callbacks (docs/API.md §11).
package hooks

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yingji/backend/internal/app"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/provider/wechat"
)

type Handlers struct{ a *app.App }

func New(a *app.App) *Handlers { return &Handlers{a: a} }

// AdReward handles the rewarded-video server callback. Always 200 to avoid retry storms.
func (h *Handlers) AdReward(c *gin.Context) {
	q := c.Request.URL.Query()
	if !wechat.VerifyAdCallback(q, h.a.Cfg.WechatAdCallbackSecret) {
		h.a.Log.Warn("ad callback: bad signature", "query", q.Encode())
		c.JSON(http.StatusOK, gin.H{"isValid": false})
		return
	}
	userID, err := h.a.Auth.UserByOpenID(c, q.Get("user_id"))
	if err == nil {
		_ = h.a.Ads.ServerVerified(c, userID, q.Get("trans_id"))
	}
	c.JSON(http.StatusOK, gin.H{"isValid": true})
}

// MediaCheck handles the mediaCheckAsync result callback.
func (h *Handlers) MediaCheck(c *gin.Context) {
	var body struct {
		TraceID string `json:"trace_id"`
		Result  struct {
			Suggest string `json:"suggest"`
			Label   int    `json:"label"`
		} `json:"result"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.TraceID == "" {
		c.String(http.StatusOK, "success")
		return
	}
	target, err := h.a.RDB.Get(c, "mod:"+body.TraceID).Result()
	if err != nil {
		c.String(http.StatusOK, "success")
		return
	}
	risky := body.Result.Suggest == "risky"
	status := domain.ModPass
	if risky {
		status = domain.ModRisky
	}
	kind, id, _ := strings.Cut(target, ":")
	switch kind {
	case "work":
		_ = h.a.Work.SetModeration(c, id, status)
		if risky {
			if w, err := h.a.Work.Load(c, id); err == nil {
				if w.TaskID != nil {
					if err := h.a.Task.RejectContent(c, *w.TaskID); err != nil {
						// the consistency job retries rejections of risky works
						h.a.Log.Error("moderation: reject task", "task", *w.TaskID, "err", err)
					}
				}
				h.a.Share.RevokeByWork(c, w.ID)
			}
		}
	case "photo":
		_ = h.a.Photo.SetModeration(c, id, status)
	}
	c.String(http.StatusOK, "success")
}
