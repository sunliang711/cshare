package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/keygen"
	"crossshare-server/internal/model"
	"crossshare-server/internal/storage"
)

var Module = fx.Options(
	fx.Provide(NewShareService),
)

type ShareService struct {
	storage *storage.RedisStorage
	config  *config.Config
	logger  zerolog.Logger
}

func NewShareService(s *storage.RedisStorage, cfg *config.Config, logger zerolog.Logger) *ShareService {
	return &ShareService{
		storage: s,
		config:  cfg,
		logger:  logger.With().Str("component", "service").Logger(),
	}
}

type PushTextRequest struct {
	Text        string
	TTL         int
	Filename    string
	ContentType string
	Creator     string
}

type PushBinaryRequest struct {
	Data        []byte
	TTL         int
	Filename    string
	ContentType string
	Creator     string
}

func (s *ShareService) PushText(ctx context.Context, req *PushTextRequest) (*model.PushResult, error) {
	content := []byte(req.Text)

	if int64(len(content)) > s.config.Business.TextJSONLimit {
		return nil, apperr.ErrPayloadTooLarge
	}

	ttl := s.resolveTTL(req.TTL)
	if ttl < 0 {
		return nil, apperr.ErrInvalidTTL
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}

	key, exists, err := s.resolveKey(ctx, content)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	expireAt := now + int64(ttl)

	if exists {
		if err := s.storage.Expire(ctx, key, time.Duration(ttl)*time.Second); err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("failed to refresh ttl")
			return nil, apperr.ErrStorageInternal
		}
		s.logger.Debug().Str("key", key).Msg("dedup hit, refreshed ttl")
		return &model.PushResult{
			Key:      key,
			TTL:      ttl,
			Size:     len(content),
			Type:     "text",
			ExpireAt: expireAt,
		}, nil
	}

	share := &model.Share{
		Key:         key,
		Name:        sanitizeFilename(req.Filename),
		Content:     content,
		ContentType: contentType,
		ContentSize: len(content),
		Hash:        hashContent(content),
		CreatedAt:   now,
		ExpireAt:    expireAt,
		Creator:     req.Creator,
		Type:        "text",
	}

	if err := s.storage.Save(ctx, share, time.Duration(ttl)*time.Second); err != nil {
		s.logger.Error().Err(err).Str("key", key).Msg("failed to save share")
		return nil, apperr.ErrStorageInternal
	}

	return &model.PushResult{
		Key:      key,
		TTL:      ttl,
		Size:     share.ContentSize,
		Type:     "text",
		ExpireAt: share.ExpireAt,
	}, nil
}

func (s *ShareService) PushBinary(ctx context.Context, req *PushBinaryRequest) (*model.PushResult, error) {
	if int64(len(req.Data)) > s.config.Business.BinaryPushLimit {
		return nil, apperr.ErrPayloadTooLarge
	}

	ttl := s.resolveTTL(req.TTL)
	if ttl < 0 {
		return nil, apperr.ErrInvalidTTL
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	key, exists, err := s.resolveKey(ctx, req.Data)
	if err != nil {
		return nil, err
	}

	filename := sanitizeFilename(req.Filename)
	now := time.Now().Unix()
	expireAt := now + int64(ttl)

	if exists {
		if err := s.storage.Expire(ctx, key, time.Duration(ttl)*time.Second); err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("failed to refresh ttl")
			return nil, apperr.ErrStorageInternal
		}
		s.logger.Debug().Str("key", key).Msg("dedup hit, refreshed ttl")
		result := &model.PushResult{
			Key:      key,
			TTL:      ttl,
			Size:     len(req.Data),
			Type:     "binary",
			ExpireAt: expireAt,
		}
		if filename != "" {
			result.Filename = filename
		}
		return result, nil
	}

	share := &model.Share{
		Key:         key,
		Name:        filename,
		Content:     req.Data,
		ContentType: contentType,
		ContentSize: len(req.Data),
		Hash:        hashContent(req.Data),
		CreatedAt:   now,
		ExpireAt:    expireAt,
		Creator:     req.Creator,
		Type:        "binary",
	}

	if err := s.storage.Save(ctx, share, time.Duration(ttl)*time.Second); err != nil {
		s.logger.Error().Err(err).Str("key", key).Msg("failed to save share")
		return nil, apperr.ErrStorageInternal
	}

	result := &model.PushResult{
		Key:      key,
		TTL:      ttl,
		Size:     share.ContentSize,
		Type:     "binary",
		ExpireAt: share.ExpireAt,
	}
	if filename != "" {
		result.Filename = filename
	}
	return result, nil
}

func (s *ShareService) Pull(ctx context.Context, key string) (*model.Share, error) {
	share, err := s.storage.Get(ctx, key)
	if err != nil {
		s.logger.Error().Err(err).Str("key", key).Msg("storage get failed")
		return nil, apperr.ErrStorageInternal
	}
	if share == nil {
		return nil, apperr.ErrNotFound
	}
	return share, nil
}

func (s *ShareService) Delete(ctx context.Context, key string) (bool, error) {
	deleted, err := s.storage.Delete(ctx, key)
	if err != nil {
		s.logger.Error().Err(err).Str("key", key).Msg("storage delete failed")
		return false, apperr.ErrStorageInternal
	}
	return deleted, nil
}

func (s *ShareService) resolveTTL(requested int) int {
	if requested == 0 {
		return s.config.Business.DefaultTTL
	}
	if requested < 0 || requested > s.config.Business.MaxTTL {
		return -1
	}
	return requested
}

const (
	minKeyLen = 6
	maxKeyLen = 12
)

// resolveKey derives a deterministic key from content hash (base62-encoded prefix).
// Returns the key, whether identical content already exists, and any error.
// On hash-prefix collision with different content, the prefix length is extended.
func (s *ShareService) resolveKey(ctx context.Context, content []byte) (string, bool, error) {
	hashBytes := sha256.Sum256(content)
	hashHex := fmt.Sprintf("%x", hashBytes)
	for length := minKeyLen; length <= maxKeyLen; length++ {
		key := keygen.FromHash(hashBytes[:], length)
		storedHash, err := s.storage.GetHash(ctx, key)
		if err != nil {
			return "", false, apperr.ErrStorageInternal
		}
		if storedHash == "" {
			return key, false, nil
		}
		if storedHash == hashHex {
			return key, true, nil
		}
	}
	return "", false, apperr.ErrStorageInternal
}

func hashContent(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func sanitizeFilename(name string) string {
	if name == "" {
		return ""
	}
	cleaned := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '/' || c == '\\' || c == '\x00' {
			continue
		}
		cleaned = append(cleaned, c)
	}
	result := string(cleaned)
	if len(result) > 255 {
		result = result[:255]
	}
	return result
}
