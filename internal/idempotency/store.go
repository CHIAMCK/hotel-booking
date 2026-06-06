package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/redis/go-redis/v9"
)

// BookingStore caches successful booking creates by idempotency key for safe HTTP retries.
type BookingStore interface {
	GetBooking(ctx context.Context, idempotencyKey string) (*models.Booking, error)
	SetBooking(ctx context.Context, idempotencyKey string, b models.Booking, ttl time.Duration) error
}

const redisKeyPrefix = "booking:idempotency:"

// RedisBookingStore persists idempotent replay payloads in Redis (not in Postgres).
type RedisBookingStore struct {
	client *redis.Client
}

func NewRedisBookingStore(client *redis.Client) *RedisBookingStore {
	return &RedisBookingStore{client: client}
}

func redisKey(idempotencyKey string) string {
	return redisKeyPrefix + idempotencyKey
}

func (s *RedisBookingStore) GetBooking(ctx context.Context, idempotencyKey string) (*models.Booking, error) {
	val, err := s.client.Get(ctx, redisKey(idempotencyKey)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var b models.Booking
	if err := json.Unmarshal(val, &b); err != nil {
		return nil, fmt.Errorf("decode cached booking: %w", err)
	}
	return &b, nil
}

func (s *RedisBookingStore) SetBooking(ctx context.Context, idempotencyKey string, b models.Booking, ttl time.Duration) error {
	payload, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("encode booking: %w", err)
	}
	return s.client.Set(ctx, redisKey(idempotencyKey), payload, ttl).Err()
}
