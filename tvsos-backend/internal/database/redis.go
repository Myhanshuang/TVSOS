package database

import (
	"context"
	"fmt"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RedisClient *redis.Client

// InitRedis 初始化 Redis 客户端。连接失败时降级为仅内存模式，避免阻塞服务启动。
func InitRedis() {
	if !config.VP.GetBool("redis.enabled") {
		Logger.Logger.Info("Redis 未启用，运行态路径将仅保存在进程内存中")
		return
	}

	host := config.VP.GetString("redis.host")
	port := config.VP.GetString("redis.port")
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     config.VP.GetString("redis.password"),
		DB:           config.VP.GetInt("redis.db"),
		PoolSize:     config.VP.GetInt("redis.pool_size"),
		MinIdleConns: config.VP.GetInt("redis.min_idle_conns"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		Logger.Logger.Warn("Redis 连接失败，降级为仅内存模式", zap.Error(err))
		_ = client.Close()
		return
	}

	RedisClient = client
	Logger.Logger.Info("Redis 连接成功")
}
