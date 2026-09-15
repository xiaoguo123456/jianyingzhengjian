package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"yingji/backend/internal/app"
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

	srv, sched := queue.NewServer(a, enq)
	go func() {
		if err := sched.Start(); err != nil {
			a.Log.Error("scheduler", "err", err)
		}
	}()
	go func() {
		a.Log.Info("worker started", "concurrency", a.Cfg.GenConcurrency)
		if err := srv.Start(queue.Mux(a, enq)); err != nil {
			a.Log.Error("worker", "err", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	sched.Shutdown()
	srv.Shutdown()
}
