package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/model"
)

type Storage interface {
	Save(ctx context.Context, share *model.Share, ttl time.Duration) error
	Get(ctx context.Context, key string) (*model.Share, error)
	Delete(ctx context.Context, key string) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
	GetHash(ctx context.Context, key string) (string, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

var Module = fx.Options(
	fx.Provide(NewStorage),
)

func NewStorage(lc fx.Lifecycle, cfg *config.Config, logger zerolog.Logger) (Storage, error) {
	switch cfg.Storage.Type {
	case "memory":
		return NewMemoryStorage(lc, logger), nil
	case "redis", "":
		return NewRedisStorage(lc, cfg, logger), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Storage.Type)
	}
}
