package main

import (
	"github.com/hibiken/asynq"
	"net/http"
	"os"
	"time"
	"yingji/backend/internal/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		c, e := config.Load()
		if e != nil {
			os.Exit(1)
		}
		i := asynq.NewInspector(asynq.RedisClientOpt{Addr: c.RedisAddr, Password: c.RedisPassword, DB: c.RedisDB})
		defer i.Close()
		servers, e := i.Servers()
		if e != nil {
			os.Exit(1)
		}
		host, _ := os.Hostname()
		for _, s := range servers {
			if s.Host == host && s.Status == "active" {
				return
			}
		}
		os.Exit(1)
	}
	c := http.Client{Timeout: 3 * time.Second}
	r, e := c.Get("http://127.0.0.1:8080/readyz")
	if e != nil {
		os.Exit(1)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		os.Exit(1)
	}
}
