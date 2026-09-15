package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yingji/backend/internal/app"
	httptransport "yingji/backend/internal/transport/http"
	"yingji/backend/internal/transport/queue"
)

func main() {
	ctx := context.Background()
	a, err := app.New(ctx)
	if err != nil {
		panic(err)
	}
	defer a.Close()

	enq := queue.NewEnqueuer(a.Cfg)
	defer enq.Close()
	a.Task.SetEnqueuer(enq)

	srv := &http.Server{Addr: a.Cfg.HTTPAddr, Handler: httptransport.New(a), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		a.Log.Info("api listening", "addr", a.Cfg.HTTPAddr, "env", a.Cfg.AppEnv, "storage", a.Cfg.StorageDriver, "gen", a.Cfg.GenProviderDefault)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Log.Error("server", "err", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
