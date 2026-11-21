package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/model"
)

type RedisStorage struct {
	client *redis.Client
	logger zerolog.Logger
}

// New creates a RedisStorage from an existing client (useful for tests).
func New(client *redis.Client, logger zerolog.Logger) *RedisStorage {
	return &RedisStorage{
		client: client,
		logger: logger.With().Str("component", "storage").Logger(),
	}
}

func NewRedisStorage(lc fx.Lifecycle, cfg *config.Config, logger zerolog.Logger) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	s := &RedisStorage{
		client: client,
		logger: logger.With().Str("component", "storage").Logger(),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("redis ping failed: %w", err)
			}
			s.logger.Info().Str("addr", cfg.Redis.Addr).Msg("redis connected")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})

	return s
}

type shareMeta struct {
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	ContentSize int               `json:"content_size"`
	StoredSize  int               `json:"stored_size"`
	Hash        string            `json:"hash"`
	CreatedAt   int64             `json:"created_at"`
	ExpireAt    int64             `json:"expire_at"`
	Creator     string            `json:"creator"`
	Type        string            `json:"type"`
	Files       []model.ShareFile `json:"files,omitempty"`
}

func redisKey(key string) string {
	return "share:" + key
}

func (s *RedisStorage) Save(ctx context.Context, share *model.Share, ttl time.Duration) error {
	meta := shareMeta{
		Key:         share.Key,
		Name:        share.Name,
		ContentType: share.ContentType,
		ContentSize: share.ContentSize,
		StoredSize:  share.StoredSize,
		Hash:        share.Hash,
		CreatedAt:   share.CreatedAt,
		ExpireAt:    share.ExpireAt,
		Creator:     share.Creator,
		Type:        share.Type,
		Files:       share.Files,
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	rk := redisKey(share.Key)
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, rk, "meta", metaBytes, "data", share.Content)
	pipe.Expire(ctx, rk, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStorage) Get(ctx context.Context, key string) (*model.Share, error) {
	rk := redisKey(key)
	result, err := s.client.HGetAll(ctx, rk).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	var meta shareMeta
	if err := json.Unmarshal([]byte(result["meta"]), &meta); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}

	return &model.Share{
		Key:         meta.Key,
		Name:        meta.Name,
		Content:     []byte(result["data"]),
		ContentType: meta.ContentType,
		ContentSize: meta.ContentSize,
		StoredSize:  meta.StoredSize,
		Hash:        meta.Hash,
		CreatedAt:   meta.CreatedAt,
		ExpireAt:    meta.ExpireAt,
		Creator:     meta.Creator,
		Type:        meta.Type,
		Files:       meta.Files,
	}, nil
}

func (s *RedisStorage) Delete(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Del(ctx, redisKey(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *RedisStorage) Exists(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, redisKey(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *RedisStorage) GetHash(ctx context.Context, key string) (string, error) {
	metaStr, err := s.client.HGet(ctx, redisKey(key), "meta").Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var meta shareMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return "", fmt.Errorf("unmarshal meta: %w", err)
	}
	return meta.Hash, nil
}

func (s *RedisStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Expire(ctx, redisKey(key), ttl).Err()
}

func (s *RedisStorage) HealthCheck(ctx context.Context) (int64, error) {
	start := time.Now()
	err := s.client.Ping(ctx).Err()
	return time.Since(start).Milliseconds(), err
}

func (s *RedisStorage) Stats(ctx context.Context) (StorageStats, error) {
	stats := StorageStats{Backend: "redis"}
	var cursor uint64
	var keys []string
	for {
		batch, next, err := s.client.Scan(ctx, cursor, "share:*", 500).Result()
		if err != nil {
			return stats, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	stats.ActiveShares = int64(len(keys))
	if len(keys) == 0 {
		return stats, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HStrLen(ctx, k, "data")
	}
	_, _ = pipe.Exec(ctx)
	for _, cmd := range cmds {
		if n, err := cmd.Result(); err == nil && n > 0 {
			stats.TotalBytes += n
		}
	}
	return stats, nil
}
