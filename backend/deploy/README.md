# 映己后端部署

## 环境与入口

| 环境 | 服务器 | 目录 | 入口 | 发布方式 |
|---|---|---|---|---|
| 测试 | 47.93.60.25 | /opt/yingji-test | https://test-www.qhzhiyin.com/yingji | main 的后端变更推送后自动部署 |
| 生产 | 39.105.228.11 | /opt/yingji-production | https://platform.qhzhiyin.com/yingji | Actions 手动输入完整提交 SHA |

生产只接受已通过测试环境部署的提交。发布失败恢复上一应用镜像，数据库不自动降级。

参考 EnerSight 的环境分离、固定提交发布、SSH 主机校验和回滚方式。镜像在 GitHub 构建后通过 SSH 传输并校验 SHA256，无需复制其他项目的 ACR 凭据。

## 首次配置

1. 在仓库设置 `SSH_KEY`、`TEST_SSH_HOST`、`TEST_SSH_KNOWN_HOSTS`、`PROD_SSH_HOST`、`SSH_KNOWN_HOSTS`。
2. 创建 GitHub `test` 与 `production` 环境。生产由 `prod.yml` 的手动入口控制，测试无需审批。
3. 在服务器的独立目录准备 `.env`，参考本目录示例，权限必须为 `600`。密钥不提交 Git。
4. 测试使用 `DEPLOY_ENV=test`、`APP_ENV=staging`、端口 `8013`、网络 `weishen-test_default`；生产使用 `DEPLOY_ENV=production`、`APP_ENV=prod`、端口 `8014`、网络 `weishen-prod_default`。
5. 为现有网关增加 `gateway/` 中对应的映己路由。Nginx 校验成功后重载，保留所有现有路由。测试网关由花花狗项目管理，后续发布必须保留此路由。

服务器仅运行映己 API、Worker，复用现有 PostgreSQL 16 与 Redis 实例，不部署数据库容器。测试数据库 `yingji_test`、账号 `yingji_test_app`；生产数据库 `yingji_prod`、账号 `yingji_prod_app`。Redis 测试 DB 5、前缀 `yingji:test:`；生产 DB 6、前缀 `yingji:prod:`。缓存、Asynq 内部队列键及订阅频道均加前缀。禁止清空共享 Redis 或扫描删除其他项目前缀。

数据库每进程最多 2 个连接、空闲连接 0；生图并发 1，API/Worker 分别限制 128/384 MB。现有 RDS 未启用 SSL，通过同 VPC 内网以 `sslmode=disable` 连接。发布检查数据库、Redis、API 就绪及 Worker 心跳。

首次发布执行版本化 SQL 和初始模板导入；后续只执行新 SQL，不覆盖后台已编辑的模板。SQL 文件位于 `internal/migration/postgres` 并嵌入二进制，每个版本在事务中执行，校验历史摘要并使用数据库迁移锁。失败会回滚该版本 DDL。`.initialized-postgres` 标记该环境已导入初始数据。

## 接口与存储

- `GEN_PROVIDER_DEFAULT=newapi`，`NEWAPI_BASE_URL=https://www.ggwk1.online/v1`，`NEWAPI_MODEL=gpt-image-2.5`。
- 原图通过 multipart `/images/edits` 上传；支持 Base64 或 HTTPS 图片链接响应。请求失败不自动重发付费请求。
- 输出先按模型支持的尺寸生成，再裁切到模板配置的尺寸。第三方编辑请求已使用项目生成样片验证成功。
- `STORAGE_DRIVER=oss`。测试与生产使用 `yingji/test`、`yingji/production` 前缀。上传可走内网端点，客户端签名始终使用公网端点。
- 所有 OSS 对象设置为私有；素材链接签名有效期 24 小时，用户原图和作品沿用业务设置的短时签名。不改变共享 Bucket 权限。
- 微信平台需配置本项目的 AppID/AppSecret，并加入 API 与 OSS 公网域名白名单。测试与生产不开放 `h5_dev` 登录。
- 不使用人脸检测、比对与抠图服务。上传质检调用同一网关的多模态模型：`INSPECT_PROVIDER=newapi`，`INSPECT_MODEL` 填网关上可识图的模型名。测试可用 `mock`，生产禁止模拟。旧的 `FACE_PROVIDER`、`TENCENT_*` 变量已不再读取，可从服务器 `.env` 删除。

## 模板随时新增

现有管理端为 API，尚无可视化运营后台。管理员登录后通过 `/admin/v1/assets` 上传封面，通过 `/admin/v1/templates` 新增或更新模板，可配置分类、排序、状态、提示词、模型、输出尺寸等。接口自动清理服务端首页缓存；客户端首页和模板详情缓存 60 秒。刷新后获取新配置，无需重新发布小程序。

不要将 `seed` 作为日常模板发布方式，它会覆盖同 ID 的初始模板和配置。

## 回滚与备份

```sh
cd /opt/yingji-test
bash scripts/rollback.sh
```

生产对应 `/opt/yingji-production`。回滚只恢复应用镜像，不恢复数据库；数据库变更应向后兼容。数据库备份由现有 RDS 运维体系负责，应用镜像不能替代数据备份。首次从 MySQL 切换后不允许普通镜像回滚到旧引擎版本；原 MySQL 数据目录保留用于核对。

本地 `.env.deploy.local` 用于填写两套连接凭据，已被 Git 忽略。同步到服务器 `.env.next`（600）后，下次发布切换配置；失败自动恢复发布前 `.env`。发布前配置备份位于该次 release 的 `env.before`，权限 600，勿对外提供。
