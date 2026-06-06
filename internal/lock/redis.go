package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLock struct {
	client *redis.Client
}

func NewRedisLock(client *redis.Client) *RedisLock {
	return &RedisLock{client: client}
}

func (l *RedisLock) TryLock(ctx context.Context, key string, ttl time.Duration) (func(), bool, error) {
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	acquired, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !acquired {
		return nil, false, nil
	}

	unlock := func() {
		script := redis.NewScript(`
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			end
			return 0
		`)
		_ = script.Run(context.Background(), l.client, []string{key}, token).Err()
	}

	return unlock, true, nil
}
