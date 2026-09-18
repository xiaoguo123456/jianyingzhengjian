// Package redisx 隔离映己缓存和 Asynq 队列，包括脚本内使用的键前缀及订阅频道。
package redisx

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
	"yingji/backend/internal/config"
)

func client(c *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: c.RedisAddr, Username: c.RedisUsername, Password: c.RedisPassword, DB: c.RedisDB, PoolSize: 4, MinIdleConns: 0, DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second})
}
func New(c *config.Config) *redis.Client {
	r := client(c)
	r.AddHook(hook{prefix: c.RedisPrefix})
	return r
}

type QueueOpt struct{ Cfg *config.Config }

func (o QueueOpt) MakeRedisClient() interface{} {
	r := client(o.Cfg)
	r.AddHook(hook{prefix: o.Cfg.RedisPrefix, queue: true})
	return &queueClient{Client: r, prefix: o.Cfg.RedisPrefix}
}

type queueClient struct {
	*redis.Client
	prefix string
}

func (c *queueClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	out := make([]string, len(channels))
	for i, v := range channels {
		out[i] = c.prefix + v
	}
	return c.Client.Subscribe(ctx, out...)
}

type hook struct {
	prefix string
	queue  bool
}

func (h hook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := h.apply(cmd); err != nil {
			return err
		}
		return next(ctx, cmd)
	}
}
func (h hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			if err := h.apply(cmd); err != nil {
				return err
			}
		}
		return next(ctx, cmds)
	}
}
func (h hook) apply(cmd redis.Cmder) error {
	args := cmd.Args()
	if h.queue {
		// Asynq 0.26 的 Lua 通过 KEYS/ARGV 接收完整键和动态键前缀；不修改消息二进制或脚本文本。
		for i := 1; i < len(args); i++ {
			if s, ok := args[i].(string); ok && strings.HasPrefix(s, "asynq:") {
				args[i] = h.prefix + s
			}
		}
		return nil
	}
	positions := []int{}
	switch cmd.Name() {
	case "get", "set", "setnx", "incr", "expire", "pexpire", "ttl", "pttl", "exists":
		positions = []int{1}
	case "del", "unlink":
		for i := 1; i < len(args); i++ {
			positions = append(positions, i)
		}
	case "ping", "hello", "auth", "select", "client", "multi", "exec":
		return nil
	default:
		return fmt.Errorf("未声明命名空间规则的 Redis 命令：%s", cmd.Name())
	}
	for _, i := range positions {
		if i < len(args) {
			s, ok := args[i].(string)
			if !ok {
				return fmt.Errorf("Redis key 类型无效")
			}
			args[i] = h.prefix + s
		}
	}
	return nil
}
