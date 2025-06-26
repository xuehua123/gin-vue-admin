package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败: %v", err)
	}

	fmt.Println("🚀 开始清理旧配对架构数据...")

	patterns := []string{
		"mqtt:active:*",
		"mqtt:role:*",
		"mqtt:seq:*",
		"mqtt:client_ref:*",
	}

	total := 0
	for _, pattern := range patterns {
		keys, err := rdb.Keys(ctx, pattern).Result()
		if err != nil {
			log.Printf("获取键失败 %s: %v", pattern, err)
			continue
		}

		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				log.Printf("删除键失败 %s: %v", pattern, err)
				continue
			}
			fmt.Printf("清理 %s: %d 个键\n", pattern, len(keys))
			total += len(keys)
		}
	}

	// 设置新架构标记
	info := map[string]interface{}{
		"version": "2.0",
		"time":    time.Now().Unix(),
	}
	rdb.HMSet(ctx, "pairing:architecture:info", info)

	fmt.Printf("✅ 迁移完成，清理了 %d 个旧键\n", total)
}
