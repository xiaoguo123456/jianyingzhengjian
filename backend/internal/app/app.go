// Package app wires configuration, infrastructure, providers, engines and services (shared by api and worker).
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"yingji/backend/internal/config"
	gmrouter "yingji/backend/internal/engine/genmodel"
	"yingji/backend/internal/engine/vision"
	"yingji/backend/internal/pipeline/steps"
	"yingji/backend/internal/pkg/jwtx"
	"yingji/backend/internal/provider/face"
	gm "yingji/backend/internal/provider/genmodel"
	"yingji/backend/internal/provider/matting"
	"yingji/backend/internal/provider/storage"
	"yingji/backend/internal/provider/wechat"
	"yingji/backend/internal/service/admin"
	"yingji/backend/internal/service/ads"
	"yingji/backend/internal/service/auth"
	"yingji/backend/internal/service/catalogue"
	"yingji/backend/internal/service/credit"
	"yingji/backend/internal/service/event"
	"yingji/backend/internal/service/favorite"
	"yingji/backend/internal/service/notify"
	"yingji/backend/internal/service/photo"
	"yingji/backend/internal/service/profile"
	"yingji/backend/internal/service/share"
	"yingji/backend/internal/service/task"
	"yingji/backend/internal/service/work"
)

type App struct {
	Cfg        *config.Config
	Runtime    *config.Runtime
	DB         *gorm.DB
	RDB        *redis.Client
	Log        *slog.Logger
	Store      storage.ObjectStore
	LocalStore *storage.Local
	WX         *wechat.Client
	Vision     *vision.Engine
	Gen        *gmrouter.Router

	UserSigner  *jwtx.Signer
	AdminSigner *jwtx.Signer

	Credit    *credit.Service
	Auth      *auth.Service
	Profile   *profile.Service
	Ads       *ads.Service
	Catalogue *catalogue.Service
	Photo     *photo.Service
	Task      *task.Service
	Work      *work.Service
	Favorite  *favorite.Service
	Share     *share.Service
	Event     *event.Service
	Notify    *notify.Service
	Admin     *admin.Service

	Pipeline *steps.Deps
}

func New(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	level := slog.LevelInfo
	if strings.EqualFold(cfg.LogLevel, "debug") {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent), NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MySQLMaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MySQLMaxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}
	rt := config.NewRuntime(db, rdb)

	var store storage.ObjectStore
	var local *storage.Local
	switch cfg.StorageDriver {
	case "oss":
		store, err = storage.NewOSS(cfg.OSSRegion, cfg.OSSEndpoint, cfg.OSSPublicEndpoint, cfg.OSSBucket, cfg.OSSPrefix, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret)
	case "cos":
		store, err = storage.NewCOS(cfg.COSBucketURL, cfg.COSSecretID, cfg.COSSecretKey, cfg.COSCDNHost)
	default:
		local, err = storage.NewLocal(cfg.LocalStorageDir, cfg.PublicBaseURL, cfg.SignSecret)
		store = local
	}
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	wx := wechat.New(cfg.WechatAppID, cfg.WechatSecret, rdb, cfg.IsDev())

	var det face.Detector = face.Mock{}
	var cmp face.Comparer = face.Mock{}
	var mat matting.Matter = matting.Mock{}
	if cfg.FaceProvider == "disabled" {
		det, cmp = face.Disabled{}, face.Disabled{}
	}
	if cfg.FaceProvider == "tencent" {
		t := face.NewTencent(cfg.TencentSecretID, cfg.TencentSecretKey, cfg.TencentRegion)
		det, cmp = t, t
		mat = matting.NewTencent(cfg.TencentSecretID, cfg.TencentSecretKey, cfg.TencentRegion)
	}
	vis := vision.New(det, cmp, mat)

	var models []gm.Model
	if cfg.GenProviderDefault == "mock" && !cfg.IsProd() {
		models = append(models, gm.Mock{Latency: 1500 * time.Millisecond})
	}
	if cfg.NewAPIKey != "" {
		var prices map[string]int
		_ = rt.JSON(ctx, "provider_prices", &prices)
		models = append(models, gm.NewNewAPI(cfg.NewAPIKey, cfg.NewAPIModel, cfg.NewAPIBaseURL, prices["newapi/"+cfg.NewAPIModel]))
	}
	if cfg.VolcengineAPIKey != "" {
		var prices map[string]int
		_ = rt.JSON(ctx, "provider_prices", &prices)
		models = append(models, gm.NewVolcengine(cfg.VolcengineAPIKey, cfg.VolcengineModel, cfg.VolcengineBaseURL, prices["volcengine/"+cfg.VolcengineModel]))
	}
	def := cfg.GenProviderDefault
	if def == "" {
		def = rt.String(ctx, "default_provider")
	}
	gen := gmrouter.NewRouter(def, cfg.GenConcurrency, models...)

	userSigner := jwtx.New(cfg.JWTSecretMP, "mp", 24*time.Hour)
	adminSigner := jwtx.New(cfg.JWTSecretAdmin, "admin", 12*time.Hour)

	a := &App{Cfg: cfg, Runtime: rt, DB: db, RDB: rdb, Log: log, Store: store, LocalStore: local, WX: wx, Vision: vis, Gen: gen,
		UserSigner: userSigner, AdminSigner: adminSigner}
	a.Credit = credit.New(db, rt)
	a.Auth = auth.New(db, wx, userSigner, cfg.IsDev())
	a.Profile = profile.New(db, store)
	a.Ads = ads.New(db, rt, a.Credit, cfg.WechatRewardAdUnitID)
	a.Catalogue = catalogue.New(db, rdb)
	a.Photo = photo.New(db, rt, store, vis)
	a.Task = task.New(db, rt, a.Credit, a.Catalogue, a.Photo, gen, nil)
	a.Work = work.New(db, store, a.Catalogue)
	a.Favorite = favorite.New(db)
	a.Event = event.New(db)
	a.Notify = notify.New(wx, a.Auth, cfg.WechatSubscribeTmplID, log)
	a.Share = share.New(db, rt, store, wx, a.Catalogue, a.Work, a.Credit, a.Event, cfg.PosterFontPath, log)
	a.Admin = admin.New(db, rt, adminSigner, gen)
	a.Pipeline = &steps.Deps{Store: store, Vision: vis, Gen: gen, Cfg: rt, Log: log, FontPath: cfg.PosterFontPath, AppID: cfg.WechatAppID}
	return a, nil
}

func (a *App) Close() {
	if sqlDB, err := a.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = a.RDB.Close()
}
