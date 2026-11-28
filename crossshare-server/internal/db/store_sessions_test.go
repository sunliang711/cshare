package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyAdminsReceiveAdministratorRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = legacy.Exec(`CREATE TABLE admins (
		id TEXT PRIMARY KEY, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL,
		is_active INTEGER NOT NULL, last_login_at DATETIME, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`)
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = legacy.Exec(`INSERT INTO admins(id, username, password_hash, is_active, created_at, updated_at) VALUES ('legacy', 'legacy', 'hash', 1, ?, ?)`, now, now)
	require.NoError(t, err)
	require.NoError(t, legacy.Close())
	store, err := Open(path, t.TempDir())
	require.NoError(t, err)
	defer store.Close()
	admin, err := store.GetAdminByID(t.Context(), "legacy")
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, admin.Role)
	assert.False(t, admin.MustChangePassword)
}

func TestSessionPersistsAcrossStoreRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.db")
	store, err := Open(path, dir)
	require.NoError(t, err)
	admin := &Admin{ID: "admin", Username: "admin", PasswordHash: "hash", IsActive: true}
	require.NoError(t, store.CreateAdmin(t.Context(), admin))
	now := time.Now().UTC()
	session := &AdminSession{ID: "session", AdminID: admin.ID, TokenHash: AdminTokenHash("secret"), CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	require.NoError(t, store.CreateAdminSession(t.Context(), admin, session))
	require.NoError(t, store.Close())
	store, err = Open(path, dir)
	require.NoError(t, err)
	defer store.Close()
	got, err := store.GetAdminSession(t.Context(), "secret")
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)
	require.NoError(t, store.DeleteAdminSession(t.Context(), admin.ID, session.ID))
	_, err = store.GetAdminSession(t.Context(), "secret")
	assert.ErrorIs(t, err, ErrNotFound)
}
