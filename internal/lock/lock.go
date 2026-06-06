package lock

import (
	"context"
	"time"
)

type DistributedLock interface {
	TryLock(ctx context.Context, key string, exp time.Duration) (unlock func(), acquired bool, err error)
}
