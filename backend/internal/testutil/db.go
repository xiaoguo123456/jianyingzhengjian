// Package testutil 使用本地 PostgreSQL CI 数据库；每个测试拥有独立 schema。
package testutil

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
)

var serial atomic.Uint64
var nonWord = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func DB(t *testing.T) *gorm.DB {
	db := EmptyDB(t)
	if e := db.AutoMigrate(domain.AllModels()...); e != nil {
		t.Fatal(e)
	}
	return db
}
func EmptyDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("设置 TEST_DATABASE_URL 可运行 PostgreSQL 集成测试")
	}
	u, e := url.Parse(dsn)
	if e != nil {
		t.Fatal("测试数据库地址无效")
	}
	if !strings.HasSuffix(u.Path, "_ci") {
		t.Fatal("测试工具仅允许数据库名称以 _ci 结尾，禁止连接业务库")
	}
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	schema := fmt.Sprintf("test_%s_%d", nonWord.ReplaceAllString(t.Name(), "_"), serial.Add(1))
	if len(schema) > 60 {
		schema = fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), serial.Add(1))
	}
	schema = strings.ToLower(schema)
	if e = root.Exec(`CREATE SCHEMA "` + schema + `"`).Error; e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent), NowFunc: func() time.Time { return time.Now().UTC() }})
	if e != nil {
		t.Fatal(e)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(2)
	t.Cleanup(func() { sqlDB.Close(); root.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); r, _ := root.DB(); r.Close() })
	return db
}
func Truncate(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, tbl := range []string{"users", "user_identities", "credit_accounts", "credit_ledger", "ad_sessions", "photos", "tasks", "works", "favorites", "shares", "share_opens", "events", "app_configs", "specs", "templates", "categories"} {
		if e := db.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; e != nil {
			t.Fatal(e)
		}
	}
}
func Runtime(t *testing.T, db *gorm.DB, overrides map[string]any) *config.Runtime {
	t.Helper()
	rt := config.NewRuntime(db, nil)
	for k, v := range overrides {
		if e := rt.Set(t.Context(), k, v, "test"); e != nil {
			t.Fatal(e)
		}
	}
	return rt
}
