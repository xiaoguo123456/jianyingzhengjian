# Migrations

V1.0 creates the schema with GORM `AutoMigrate` (`go run ./cmd/migrate up`), because the
model definitions in `internal/domain/models.go` are the single source of truth while the
schema is still moving.

From the first production release onward, schema changes are numbered SQL files in this
folder applied with golang-migrate (`NNNN_name.up.sql` / `.down.sql`), and `AutoMigrate`
is disabled for `APP_ENV=prod`. Drop/rename changes go through the two-release pattern in
docs/DEPLOYMENT.md §4.
