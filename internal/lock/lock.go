package lock

import (
	"context"
	"time"
)

type DistributedLock interface {
	TryLock(ctx context.Context, key string, exp time.Duration) (acquired bool, err error)
	Unlock(ctx context.Context, key string) error
}
