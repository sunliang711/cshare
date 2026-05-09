package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"
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
	fx.Provide(
		NewShareService,
		NewP2PService,
	),
)

type ShareService struct {
	storage storage.Storage
	config  *config.Config
	logger  zerolog.Logger
}

func NewShareService(s storage.Storage, cfg *config.Config, logger zerolog.Logger) *ShareService {
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

type PushFileInput struct {
	Data        []byte
	Filename    string
	ContentType string
}

type PushFilesRequest struct {
	Files   []PushFileInput
	TTL     int
	Name    string
	Creator string
}

type PulledFile struct {
	Data        []byte
	Filename    string
	ContentType string
	Size        int
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
			Key:        key,
			TTL:        ttl,
			Size:       len(content),
			StoredSize: len(content),
			Type:       "text",
			ExpireAt:   expireAt,
		}, nil
	}

	share := &model.Share{
		Key:         key,
		Name:        sanitizeFilename(req.Filename),
		Content:     content,
		ContentType: contentType,
		ContentSize: len(content),
		StoredSize:  len(content),
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
		Key:        key,
		TTL:        ttl,
		Size:       share.ContentSize,
		StoredSize: share.StoredSize,
		Type:       "text",
		ExpireAt:   share.ExpireAt,
	}, nil
}

func (s *ShareService) PushBinary(ctx context.Context, req *PushBinaryRequest) (*model.PushResult, error) {
	return s.PushFiles(ctx, &PushFilesRequest{
		Files: []PushFileInput{
			{
				Data:        req.Data,
				Filename:    req.Filename,
				ContentType: req.ContentType,
			},
		},
		TTL:     req.TTL,
		Creator: req.Creator,
	})
}

func (s *ShareService) PushFiles(ctx context.Context, req *PushFilesRequest) (*model.PushResult, error) {
	if len(req.Files) == 0 {
		return nil, apperr.ErrInvalidPayload
	}
	if s.config.Business.MaxFilesPerPush > 0 && len(req.Files) > s.config.Business.MaxFilesPerPush {
		return nil, apperr.ErrPayloadTooLarge
	}

	files, contents, totalSize, err := normalizePushFiles(req.Files)
	if err != nil {
		return nil, err
	}
	if int64(totalSize) > s.config.Business.FilesPushLimit {
		return nil, apperr.ErrPayloadTooLarge
	}

	ttl := s.resolveTTL(req.TTL)
	if ttl < 0 {
		return nil, apperr.ErrInvalidTTL
	}

	zipData, err := buildFilesZip(files, contents)
	if err != nil {
		return nil, apperr.ErrStorageInternal
	}

	key, exists, err := s.resolveKey(ctx, zipData)
	if err != nil {
		return nil, err
	}

	filename := resolveFilesShareName(req.Name, files)
	now := time.Now().Unix()
	expireAt := now + int64(ttl)

	if exists {
		if err := s.storage.Expire(ctx, key, time.Duration(ttl)*time.Second); err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("failed to refresh ttl")
			return nil, apperr.ErrStorageInternal
		}
		stored, err := s.storage.Get(ctx, key)
		if err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("failed to load dedup share")
			return nil, apperr.ErrStorageInternal
		}
		storedSize := len(zipData)
		if stored != nil && stored.Type == "files" {
			filename = stored.Name
			files = stored.Files
			totalSize = stored.ContentSize
			storedSize = stored.StoredSize
			if storedSize == 0 {
				storedSize = len(stored.Content)
			}
		}
		s.logger.Debug().Str("key", key).Msg("dedup hit, refreshed ttl")
		result := &model.PushResult{
			Key:        key,
			TTL:        ttl,
			Size:       totalSize,
			StoredSize: storedSize,
			Type:       "files",
			Filename:   filename,
			FileCount:  len(files),
			Files:      files,
			ExpireAt:   expireAt,
		}
		return result, nil
	}

	share := &model.Share{
		Key:         key,
		Name:        filename,
		Content:     zipData,
		ContentType: "application/zip",
		ContentSize: totalSize,
		StoredSize:  len(zipData),
		Hash:        hashContent(zipData),
		CreatedAt:   now,
		ExpireAt:    expireAt,
		Creator:     req.Creator,
		Type:        "files",
		Files:       files,
	}

	if err := s.storage.Save(ctx, share, time.Duration(ttl)*time.Second); err != nil {
		s.logger.Error().Err(err).Str("key", key).Msg("failed to save share")
		return nil, apperr.ErrStorageInternal
	}

	result := &model.PushResult{
		Key:        key,
		TTL:        ttl,
		Size:       share.ContentSize,
		StoredSize: share.StoredSize,
		Type:       "files",
		Filename:   filename,
		FileCount:  len(files),
		Files:      files,
		ExpireAt:   share.ExpireAt,
	}
	return result, nil
}

func (s *ShareService) PullSingleFile(share *model.Share) (*PulledFile, error) {
	if share.Type != "files" || len(share.Files) != 1 {
		return nil, apperr.ErrInvalidPayload
	}

	zr, err := zip.NewReader(bytes.NewReader(share.Content), int64(len(share.Content)))
	if err != nil {
		return nil, apperr.ErrStorageInternal
	}
	if len(zr.File) != 1 {
		return nil, apperr.ErrStorageInternal
	}

	rc, err := zr.File[0].Open()
	if err != nil {
		return nil, apperr.ErrStorageInternal
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, apperr.ErrStorageInternal
	}

	file := share.Files[0]
	return &PulledFile{
		Data:        data,
		Filename:    file.Name,
		ContentType: file.ContentType,
		Size:        file.Size,
	}, nil
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

func normalizePushFiles(inputs []PushFileInput) ([]model.ShareFile, map[string][]byte, int, error) {
	files := make([]model.ShareFile, 0, len(inputs))
	contents := make(map[string][]byte, len(inputs))
	seen := make(map[string]int, len(inputs))
	totalSize := 0
	for i, input := range inputs {
		if len(input.Data) == 0 {
			return nil, nil, 0, apperr.ErrInvalidPayload
		}

		name := sanitizeFilename(input.Filename)
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("file-%d", i+1)
		}
		name = uniqueFilename(name, seen)

		contentType := input.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		files = append(files, model.ShareFile{
			Name:        name,
			ContentType: contentType,
			Size:        len(input.Data),
			Hash:        hashContent(input.Data),
		})
		contents[name] = input.Data
		totalSize += len(input.Data)
	}
	return files, contents, totalSize, nil
}

func buildFilesZip(files []model.ShareFile, contents map[string][]byte) ([]byte, error) {
	ordered := append([]model.ShareFile(nil), files...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	for _, file := range ordered {
		header := &zip.FileHeader{
			Name:   file.Name,
			Method: zip.Deflate,
		}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		w, err := zw.CreateHeader(header)
		if err != nil {
			zw.Close()
			return nil, err
		}
		content, ok := contents[file.Name]
		if !ok {
			zw.Close()
			return nil, fmt.Errorf("missing file content: %s", file.Name)
		}
		if _, err := w.Write(content); err != nil {
			zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func resolveFilesShareName(name string, files []model.ShareFile) string {
	if len(files) == 1 {
		return files[0].Name
	}
	cleaned := sanitizeFilename(name)
	if strings.TrimSpace(cleaned) == "" {
		cleaned = "crossshare-files.zip"
	}
	if !strings.HasSuffix(strings.ToLower(cleaned), ".zip") {
		cleaned += ".zip"
	}
	return cleaned
}

func uniqueFilename(name string, seen map[string]int) string {
	count := seen[name]
	seen[name] = count + 1
	if count == 0 {
		return name
	}
	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 {
		return fmt.Sprintf("%s (%d)", name, count)
	}
	return fmt.Sprintf("%s (%d)%s", name[:dot], count, name[dot:])
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
