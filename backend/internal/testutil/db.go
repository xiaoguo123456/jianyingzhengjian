// Package testutil provides a MySQL-backed test harness. Tests are skipped when
// TEST_MYSQL / TEST_MYSQL_DSN are unset, so `go test ./...` works without a database.
package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
)

const defaultDSN = "root:@tcp(127.0.0.1:3306)/yingji_test?parseTime=true&loc=UTC&charset=utf8mb4"

var nonWord = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// dbNameForPackage suffixes the database name with the test binary's name so that
// packages tested in parallel (the default for `go test ./...`) do not truncate
// each other's tables.
func dbNameForPackage(base string) string {
	pkg := strings.TrimSuffix(filepath.Base(os.Args[0]), ".test")
	pkg = nonWord.ReplaceAllString(pkg, "_")
	if pkg == "" {
		return base
	}
	name := base + "_" + pkg
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// DB returns a migrated, empty database dedicated to the calling package.
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		if os.Getenv("TEST_MYSQL") == "" {
			t.Skip("set TEST_MYSQL=1 (or TEST_MYSQL_DSN) to run database tests")
		}
		dsn = defaultDSN
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse TEST_MYSQL_DSN: %v", err)
	}
	target := dbNameForPackage(cfg.DBName)

	// create the per-package database if it does not exist
	serverCfg := *cfg
	serverCfg.DBName = ""
	admin, err := sql.Open("mysql", serverCfg.FormatDSN())
	if err != nil {
		t.Fatalf("connect to mysql: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE IF NOT EXISTS `" + target + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatalf("create test database %s: %v", target, err)
	}

	cfg.DBName = target
	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger:  gormlogger.Default.LogMode(gormlogger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("open %s: %v", target, err)
	}
	if err := db.AutoMigrate(domain.AllModels()...); err != nil {
		t.Fatalf("migrate %s: %v", target, err)
	}
	Truncate(t, db)
	return db
}

func Truncate(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, tbl := range []string{"users", "user_identities", "credit_accounts", "credit_ledger", "ad_sessions", "photos",
		"tasks", "works", "favorites", "shares", "share_opens", "events", "app_configs", "specs", "templates", "categories"} {
		db.Exec("TRUNCATE TABLE " + tbl)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

// Runtime returns a config.Runtime backed by the test DB with the given overrides.
func Runtime(t *testing.T, db *gorm.DB, overrides map[string]any) *config.Runtime {
	t.Helper()
	rt := config.NewRuntime(db, nil)
	for k, v := range overrides {
		if err := rt.Set(t.Context(), k, v, "test"); err != nil {
			t.Fatalf("set config %s: %v", k, err)
		}
	}
	return rt
}
