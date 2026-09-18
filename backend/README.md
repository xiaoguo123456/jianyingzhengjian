# 映己后端（Go）

Gin + GORM（PostgreSQL 16）+ Redis/Asynq。部署方式、环境地址、模板维护与第三方配置见 [部署说明](deploy/README.md)。

## 本地运行

准备 PostgreSQL 和 Redis，复制 `.env.example` 为 `.env`，填写 `DATABASE_URL`、`REDIS_ADDR`。生产和测试复用现有实例，本地可选择 `deploy/docker-compose.yml` 的数据库服务。

```sh
go run ./cmd/migrate up
go run ./cmd/migrate seed
go run ./cmd/api
go run ./cmd/worker
```

API 和 Worker 分别在两个终端启动。开发环境支持模拟视觉、生图与本地存储；测试和生产禁用 `h5_dev` 登录。`seed` 仅用于首次初始化，会覆盖同 ID 的初始模板。

## 测试

```sh
go test ./...
go vet ./...
TEST_DATABASE_URL='postgres://账号:密码@127.0.0.1:5432/yingji_ci?sslmode=disable' TEST_REDIS_ADDR=127.0.0.1:6379 go test ./...
```

数据库测试仅接受名称以 `_ci` 结尾的数据库，每个测试使用独立 schema，结束后删除该 schema。Redis 隔离测试仅接受本地实例，使用 DB 12 和独立前缀，不清空数据库。未提供测试连接时，对应集成测试跳过。CI 使用 PostgreSQL 16 与 Redis 验证积分、任务、迁移及队列隔离。

## 目录

- `cmd`：API、Worker、迁移、健康检查、OSS 验收。
- `internal/domain`：数据模型。
- `internal/migration/postgres`：版本化 PostgreSQL SQL。
- `internal/provider`：NewAPI、OSS、微信、视觉等适配器。
- `internal/pkg/redisx`：缓存与 Asynq 命名空间隔离。
- `internal/service`：业务服务。
- `internal/transport`：HTTP 与任务队列。

## 外部服务

NewAPI 图像编辑和 OSS 已通过真实接口验收。微信与腾讯视觉服务仍需本项目凭据和真实业务验证。生产使用 `FACE_PROVIDER=disabled` 时会拒绝图片处理，配置真实视觉服务后才能开放生成流程。

海报字体通过 `POSTER_FONT_PATH` 配置，必须为可读取的中文字体文件。
