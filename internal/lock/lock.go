package lock

import (
	"context"
	"time"
)

type DistributedLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (unlock func(), acquired bool, err error)
}
