package audit

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/db"
	"crossshare-server/internal/settings"
)

func TestRecorderAsyncAndClientToggle(t *testing.T) {
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "crossshare.db"), dir)
	require.NoError(t, err)
	defer store.Close()

	mgr, err := settings.NewManager(store, nil)
	require.NoError(t, err)

	rec := NewRecorder(store, mgr, zerolog.Nop())
	rec.Start()

	rec.Record(Event{
		ActorType: "client",
		Action:    "push.text",
		ShareKey:  "Abc123",
		IPAddress: "127.0.0.1",
		Detail:    map[string]any{"size": 42},
	})

	// Disable client audit logging
	s := mgr.Get()
	s.AuditClientOps = false
	require.NoError(t, mgr.Update(context.Background(), s, "admin"))

	rec.Record(Event{
		ActorType: "client",
		Action:    "pull",
		ShareKey:  "Abc123",
	})
	rec.Record(Event{
		ActorType: "admin",
		ActorName: "root",
		Action:    "settings.update",
	})

	rec.Stop()

	logs, total, err := store.ListAuditLogs(context.Background(), db.AuditFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
}
