package easycache

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, val any, expires time.Duration) error
	Get(ctx context.Context, key string) (any, error)
	Del(ctx context.Context, key string) error
}
