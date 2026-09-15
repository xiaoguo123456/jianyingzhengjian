// migrate: `up` applies the schema (GORM AutoMigrate in V1, see migrations/README.md), `seed` loads catalogue + admin.
package main

import (
	"context"
	"fmt"
	"os"

	"yingji/backend/internal/app"
	"yingji/backend/internal/migration"
	"yingji/backend/internal/seed"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: migrate up|seed")
		os.Exit(2)
	}
	ctx := context.Background()
	a, err := app.New(ctx)
	if err != nil {
		panic(err)
	}
	defer a.Close()
	switch os.Args[1] {
	case "up":
		db, err := a.DB.DB()
		if err != nil {
			panic(err)
		}
		if err := migration.Up(ctx, db); err != nil {
			panic(err)
		}
		fmt.Println("数据库迁移完成")
	case "seed":
		dir := "seed/assets"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		if err := seed.Run(ctx, a.DB, a.Store, dir, a.Cfg.AdminInitUser, a.Cfg.AdminInitPassword); err != nil {
			panic(err)
		}
		a.Catalogue.InvalidateHome(ctx)
		fmt.Println("seed done")
	default:
		fmt.Println("unknown command")
		os.Exit(2)
	}
}
