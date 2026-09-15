// migrate: `up` applies the schema (GORM AutoMigrate in V1, see migrations/README.md), `seed` loads catalogue + admin.
package main

import (
	"context"
	"fmt"
	"os"

	"yingji/backend/internal/app"
	"yingji/backend/internal/domain"
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
		if a.Cfg.IsProd() {
			fmt.Println("refusing AutoMigrate in prod; use SQL migrations (migrations/README.md)")
			os.Exit(1)
		}
		if err := a.DB.AutoMigrate(domain.AllModels()...); err != nil {
			panic(err)
		}
		fmt.Println("schema up to date")
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
