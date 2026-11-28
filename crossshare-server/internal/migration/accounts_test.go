package migration

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

func encodeAccountBundle(t *testing.T, bundle *Bundle) []byte {
	t.Helper()
	data, err := json.Marshal(bundle.Data)
	require.NoError(t, err)
	sum := sha256.Sum256(data)
	bundle.Checksum = hex.EncodeToString(sum[:])
	raw, err := json.Marshal(bundle)
	require.NoError(t, err)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err = writer.Write(raw)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	blob, err := crypto.EncodeBundle(compressed.Bytes(), "")
	require.NoError(t, err)
	return blob
}

func TestAccountRolesMigrationAndLegacyBundles(t *testing.T) {
	src, _, exporter := setupEnv(t)
	for _, account := range []*db.Admin{
		{ID: "root", Username: "root", PasswordHash: "root-hash", Role: db.RoleAdmin, IsActive: true},
		{ID: "viewer", Username: "viewer", PasswordHash: "viewer-hash", Role: db.RoleViewer, IsActive: true, MustChangePassword: true},
	} {
		require.NoError(t, src.CreateAdmin(t.Context(), account))
	}
	blob, err := exporter.Export(t.Context(), ExportOptions{IncludeAdmins: true})
	require.NoError(t, err)
	bundle, err := decodeBundle(blob, "")
	require.NoError(t, err)

	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "roles", true: "legacy"}[legacy], func(t *testing.T) {
			dst, _, importer := setupEnv(t)
			local := &db.Admin{ID: "local", Username: "local", PasswordHash: "local-hash", IsActive: true}
			require.NoError(t, dst.CreateAdmin(t.Context(), local))
			now := time.Now().UTC()
			require.NoError(t, dst.CreateAdminSession(t.Context(), local, &db.AdminSession{ID: "local-session", AdminID: local.ID,
				TokenHash: db.AdminTokenHash("local-token"), CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}))
			if legacy {
				for i := range bundle.Data.Admins {
					bundle.Data.Admins[i].Role = ""
				}
				blob = encodeAccountBundle(t, bundle)
			}
			_, err := importer.Import(t.Context(), blob, ImportOptions{Mode: "replace", IncludeAdmins: true})
			require.NoError(t, err)
			viewer, err := dst.GetAdminByID(t.Context(), "viewer")
			require.NoError(t, err)
			if legacy {
				assert.Equal(t, db.RoleAdmin, viewer.Role)
			} else {
				assert.Equal(t, db.RoleViewer, viewer.Role)
			}
			assert.True(t, viewer.MustChangePassword)
			_, err = dst.GetAdminSession(t.Context(), "local-token")
			assert.ErrorIs(t, err, db.ErrNotFound)
		})
	}
}

func TestAccountImportCannotRemoveLastAdministrator(t *testing.T) {
	src, _, exporter := setupEnv(t)
	require.NoError(t, src.CreateAdmin(t.Context(), &db.Admin{ID: "viewer", Username: "viewer", Role: db.RoleViewer, IsActive: true}))
	blob, err := exporter.Export(t.Context(), ExportOptions{IncludeAdmins: true})
	require.NoError(t, err)
	dst, _, importer := setupEnv(t)
	require.NoError(t, dst.CreateAdmin(t.Context(), &db.Admin{ID: "root", Username: "root", IsActive: true}))
	_, err = importer.Import(t.Context(), blob, ImportOptions{Mode: "replace", IncludeAdmins: true})
	assert.ErrorIs(t, err, db.ErrLastAdmin)
	root, err := dst.GetAdminByID(t.Context(), "root")
	require.NoError(t, err)
	assert.Equal(t, db.RoleAdmin, root.Role)
}
