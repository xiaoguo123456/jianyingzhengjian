// 仅用于生成首版建表脚本；后续变更必须新增经过评审的 SQL。
package main

import (
	"context"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
	"yingji/backend/internal/domain"
)

type output struct{ logger.Interface }

func (output) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	s, _ := fc()
	fmt.Println(s + ";")
}
func main() {
	db, e := gorm.Open(postgres.New(postgres.Config{DSN: "postgres://localhost/yingji?sslmode=disable"}), &gorm.Config{DisableAutomaticPing: true, DryRun: true, Logger: output{logger.Default}})
	if e != nil {
		panic(e)
	}
	if e = db.Migrator().CreateTable(domain.AllModels()...); e != nil {
		panic(e)
	}
}
