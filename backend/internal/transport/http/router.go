// Package http assembles the gin router (docs/BACKEND_ARCHITECTURE.md §4).
package http

import (
	"context"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"yingji/backend/internal/app"
	"yingji/backend/internal/provider/storage"
	adminh "yingji/backend/internal/transport/http/admin"
	"yingji/backend/internal/transport/http/hooks"
	"yingji/backend/internal/transport/http/middleware"
	"yingji/backend/internal/transport/http/public"
)

func New(a *app.App) *gin.Engine {
	if a.Cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(a.Log), middleware.Logger(a.Log), middleware.CORS(a.Cfg.CORSList()))
	r.MaxMultipartMemory = 16 << 20

	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 2*time.Second)
		defer cancel()
		if sqlDB, err := a.DB.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
			c.String(503, "db")
			return
		}
		if err := a.RDB.Ping(ctx).Err(); err != nil {
			c.String(503, "redis")
			return
		}
		c.String(200, "ok")
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if a.LocalStore != nil {
		r.GET("/files/*key", serveLocal(a))
	}

	pub := public.New(a)
	v1 := r.Group("/v1")
	v1.Use(middleware.BodyLimit(a.Cfg.UploadMaxBytes + 1<<20))
	v1.POST("/auth/login", middleware.RateLimit(a.RDB, "login", 30, time.Minute), pub.Login)
	v1.POST("/shares/:id/open", middleware.OptionalUser(a.Auth.Parse), pub.OpenShare)

	user := v1.Group("", middleware.UserAuth(a.Auth.Parse), middleware.RateLimit(a.RDB, "general", 120, time.Minute))
	{
		user.GET("/me", pub.Me)
		user.PUT("/me", pub.UpdateMe)
		user.POST("/me/avatar", pub.UploadAvatar)
		user.POST("/me/privacy-agree", pub.AgreePrivacy)
		user.DELETE("/me", pub.DeleteMe)

		user.GET("/credits", pub.Credits)
		user.POST("/ads/sessions", middleware.RateLimit(a.RDB, "ad_session", 30, time.Hour), pub.CreateAdSession)
		user.POST("/ads/sessions/:id/claim", middleware.RateLimit(a.RDB, "ad_claim", 30, time.Hour), pub.ClaimAd)

		user.GET("/home/:module", pub.Home)
		user.GET("/spec-categories", pub.SpecCategories)
		user.GET("/specs", pub.Specs)
		user.GET("/specs/:id", pub.Spec)
		user.GET("/template-categories", pub.TemplateCategories)
		user.GET("/templates", pub.Templates)
		user.GET("/templates/:id", pub.Template)
		user.GET("/collections", pub.Collections)
		user.GET("/collections/:id", pub.Collection)
		user.GET("/clothing-options", pub.ClothingOptions)

		user.POST("/photos", middleware.RateLimit(a.RDB, "upload", 30, time.Hour), pub.UploadPhoto)
		user.GET("/photos", pub.Photos)
		user.DELETE("/photos/:id", pub.DeletePhoto)

		user.POST("/tasks", middleware.RateLimit(a.RDB, "task_create", 10, time.Minute), pub.CreateTask)
		user.GET("/tasks", pub.Tasks)
		user.GET("/tasks/:id", pub.Task)
		user.POST("/tasks/:id/regenerate", middleware.RateLimit(a.RDB, "task_create", 10, time.Minute), pub.Regenerate)

		user.GET("/works", pub.Works)
		user.GET("/works/summary", pub.WorksSummary)
		user.GET("/works/:id", pub.Work)
		user.GET("/works/:id/download", pub.DownloadWork)
		user.DELETE("/works/:id", pub.DeleteWork)

		user.GET("/favorites", pub.Favorites)
		user.PUT("/favorites/:id", pub.Favorite)
		user.DELETE("/favorites/:id", pub.Unfavorite)

		user.POST("/shares", pub.CreateShare)
		user.GET("/shares", pub.Shares)
		user.GET("/shares/rewards", pub.ShareRewards)
		user.DELETE("/shares/:id", pub.DeleteShare)

		user.POST("/events", pub.Events)
	}

	hk := hooks.New(a)
	r.GET("/webhooks/wechat/ad-reward", hk.AdReward)
	r.POST("/webhooks/wechat/media-check", hk.MediaCheck)

	ad := adminh.New(a)
	adm := r.Group("/admin/v1")
	adm.POST("/auth/login", middleware.RateLimit(a.RDB, "admin_login", 10, time.Minute), ad.Login)
	auth := adm.Group("", middleware.AdminAuth(a.Admin.Parse))
	{
		auth.GET("/configs", ad.Configs)
		auth.PUT("/configs", ad.SetConfigs)
		auth.POST("/assets", ad.UploadAsset)
		auth.POST("/templates/validate", ad.ValidateTemplate)
		auth.GET("/users", ad.User)
		auth.GET("/users/:id", ad.User)
		auth.POST("/users/:id/credits", ad.AdjustCredits)
		auth.GET("/tasks", ad.Tasks)
		auth.POST("/tasks/:id/refund", ad.RefundTask)
		auth.GET("/stats/funnel", ad.Funnel)
		auth.GET("/stats/templates", ad.TemplateStats)
		auth.GET("/stats/shares", ad.ShareStats)
		auth.GET("/audit-logs", ad.AuditLogs)
		auth.GET("/:resource", ad.List)
		auth.POST("/:resource", ad.Upsert)
		auth.DELETE("/:resource/:id", ad.Delete)
	}
	return r
}

// serveLocal serves objects from the local store; private keys need a valid signature.
func serveLocal(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimPrefix(c.Param("key"), "/")
		if strings.Contains(key, "..") {
			c.Status(http.StatusBadRequest)
			return
		}
		if !storage.IsPublicKey(key) {
			exp, _ := strconv.ParseInt(c.Query("exp"), 10, 64)
			if !a.LocalStore.Verify(key, exp, c.Query("sig")) {
				c.Status(http.StatusForbidden)
				return
			}
		}
		rc, err := a.Store.Get(c, key)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer rc.Close()
		ct := "application/octet-stream"
		switch path.Ext(key) {
		case ".jpg", ".jpeg":
			ct = "image/jpeg"
		case ".png":
			ct = "image/png"
		}
		c.Header("Content-Type", ct)
		c.Header("Cache-Control", "private, max-age=600")
		if storage.IsPublicKey(key) {
			c.Header("Cache-Control", "public, max-age=86400")
		}
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, rc)
	}
}
