# 数据库迁移

正式迁移文件位于 `internal/migration/sql/`，由 `cmd/migrate up` 执行，测试和生产使用同一机制。
首版 `0001_initial.sql` 由现有模型生成。后续按顺序新增 SQL，不修改已执行的文件。

迁移器使用数据库锁、校验和与脏状态记录，失败时停止发布。MySQL DDL 不保证事务回滚，禁止自动删除失败记录后重试。
开发和单元测试仍可使用 GORM AutoMigrate；部署不再使用自动改表。已有 AutoMigrate 数据库需要先人工比对结构后制定接管迁移，不能直接套用首版建表 SQL。
