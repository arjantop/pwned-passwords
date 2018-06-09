package storage

import (
	"context"
)

type Storage interface {
	Get(ctx context.Context, key string) ([][]byte, error)
}
