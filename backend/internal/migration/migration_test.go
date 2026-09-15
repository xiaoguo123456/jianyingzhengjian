package migration

import (
	"context"
	"testing"
	"yingji/backend/internal/testutil"
)

func TestMigrationRejectsUnversionedExistingSchema(t *testing.T) {
	db := testutil.DB(t)
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	// 此数据库已经由测试工具自动建表，迁移不可静默接管并修改旧表。
	if e = Up(context.Background(), sqlDB); e == nil {
		t.Fatal("不应静默接管未记录版本的旧数据库")
	}
	if e = Up(context.Background(), sqlDB); e == nil {
		t.Fatal("不应自动重试失败的 DDL")
	}
}

func TestMigrationFreshAndRepeat(t *testing.T) {
	db := testutil.EmptyDB(t)
	conn, _ := db.DB()
	for i := 0; i < 2; i++ {
		if err := Up(t.Context(), conn); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Table("templates").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schema_migrations SET checksum='tampered'").Error; err != nil {
		t.Fatal(err)
	}
	if err := Up(t.Context(), conn); err == nil {
		t.Fatal("必须拒绝被修改的历史迁移")
	}
}
