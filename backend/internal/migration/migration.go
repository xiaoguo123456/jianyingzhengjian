// Package migration 在 PostgreSQL 事务中执行版本化 SQL，并校验历史文件摘要。
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed postgres/*.sql
var files embed.FS

func Up(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext(current_database()), hashtext('yingji_schema'))").Scan(&locked); err != nil || !locked {
		return fmt.Errorf("无法取得数据库迁移锁")
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext(current_database()), hashtext('yingji_schema'))")
	if _, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version varchar(100) PRIMARY KEY, checksum varchar(64) NOT NULL, applied_at timestamptz NOT NULL)"); err != nil {
		return err
	}
	entries, err := files.ReadDir("postgres")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		raw, e := files.ReadFile("postgres/" + name)
		if e != nil {
			return e
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(raw))
		var old string
		err = conn.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", name).Scan(&old)
		if err == nil {
			if old != sum {
				return fmt.Errorf("迁移 %s 已修改，停止发布", name)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		tx, e := conn.BeginTx(ctx, nil)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, string(raw)); e == nil {
			_, e = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum,applied_at) VALUES($1,$2,NOW())", name, sum)
		}
		if e != nil {
			_ = tx.Rollback()
			return fmt.Errorf("迁移 %s 失败，已回滚: %w", name, e)
		}
		if e = tx.Commit(); e != nil {
			return e
		}
	}
	return nil
}
