package safekeep

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientUploadAndTestConnection(t *testing.T) {
	var gotKey string
	var gotMode, gotPath, gotTargets string
	var gotFileContent []byte
	var gotFilename string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		if gotKey != "sk_test_123" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"invalid api key"}}`))
			return
		}
		if r.URL.Path == "/api/v1/upstreams" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"name":"prod-s3"},{"name":"dropbox"}]}`))
			return
		}
		if r.URL.Path == "/api/v1/backups" && r.Method == http.MethodPost {
			require.NoError(t, r.ParseMultipartForm(10<<20))
			gotMode = r.FormValue("mode")
			gotPath = r.FormValue("path")
			gotTargets = r.FormValue("targets")
			f, hdr, err := r.FormFile("file")
			require.NoError(t, err)
			defer f.Close()
			gotFilename = hdr.Filename
			gotFileContent, err = io.ReadAll(f)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task": map[string]any{
					"id":              "task-01",
					"status":          "succeeded",
					"remote_filename": gotFilename,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClientWithHTTP(srv.Client())

	// TestConnection
	tr, err := client.TestConnection(context.Background(), srv.URL, "sk_test_123")
	require.NoError(t, err)
	assert.True(t, tr.Reachable)
	assert.Equal(t, []string{"prod-s3", "dropbox"}, tr.Upstreams)

	// Invalid key
	_, err = client.TestConnection(context.Background(), srv.URL, "bad_key")
	assert.Error(t, err)

	// UploadBackup
	res, err := client.UploadBackup(context.Background(), UploadRequest{
		APIBaseURL: srv.URL,
		APIKey:     "sk_test_123",
		Mode:       "sync",
		Path:       "cshare-backups",
		Targets:    "prod-s3",
		Filename:   "cshare-test.csbundle",
		Data:       []byte("bundle-bytes"),
	})
	require.NoError(t, err)
	assert.Equal(t, "task-01", res.TaskID)
	assert.Equal(t, "succeeded", res.TaskStatus)
	assert.Equal(t, "sync", gotMode)
	assert.Equal(t, "cshare-backups", gotPath)
	assert.Equal(t, "prod-s3", gotTargets)
	assert.Equal(t, []byte("bundle-bytes"), gotFileContent)
}
