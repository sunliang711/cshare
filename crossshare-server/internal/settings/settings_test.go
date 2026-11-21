package settings

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

func TestManagerPersistenceAndHotReload(t *testing.T) {
	dir := t.TempDir()
	key, err := crypto.LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)

	store, err := db.Open(filepath.Join(dir, "crossshare.db"), dir)
	require.NoError(t, err)
	defer store.Close()

	mgr, err := NewManager(store, cipher)
	require.NoError(t, err)

	var notified db.SystemSettings
	mgr.OnChange(func(s db.SystemSettings) {
		notified = s
	})

	cur := mgr.Get()
	assert.Equal(t, 600, cur.DefaultTTL)
	assert.False(t, cur.ClientAuthEnable)

	next := cur
	next.DefaultTTL = 1200
	next.MaxTTL = 172800
	next.ClientAuthEnable = true
	next.ClientJWTSecret = "jwt-secret-xyz"
	next.RateLimitRPM = 45

	require.NoError(t, mgr.Update(context.Background(), next, "admin"))
	assert.Equal(t, 1200, notified.DefaultTTL)
	assert.True(t, notified.ClientAuthEnable)
	assert.Equal(t, "jwt-secret-xyz", notified.ClientJWTSecret)

	// Re-open manager and confirm encrypted secret + settings round-trip
	mgr2, err := NewManager(store, cipher)
	require.NoError(t, err)
	got := mgr2.Get()
	assert.Equal(t, 1200, got.DefaultTTL)
	assert.Equal(t, 172800, got.MaxTTL)
	assert.True(t, got.ClientAuthEnable)
	assert.True(t, got.HasClientJWTSecret)
	assert.Equal(t, "jwt-secret-xyz", got.ClientJWTSecret)
	assert.Equal(t, 45, got.RateLimitRPM)
}

func TestManagerValidation(t *testing.T) {
	mgr := NewStaticManager(db.DefaultSystemSettings())
	bad := mgr.Get()
	bad.ClientAuthEnable = true
	bad.ClientJWTSecret = ""
	err := mgr.Update(context.Background(), bad, "admin")
	assert.Error(t, err)
}
