# 数据库迁移

正式迁移文件位于 `internal/migration/postgres/`，由 `cmd/migrate up` 执行。测试和生产使用 PostgreSQL 16。
首版 SQL 从模型生成，后续按顺序新增 SQL，不修改已执行的文件。

迁移器使用 PostgreSQL advisory lock 防止并发发布，每个版本通过事务执行并记录校验和；失败回滚当前版本，历史摘要不一致时拒绝发布。
`internal/migration/sql/` 仅保留旧 MySQL 版本历史，不再执行。

测试工具可通过 AutoMigrate 创建隔离 schema。已有业务库必须先比对结构并制定迁移方案，不能直接套用首版建表 SQL。
