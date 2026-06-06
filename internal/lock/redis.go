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

func (l *RedisLock) TryLock(ctx context.Context, key string, exp time.Duration) (func(), bool, error) {
	// SET key NX with expiration so the lock is released if the holder crashes.
	acquired, err := l.client.SetNX(ctx, key, "1", exp).Result()
	if err != nil {
		return nil, false, err
	}
	if !acquired {
		return nil, false, nil
	}

	unlock := func() {
		_ = l.client.Del(context.Background(), key).Err()
	}

	return unlock, true, nil
}
