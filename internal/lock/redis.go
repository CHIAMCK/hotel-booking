package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLock struct {
	client *redis.Client
}

func NewRedisLock(client *redis.Client) *RedisLock {
	return &RedisLock{client: client}
}

func (l *RedisLock) TryLock(ctx context.Context, key string, exp time.Duration) (bool, error) {
	acquired, err := l.client.SetNX(ctx, key, "1", exp).Result()
	if err != nil {
		return false, err
	}

	return acquired, nil
}

func (l *RedisLock) Unlock(ctx context.Context, key string) error {
	return l.client.Del(ctx, key).Err()
}
