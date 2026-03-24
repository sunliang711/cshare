package storage

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/model"
)

type memEntry struct {
	share    *model.Share
	expireAt time.Time
}

type MemoryStorage struct {
	mu     sync.RWMutex
	items  map[string]*memEntry
	logger zerolog.Logger
	stopCh chan struct{}
}

func NewMemoryStorage(lc fx.Lifecycle, logger zerolog.Logger) *MemoryStorage {
	s := &MemoryStorage{
		items:  make(map[string]*memEntry),
		logger: logger.With().Str("component", "storage").Logger(),
		stopCh: make(chan struct{}),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go s.cleanupLoop()
			s.logger.Info().Msg("memory storage started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(s.stopCh)
			return nil
		},
	})

	return s
}

func (s *MemoryStorage) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evict()
		}
	}
}

func (s *MemoryStorage) evict() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, e := range s.items {
		if now.After(e.expireAt) {
			delete(s.items, k)
		}
	}
}

func (s *MemoryStorage) Save(_ context.Context, share *model.Share, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := *share
	copied.Content = append([]byte(nil), share.Content...)

	s.items[share.Key] = &memEntry{
		share:    &copied,
		expireAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *MemoryStorage) Get(_ context.Context, key string) (*model.Share, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expireAt) {
		return nil, nil
	}

	copied := *entry.share
	copied.Content = append([]byte(nil), entry.share.Content...)
	return &copied, nil
}

func (s *MemoryStorage) Delete(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expireAt) {
		delete(s.items, key)
		return false, nil
	}
	delete(s.items, key)
	return true, nil
}

func (s *MemoryStorage) Exists(_ context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expireAt) {
		return false, nil
	}
	return true, nil
}

func (s *MemoryStorage) GetHash(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.items[key]
	if !ok || time.Now().After(entry.expireAt) {
		return "", nil
	}
	return entry.share.Hash, nil
}

func (s *MemoryStorage) Expire(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.items[key]; ok {
		entry.expireAt = time.Now().Add(ttl)
	}
	return nil
}
