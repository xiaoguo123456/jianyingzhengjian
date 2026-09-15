// Package migration 按版本执行 SQL；记录校验和，失败后保留脏状态等待人工处理。
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed sql/*.sql
var files embed.FS

func Up(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK('yingji_schema', 30)").Scan(&locked); err != nil || locked != 1 {
		return fmt.Errorf("无法取得迁移锁")
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK('yingji_schema')")
	if _, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version varchar(100) PRIMARY KEY, checksum varchar(64) NOT NULL, dirty boolean NOT NULL, applied_at datetime(3) NOT NULL)"); err != nil {
		return err
	}
	entries, err := files.ReadDir("sql")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		raw, _ := files.ReadFile("sql/" + name)
		sum := fmt.Sprintf("%x", sha256.Sum256(raw))
		var old string
		var dirty bool
		err = conn.QueryRowContext(ctx, "SELECT checksum, dirty FROM schema_migrations WHERE version=?", name).Scan(&old, &dirty)
		if err == nil {
			if old != sum || dirty {
				return fmt.Errorf("迁移 %s 已修改或处于失败状态，需人工核对数据库", name)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum,dirty,applied_at) VALUES(?,?,true,NOW(3))", name, sum); err != nil {
			return err
		}
		for _, stmt := range strings.Split(string(raw), ";") {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err = conn.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("迁移 %s 失败: %w", name, err)
			}
		}
		if _, err = conn.ExecContext(ctx, "UPDATE schema_migrations SET dirty=false,applied_at=NOW(3) WHERE version=?", name); err != nil {
			return err
		}
	}
	return nil
}
