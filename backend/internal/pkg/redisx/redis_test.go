package redisx

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"os"
	"strings"
	"testing"
	"time"
	"yingji/backend/internal/config"
)

func TestEnvironmentIsolation(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("需要本地 Redis 集成测试地址")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("仅允许本地验收实例")
	}
	ctx := context.Background()
	raw := redis.NewClient(&redis.Options{Addr: addr, DB: 12})
	defer raw.Close()
	// DB12 仅属于本地测试；仍仅清理本测试的项目命名空间。
	stamp := time.Now().Format("150405.000000000")
	prefixes := []string{"yingji:test:" + stamp + ":", "yingji:prod:" + stamp + ":"}
	defer func() {
		for _, p := range prefixes {
			keys, _ := raw.Keys(ctx, p+"*").Result()
			if len(keys) > 0 {
				raw.Del(ctx, keys...)
			}
		}
	}()
	done := make(chan string, 4)
	for _, p := range prefixes {
		c := &config.Config{RedisAddr: addr, RedisDB: 12, RedisPrefix: p}
		app := New(c)
		defer app.Close()
		if err := app.Set(ctx, "cfg:value", "asynq:这是原始内容", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		got, err := raw.Get(ctx, p+"cfg:value").Result()
		if err != nil || got != "asynq:这是原始内容" {
			t.Fatal("缓存命名空间错误或修改了数据")
		}
		opt := QueueOpt{Cfg: c}
		srv := asynq.NewServer(opt, asynq.Config{Concurrency: 1, ShutdownTimeout: time.Second})
		defer srv.Shutdown()
		tag := p
		mux := asynq.NewServeMux()
		mux.HandleFunc("verify", func(ctx context.Context, task *asynq.Task) error {
			var v string
			_ = json.Unmarshal(task.Payload(), &v)
			if v != tag {
				t.Error("工作进程读取了其他环境的任务")
			}
			done <- v
			return nil
		})
		if err := srv.Start(mux); err != nil {
			t.Fatal(err)
		}
		scheduler := asynq.NewScheduler(opt, nil)
		defer scheduler.Shutdown()
		if _, err := scheduler.Register("@every 1h", asynq.NewTask("verify", []byte(`"unused"`))); err != nil {
			t.Fatal(err)
		}
		if err := scheduler.Start(); err != nil {
			t.Fatal(err)
		}
		client := asynq.NewClient(opt)
		defer client.Close()
		payload, _ := json.Marshal(p)
		if _, err := client.Enqueue(asynq.NewTask("verify", payload), asynq.TaskID("same-id")); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			t.Fatal("队列任务未完成")
		}
	}
	for _, p := range prefixes {
		ins := asynq.NewInspector(QueueOpt{Cfg: &config.Config{RedisAddr: addr, RedisDB: 12, RedisPrefix: p}})
		servers, e := ins.Servers()
		ins.Close()
		if e != nil || len(servers) != 1 {
			t.Fatalf("Worker 心跳隔离失败：%d, %v", len(servers), e)
		}
	}
	bare, e := raw.Keys(ctx, "asynq:*").Result()
	if e != nil || len(bare) != 0 {
		t.Fatalf("出现未加项目前缀的 Asynq key：%v", bare)
	}
}
