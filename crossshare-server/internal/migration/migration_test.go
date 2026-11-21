package migration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

func setupEnv(t *testing.T) (*db.Store, *crypto.Cipher, *Service) {
	t.Helper()
	dir := t.TempDir()
	key, err := crypto.LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)
	store, err := db.Open(filepath.Join(dir, "crossshare.db"), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store, cipher, NewService(store, cipher)
}

func TestExportInspectImport_EncryptedAndPlain(t *testing.T) {
	ctx := context.Background()
	srcStore, srcCipher, srcMig := setupEnv(t)

	// Populate settings, admin, audit log, safekeep cron job
	s := db.DefaultSystemSettings()
	s.DefaultTTL = 1800
	s.ClientAuthEnable = true
	s.ClientJWTSecret = "jwt-secret-cross-env"
	require.NoError(t, srcStore.UpdateSystemSettings(ctx, srcCipher, s, "alice"))

	hash, err := crypto.HashPassword("password123")
	require.NoError(t, err)
	require.NoError(t, srcStore.CreateAdmin(ctx, &db.Admin{
		ID:           crypto.NewULID(),
		Username:     "alice",
		PasswordHash: hash,
		IsActive:     true,
	}))
	require.NoError(t, srcStore.InsertAuditLog(ctx, &db.AuditLog{
		ActorType: "admin",
		ActorName: "alice",
		Action:    "settings.update",
		Status:    "success",
	}))

	job, err := srcStore.GetCronJobByKey(ctx, "safekeep_backup")
	require.NoError(t, err)
	encKey, err := srcCipher.EncryptString("sk_prod_abc")
	require.NoError(t, err)
	require.NoError(t, srcStore.UpdateCronJob(ctx, job.ID, "0 3 * * 0", true, 300,
		`{"api_base_url":"https://sk.example.com","api_key_encrypted":"`+encKey+`","path":"cshare","mode":"sync","include_admins":true}`))

	// 1) Export encrypted bundle
	encBlob, err := srcMig.Export(ctx, ExportOptions{
		IncludeAdmins:    true,
		IncludeAuditLogs: true,
		Passphrase:       "migrate-pass-99",
	})
	require.NoError(t, err)

	// Target environment with a DIFFERENT master key
	dstStore, dstCipher, dstMig := setupEnv(t)

	_, err = dstMig.Inspect(ctx, encBlob, "")
	assert.ErrorIs(t, err, crypto.ErrPassphraseRequired)

	summary, err := dstMig.Inspect(ctx, encBlob, "migrate-pass-99")
	require.NoError(t, err)
	assert.True(t, summary.Encrypted)
	assert.Equal(t, 1, summary.Counts.Admins)
	assert.Equal(t, 1, summary.Counts.AuditLogs)

	counts, err := dstMig.Import(ctx, encBlob, ImportOptions{
		Passphrase:       "migrate-pass-99",
		Mode:             "replace",
		IncludeAdmins:    true,
		IncludeAuditLogs: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counts.Admins)

	gotSettings, err := dstStore.GetSystemSettings(ctx, dstCipher)
	require.NoError(t, err)
	assert.Equal(t, 1800, gotSettings.DefaultTTL)
	assert.True(t, gotSettings.ClientAuthEnable)
	assert.Equal(t, "jwt-secret-cross-env", gotSettings.ClientJWTSecret)

	dstJob, err := dstStore.GetCronJobByKey(ctx, "safekeep_backup")
	require.NoError(t, err)
	assert.True(t, dstJob.Enabled)

	// 2) Export unencrypted bundle (passphrase empty)
	plainBlob, err := srcMig.Export(ctx, ExportOptions{
		IncludeAdmins: false,
		Passphrase:    "",
	})
	require.NoError(t, err)
	_, _, dstMig2 := setupEnv(t)
	plainSummary, err := dstMig2.Inspect(ctx, plainBlob, "")
	require.NoError(t, err)
	assert.False(t, plainSummary.Encrypted)
	assert.Equal(t, 0, plainSummary.Counts.Admins)
}
