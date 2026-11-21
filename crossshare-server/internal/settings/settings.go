package settings

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/fx"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

var Module = fx.Options(
	fx.Provide(NewManager),
)

type Manager struct {
	store     *db.Store
	cipher    *crypto.Cipher
	mu        sync.RWMutex
	current   db.SystemSettings
	listeners []func(db.SystemSettings)
}

func NewManager(store *db.Store, cipher *crypto.Cipher) (*Manager, error) {
	m := &Manager{
		store:   store,
		cipher:  cipher,
		current: db.DefaultSystemSettings(),
	}
	if store != nil {
		loaded, err := store.GetSystemSettings(context.Background(), cipher)
		if err != nil {
			return nil, err
		}
		m.current = loaded
	}
	return m, nil
}

// NewStaticManager builds an in-memory Manager without SQLite (useful for isolated tests).
func NewStaticManager(initial db.SystemSettings) *Manager {
	return &Manager{current: initial}
}

func (m *Manager) Get() db.SystemSettings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) OnChange(fn func(db.SystemSettings)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, fn)
}

func (m *Manager) Update(ctx context.Context, next db.SystemSettings, updatedBy string) error {
	if err := validate(next); err != nil {
		return err
	}
	next.HasClientJWTSecret = next.ClientJWTSecret != ""

	if m.store != nil {
		if err := m.store.UpdateSystemSettings(ctx, m.cipher, next, updatedBy); err != nil {
			return err
		}
	}

	m.mu.Lock()
	m.current = next
	listeners := append(([]func(db.SystemSettings))(nil), m.listeners...)
	m.mu.Unlock()

	for _, fn := range listeners {
		fn(next)
	}
	return nil
}

func (m *Manager) Reload(ctx context.Context) error {
	if m.store == nil {
		return nil
	}
	loaded, err := m.store.GetSystemSettings(ctx, m.cipher)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.current = loaded
	listeners := append(([]func(db.SystemSettings))(nil), m.listeners...)
	m.mu.Unlock()

	for _, fn := range listeners {
		fn(loaded)
	}
	return nil
}

func validate(s db.SystemSettings) error {
	if s.DefaultTTL <= 0 {
		return errors.New("default_ttl must be > 0")
	}
	if s.MaxTTL < s.DefaultTTL {
		return errors.New("max_ttl must be >= default_ttl")
	}
	if s.TextJSONLimit < 1024 {
		return errors.New("text_json_limit must be >= 1024 bytes")
	}
	if s.FilesPushLimit < 1024 {
		return errors.New("files_push_limit must be >= 1024 bytes")
	}
	if s.MaxFilesPerPush <= 0 {
		return errors.New("max_files_per_push must be > 0")
	}
	if s.RateLimitEnable && s.RateLimitRPM <= 0 {
		return errors.New("ratelimit_rpm must be > 0 when rate limiting is enabled")
	}
	if s.ClientAuthEnable && strings.TrimSpace(s.ClientJWTSecret) == "" {
		return errors.New("client_jwt_secret is required when client_auth_enable is true")
	}
	if s.AuditRetentionDays <= 0 {
		return errors.New("audit_retention_days must be > 0")
	}
	return nil
}
