package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/safekeep"
	"crossshare-server/internal/scheduler"
	"crossshare-server/internal/server"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
	"crossshare-server/internal/storage"
)

type testHarness struct {
	router      *gin.Engine
	store       *db.Store
	cipher      *crypto.Cipher
	settingsMgr *settings.Manager
	auditRec    *audit.Recorder
	sched       *scheduler.Engine
	mig         *migration.Service
}

func setupFullApp(t *testing.T, skServer *httptest.Server) *testHarness {
	t.Helper()
	dir := t.TempDir()
	cfg := defaultConfig()
	cfg.Server.DataDir = dir

	key, err := crypto.LoadOrGenerateMasterKey(dir, "")
	require.NoError(t, err)
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)

	dbStore, err := db.Open(filepath.Join(dir, "crossshare.db"), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dbStore.Close() })

	settingsMgr, err := settings.NewManager(dbStore, cipher)
	require.NoError(t, err)

	logger := zerolog.Nop()
	auditRec := audit.NewRecorder(dbStore, settingsMgr, logger)
	auditRec.Start()
	t.Cleanup(auditRec.Stop)

	memStore := storage.NewInMemoryStorage(logger)
	shareSvc := service.NewShareService(memStore, settingsMgr, logger)
	p2pSvc := service.NewP2PService(logger)
	migSvc := migration.NewService(dbStore, cipher)

	skClient := safekeep.NewClient()
	if skServer != nil {
		skClient = safekeep.NewClientWithHTTP(skServer.Client())
	}
	sched := scheduler.NewEngine(dbStore, memStore, p2pSvc, settingsMgr, auditRec, cipher, migSvc, skClient, logger)

	adminH := handler.NewAdminHandler(dbStore, cipher, settingsMgr, auditRec, sched, migSvc, memStore, p2pSvc, logger)
	rl := middleware.NewRateLimiter(settingsMgr)

	r := server.NewRouter(server.Params{
		Version:       "test",
		Config:        cfg,
		Logger:        logger,
		Store:         dbStore,
		Settings:      settingsMgr,
		HealthHandler: handler.NewHealthHandler(),
		PushHandler:   handler.NewPushHandler(shareSvc, settingsMgr, auditRec, logger),
		PullHandler:   handler.NewPullHandler(shareSvc, auditRec, logger),
		P2PHandler:    handler.NewP2PHandler(p2pSvc, auditRec, logger),
		AdminHandler:  adminH,
		RateLimiter:   rl,
	})

	return &testHarness{
		router:      r,
		store:       dbStore,
		cipher:      cipher,
		settingsMgr: settingsMgr,
		auditRec:    auditRec,
		sched:       sched,
		mig:         migSvc,
	}
}

func doAdminReq(r *gin.Engine, method, path string, body io.Reader, cookies []*http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestClientWorksBeforeAdminInitialized(t *testing.T) {
	h := setupFullApp(t, nil)

	// Confirm admin is NOT initialized
	w := doAdminReq(h.router, "GET", "/admin/api/v1/setup/status", nil, nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var st struct {
		Initialized bool `json:"initialized"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &st))
	assert.False(t, st.Initialized)

	// Client push & pull must still succeed without admin initialized
	key := pushTextHelper(t, h.router, "works without admin setup")
	wPull := doRequest(h.router, "GET", "/api/v1/pull/"+key, nil, map[string]string{"Accept": "application/json"})
	require.Equal(t, http.StatusOK, wPull.Code)
	resp := parseResponse(t, wPull)
	var pullData modelPullText
	require.NoError(t, json.Unmarshal(resp.Data, &pullData))
	assert.Equal(t, "works without admin setup", pullData.Text)
}

type modelPullText struct {
	Text string `json:"text"`
}

func TestAdminEndToEndFlow(t *testing.T) {
	var uploadedBundle []byte
	skSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "sk_e2e_key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/v1/upstreams" {
			_, _ = w.Write([]byte(`{"items":[{"name":"s3-primary"}]}`))
			return
		}
		if r.URL.Path == "/api/v1/backups" {
			require.NoError(t, r.ParseMultipartForm(10<<20))
			f, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer f.Close()
			uploadedBundle, _ = io.ReadAll(f)
			_, _ = w.Write([]byte(`{"task":{"id":"tsk-1","status":"succeeded"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer skSrv.Close()

	h := setupFullApp(t, skSrv)

	// 1. Setup initial admin
	initBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	w := doAdminReq(h.router, "POST", "/admin/api/v1/setup/init", bytes.NewReader(initBody), nil, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)
	cookies := w.Result().Cookies()
	require.NotEmpty(t, cookies)

	// Second init attempt rejected
	w2 := doAdminReq(h.router, "POST", "/admin/api/v1/setup/init", bytes.NewReader(initBody), nil, map[string]string{"Content-Type": "application/json"})
	assert.Equal(t, http.StatusConflict, w2.Code)

	// 2. Settings hot reload: enable client JWT auth
	genW := doAdminReq(h.router, "POST", "/admin/api/v1/settings/generate-jwt-secret", nil, cookies, nil)
	require.Equal(t, http.StatusOK, genW.Code)
	var genRes struct {
		Secret string `json:"client_jwt_secret"`
	}
	require.NoError(t, json.Unmarshal(genW.Body.Bytes(), &genRes))
	require.NotEmpty(t, genRes.Secret)

	updSettings, _ := json.Marshal(map[string]any{
		"default_ttl":          900,
		"max_ttl":              86400,
		"text_json_limit":      1048576,
		"files_push_limit":     20971520,
		"max_files_per_push":   20,
		"ratelimit_enable":     true,
		"ratelimit_rpm":        100,
		"client_auth_enable":   true,
		"client_jwt_secret":    genRes.Secret,
		"audit_client_ops":     true,
		"audit_retention_days": 90,
	})
	wSet := doAdminReq(h.router, "PUT", "/admin/api/v1/settings", bytes.NewReader(updSettings), cookies, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, wSet.Code)

	// Client push without token now rejected!
	txtBody, _ := json.Marshal(map[string]any{"text": "denied"})
	wNoAuth := doRequest(h.router, "POST", "/api/v1/push/text", bytes.NewReader(txtBody), map[string]string{"Content-Type": "application/json"})
	assert.Equal(t, http.StatusUnauthorized, wNoAuth.Code)

	// Client push with newly configured secret succeeds with new default TTL=900!
	tok := createJWTToken(genRes.Secret, "cli-user")
	txtBody2, _ := json.Marshal(map[string]any{"text": "allowed"})
	wAuth := doRequest(h.router, "POST", "/api/v1/push/text", bytes.NewReader(txtBody2), map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + tok,
	})
	require.Equal(t, http.StatusOK, wAuth.Code)
	respAuth := parseResponse(t, wAuth)
	var pushOut struct {
		Key string `json:"key"`
		TTL int    `json:"ttl"`
	}
	require.NoError(t, json.Unmarshal(respAuth.Data, &pushOut))
	assert.Equal(t, 900, pushOut.TTL)

	// 3. Audit logs recorded client push and admin actions
	time.Sleep(50 * time.Millisecond)
	wAudit := doAdminReq(h.router, "GET", "/admin/api/v1/audit-logs?actor_type=client&share_key="+pushOut.Key, nil, cookies, nil)
	require.Equal(t, http.StatusOK, wAudit.Code)
	var auditRes struct {
		Items []db.AuditLog `json:"items"`
		Total int64         `json:"total"`
	}
	require.NoError(t, json.Unmarshal(wAudit.Body.Bytes(), &auditRes))
	assert.Equal(t, int64(1), auditRes.Total)
	assert.Equal(t, "push.text", auditRes.Items[0].Action)

	wCSV := doAdminReq(h.router, "GET", "/admin/api/v1/audit-logs/export?format=csv", nil, cookies, nil)
	require.Equal(t, http.StatusOK, wCSV.Code)
	assert.Contains(t, wCSV.Body.String(), "settings.update")

	// 4. Cron jobs & SafeKeep backup
	wJobs := doAdminReq(h.router, "GET", "/admin/api/v1/cron-jobs", nil, cookies, nil)
	require.Equal(t, http.StatusOK, wJobs.Code)
	var jobsRes struct {
		Items []db.CronJob `json:"items"`
	}
	require.NoError(t, json.Unmarshal(wJobs.Body.Bytes(), &jobsRes))
	var skJobID string
	for _, j := range jobsRes.Items {
		if j.JobKey == "safekeep_backup" {
			skJobID = j.ID
			assert.False(t, j.Enabled)
		}
	}
	require.NotEmpty(t, skJobID)

	updCron, _ := json.Marshal(map[string]any{
		"cron_expr":   "0 3 * * 0",
		"enabled":     true,
		"timeout_sec": 300,
		"config": map[string]any{
			"api_base_url":   skSrv.URL,
			"api_key":        "sk_e2e_key",
			"path":           "cshare-backups",
			"mode":           "sync",
			"include_admins": true,
		},
	})
	wUpdCron := doAdminReq(h.router, "PUT", "/admin/api/v1/cron-jobs/"+skJobID, bytes.NewReader(updCron), cookies, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, wUpdCron.Code)

	wTestSK := doAdminReq(h.router, "POST", "/admin/api/v1/cron-jobs/"+skJobID+"/test-safekeep", bytes.NewReader([]byte(`{}`)), cookies, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, wTestSK.Code)

	wRunSK := doAdminReq(h.router, "POST", "/admin/api/v1/cron-jobs/"+skJobID+"/run", nil, cookies, nil)
	require.Equal(t, http.StatusOK, wRunSK.Code)
	assert.NotEmpty(t, uploadedBundle)

	// 5. Migration export (plain) + inspect + import on second instance
	expBody, _ := json.Marshal(map[string]any{"include_admins": true, "passphrase": ""})
	wExp := doAdminReq(h.router, "POST", "/admin/api/v1/migration/export", bytes.NewReader(expBody), cookies, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, wExp.Code)
	bundleBytes := wExp.Body.Bytes()

	h2 := setupFullApp(t, nil)
	// First create an admin on h2 so we can call inspect/import
	wInit2 := doAdminReq(h2.router, "POST", "/admin/api/v1/setup/init", bytes.NewReader(initBody), nil, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusCreated, wInit2.Code)
	cookies2 := wInit2.Result().Cookies()

	var mpBuf bytes.Buffer
	mw := multipart.NewWriter(&mpBuf)
	fw, err := mw.CreateFormFile("file", "test.csbundle")
	require.NoError(t, err)
	_, _ = fw.Write(bundleBytes)
	_ = mw.WriteField("mode", "merge")
	_ = mw.WriteField("conflict_strategy", "overwrite")
	_ = mw.WriteField("include_admins", "true")
	_ = mw.Close()

	wImp := doAdminReq(h2.router, "POST", "/admin/api/v1/migration/import", bytes.NewReader(mpBuf.Bytes()), cookies2, map[string]string{"Content-Type": mw.FormDataContentType()})
	require.Equal(t, http.StatusOK, wImp.Code)
	assert.Equal(t, 900, h2.settingsMgr.Get().DefaultTTL)
	assert.True(t, h2.settingsMgr.Get().ClientAuthEnable)
	assert.Equal(t, genRes.Secret, h2.settingsMgr.Get().ClientJWTSecret)

	// 6. Dashboard summary
	wDash := doAdminReq(h.router, "GET", "/admin/api/v1/dashboard/summary", nil, cookies, nil)
	require.Equal(t, http.StatusOK, wDash.Code)
}
