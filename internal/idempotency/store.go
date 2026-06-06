package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// BookingStore records processed booking idempotency keys for duplicate detection.
type BookingStore interface {
	CheckIdempotent(ctx context.Context, idempotencyKey string) (bool, error)
	SetIdempotent(ctx context.Context, idempotencyKey string, ttl time.Duration) error
}

const redisKeyPrefix = "booking:idempotency:"

// RedisBookingStore persists consumed idempotency keys in Redis (not in Postgres).
type RedisBookingStore struct {
	client *redis.Client
}

func NewRedisBookingStore(client *redis.Client) *RedisBookingStore {
	return &RedisBookingStore{client: client}
}

func redisKey(idempotencyKey string) string {
	return redisKeyPrefix + idempotencyKey
}

func (s *RedisBookingStore) CheckIdempotent(ctx context.Context, idempotencyKey string) (bool, error) {
	n, err := s.client.Exists(ctx, redisKey(idempotencyKey)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *RedisBookingStore) SetIdempotent(ctx context.Context, idempotencyKey string, ttl time.Duration) error {
	return s.client.Set(ctx, redisKey(idempotencyKey), "1", ttl).Err()
}
