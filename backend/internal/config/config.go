// Package config loads static configuration from the environment and exposes
// runtime configuration stored in the app_configs table.
package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config is the static, process-level configuration (docs/DEPLOYMENT.md §3).
type Config struct {
	AppEnv        string `env:"APP_ENV" envDefault:"dev"`
	HTTPAddr      string `env:"HTTP_ADDR" envDefault:":8080"`
	PublicBaseURL string `env:"PUBLIC_BASE_URL" envDefault:"http://localhost:8080"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
	CORSOrigins   string `env:"CORS_ORIGINS" envDefault:"http://localhost:5173,http://localhost:5174,http://localhost:5175"`

	DatabaseURL   string `env:"DATABASE_URL" envDefault:"postgres://localhost:5432/yingji?sslmode=disable"`
	RedisAddr     string `env:"REDIS_ADDR" envDefault:"127.0.0.1:6379"`
	RedisUsername string `env:"REDIS_USERNAME"`
	RedisPrefix   string `env:"REDIS_PREFIX" envDefault:"yingji:dev:"`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:""`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	JWTSecretMP    string `env:"JWT_SECRET_MP" envDefault:"dev-mp-secret-change-me"`
	JWTSecretAdmin string `env:"JWT_SECRET_ADMIN" envDefault:"dev-admin-secret-change-me"`
	SignSecret     string `env:"SIGN_SECRET" envDefault:"dev-sign-secret-change-me"`

	WechatAppID            string `env:"WECHAT_APPID" envDefault:""`
	WechatSecret           string `env:"WECHAT_SECRET" envDefault:""`
	WechatAdCallbackSecret string `env:"WECHAT_AD_CALLBACK_SECRET" envDefault:""`
	WechatSubscribeTmplID  string `env:"WECHAT_SUBSCRIBE_TASK_FINISHED" envDefault:""`
	WechatRewardAdUnitID   string `env:"WECHAT_REWARD_AD_UNIT_ID" envDefault:""`

	StorageDriver      string `env:"STORAGE_DRIVER" envDefault:"local"` // local | cos | oss
	LocalStorageDir    string `env:"LOCAL_STORAGE_DIR" envDefault:"./data/objects"`
	OSSRegion          string `env:"OSS_REGION" envDefault:"cn-beijing"`
	OSSBucket          string `env:"OSS_BUCKET"`
	OSSPrefix          string `env:"OSS_PREFIX"`
	OSSEndpoint        string `env:"OSS_ENDPOINT"`
	OSSPublicEndpoint  string `env:"OSS_PUBLIC_ENDPOINT"`
	OSSAccessKeyID     string `env:"OSS_ACCESS_KEY_ID"`
	OSSAccessKeySecret string `env:"OSS_ACCESS_KEY_SECRET"`
	DBMaxOpen          int    `env:"DB_MAX_OPEN" envDefault:"2"`
	DBMaxIdle          int    `env:"DB_MAX_IDLE" envDefault:"0"`
	NewAPIKey          string `env:"NEWAPI_API_KEY"`
	NewAPIBaseURL      string `env:"NEWAPI_BASE_URL" envDefault:"https://www.ggwk1.online/v1"`
	NewAPIModel        string `env:"NEWAPI_MODEL" envDefault:"gpt-image-2.5"`
	COSBucketURL       string `env:"COS_BUCKET_URL" envDefault:""` // https://<bucket>.cos.<region>.myqcloud.com
	COSSecretID        string `env:"COS_SECRET_ID" envDefault:""`
	COSSecretKey       string `env:"COS_SECRET_KEY" envDefault:""`
	COSCDNHost         string `env:"COS_CDN_HOST" envDefault:""`

	FaceProvider     string `env:"FACE_PROVIDER" envDefault:"mock"` // mock | tencent
	TencentSecretID  string `env:"TENCENT_SECRET_ID" envDefault:""`
	TencentSecretKey string `env:"TENCENT_SECRET_KEY" envDefault:""`
	TencentRegion    string `env:"TENCENT_REGION" envDefault:"ap-guangzhou"`

	GenProviderDefault string `env:"GEN_PROVIDER_DEFAULT" envDefault:"mock"` // mock | volcengine | newapi
	GenConcurrency     int    `env:"GEN_CONCURRENCY" envDefault:"4"`
	VolcengineAPIKey   string `env:"VOLCENGINE_API_KEY" envDefault:""`
	VolcengineModel    string `env:"VOLCENGINE_MODEL" envDefault:"doubao-seedream-4-0-250828"`
	VolcengineBaseURL  string `env:"VOLCENGINE_BASE_URL" envDefault:"https://ark.cn-beijing.volces.com/api/v3"`

	PosterFontPath string `env:"POSTER_FONT_PATH" envDefault:""`
	PrivacyURL     string `env:"PRIVACY_POLICY_URL" envDefault:"https://example.com/privacy"`
	AgreementURL   string `env:"USER_AGREEMENT_URL" envDefault:"https://example.com/agreement"`

	AdminInitUser     string `env:"ADMIN_INIT_USER" envDefault:"admin"`
	AdminInitPassword string `env:"ADMIN_INIT_PASSWORD" envDefault:"admin123"`

	UploadMaxBytes int64 `env:"UPLOAD_MAX_BYTES" envDefault:"10485760"`
}

// Load reads .env (if present) then the environment.
func Load() (*Config, error) {
	_ = godotenv.Load()
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) IsProd() bool { return c.AppEnv == "prod" }
func (c *Config) IsDev() bool  { return c.AppEnv == "dev" }

// CORSList splits the comma-separated origins.
func (c *Config) CORSList() []string {
	var out []string
	for _, o := range strings.Split(c.CORSOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// Validate 在启动时拒绝配置拼写错误和生产环境的开发凭据。
func (c *Config) Validate() error {
	if !strings.HasPrefix(c.RedisPrefix, "yingji:") || !strings.HasSuffix(c.RedisPrefix, ":") || strings.ContainsAny(c.RedisPrefix, " *?[]{}\\\t\n") {
		return fmt.Errorf("Redis 前缀必须为 yingji:环境:")
	}

	switch c.StorageDriver {
	case "local", "cos", "oss":
	default:
		return fmt.Errorf("未知存储类型")
	}
	switch c.GenProviderDefault {
	case "mock", "newapi", "volcengine":
	default:
		return fmt.Errorf("未知生图服务")
	}
	if c.GenProviderDefault == "newapi" && c.NewAPIKey == "" {
		return fmt.Errorf("缺少 NEWAPI_API_KEY")
	}
	switch c.FaceProvider {
	case "mock", "tencent", "disabled":
	default:
		return fmt.Errorf("未知视觉服务")
	}
	if c.FaceProvider == "tencent" && (c.TencentSecretID == "" || c.TencentSecretKey == "") {
		return fmt.Errorf("缺少腾讯视觉服务凭据")
	}
	if c.GenConcurrency < 1 || c.DBMaxOpen < 1 || c.DBMaxIdle < 0 || c.DBMaxIdle > c.DBMaxOpen {
		return fmt.Errorf("并发或连接池配置无效")
	}
	if c.IsProd() {
		if c.GenProviderDefault == "mock" || c.FaceProvider == "mock" {
			return fmt.Errorf("生产环境禁止模拟生图和人脸检测")
		}
		if c.StorageDriver != "oss" {
			return fmt.Errorf("生产环境必须使用 OSS")
		}
		for _, s := range []string{c.JWTSecretMP, c.JWTSecretAdmin, c.SignSecret} {
			if len(s) < 32 || strings.Contains(s, "change-me") {
				return fmt.Errorf("生产环境签名密钥不安全")
			}
		}
	}
	return nil
}
