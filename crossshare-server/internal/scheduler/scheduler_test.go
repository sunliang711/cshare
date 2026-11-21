package scheduler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/safekeep"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
)

func TestBuiltinJobsAndSafekeepBackup(t *testing.T) {
	dir := t.TempDir()
	key, err := crypto.LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)
	store, err := db.Open(filepath.Join(dir, "crossshare.db"), dir)
	require.NoError(t, err)
	defer store.Close()

	mgr, err := settings.NewManager(store, cipher)
	require.NoError(t, err)

	rec := audit.NewRecorder(store, mgr, zerolog.Nop())
	rec.Start()
	defer rec.Stop()

	mig := migration.NewService(store, cipher)

	// Insert an old audit log (>100 days old)
	require.NoError(t, store.InsertAuditLog(context.Background(), &db.AuditLog{
		ActorType: "client",
		Action:    "pull",
		CreatedAt: time.Now().UTC().AddDate(0, 0, -100),
	}))

	// Mock SafeKeep server
	var uploadedBytes []byte
	skSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "sk_cron_key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/v1/upstreams" {
			_, _ = w.Write([]byte(`{"items":[{"name":"r2-main"}]}`))
			return
		}
		if r.URL.Path == "/api/v1/backups" {
			require.NoError(t, r.ParseMultipartForm(10<<20))
			f, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer f.Close()
			uploadedBytes, err = io.ReadAll(f)
			require.NoError(t, err)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task": map[string]any{"id": "t-99", "status": "succeeded"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer skSrv.Close()

	skClient := safekeep.NewClientWithHTTP(skSrv.Client())
	p2p := service.NewP2PService(zerolog.Nop())
	eng := NewEngine(store, nil, p2p, mgr, rec, cipher, mig, skClient, zerolog.Nop())

	// 1) Run cleanup_audit_logs
	cleanJob, err := store.GetCronJobByKey(context.Background(), "cleanup_audit_logs")
	require.NoError(t, err)
	run, err := eng.RunNow(context.Background(), cleanJob.ID, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", run.Status)
	assert.Equal(t, int64(1), run.AffectedRows)

	// 2) Run stats_snapshot
	snapJob, err := store.GetCronJobByKey(context.Background(), "stats_snapshot")
	require.NoError(t, err)
	runSnap, err := eng.RunNow(context.Background(), snapJob.ID, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", runSnap.Status)

	// 3) Configure & run safekeep_backup
	skJob, err := store.GetCronJobByKey(context.Background(), "safekeep_backup")
	require.NoError(t, err)
	mergedCfg, err := MergeSafekeepBackupConfig(cipher, skJob.ConfigJSON, json.RawMessage(`{
		"api_base_url": "`+skSrv.URL+`",
		"api_key": "sk_cron_key",
		"path": "cshare-backups",
		"mode": "sync",
		"include_admins": true
	}`))
	require.NoError(t, err)
	require.NoError(t, store.UpdateCronJob(context.Background(), skJob.ID, "0 3 * * 0", true, 300, mergedCfg))

	testRes, err := eng.TestSafekeepBackup(context.Background(), mergedCfg, "", "")
	require.NoError(t, err)
	assert.Equal(t, []string{"r2-main"}, testRes.Upstreams)

	runSk, err := eng.RunNow(context.Background(), skJob.ID, "tester")
	require.NoError(t, err)
	assert.Equal(t, "succeeded", runSk.Status)
	assert.NotEmpty(t, uploadedBytes)

	// Uploaded bundle should be inspectable as unencrypted bundle
	summary, err := mig.Inspect(context.Background(), uploadedBytes, "")
	require.NoError(t, err)
	assert.False(t, summary.Encrypted)
}
