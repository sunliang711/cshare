package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/db"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/service"
)

const accountTestPassword = "test-password-123"

func accountRequest(t *testing.T, h *testHarness, method, path string, payload any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		require.NoError(t, json.NewEncoder(&body).Encode(payload))
	}
	return doAdminReq(h.router, method, "/admin/api/v1"+path, &body, cookies,
		map[string]string{"Content-Type": "application/json", "User-Agent": "CrossShare integration test"})
}

func setupAccount(t *testing.T, h *testHarness) []*http.Cookie {
	t.Helper()
	response := accountRequest(t, h, "POST", "/setup/init", map[string]string{"username": "admin", "password": accountTestPassword}, nil)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	return response.Result().Cookies()
}

func loginAccount(t *testing.T, h *testHarness, username, password string) []*http.Cookie {
	t.Helper()
	response := accountRequest(t, h, "POST", "/auth/login", map[string]string{"username": username, "password": password}, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	return response.Result().Cookies()
}

func createAccount(t *testing.T, h *testHarness, cookies []*http.Cookie, username, role string) db.Admin {
	t.Helper()
	response := accountRequest(t, h, "POST", "/admins", service.CreateAdminInput{
		Username: username, Password: accountTestPassword, Role: role, CurrentPassword: accountTestPassword,
	}, cookies)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var account db.Admin
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &account))
	return account
}

func listAccountSessions(t *testing.T, h *testHarness, cookies []*http.Cookie) []service.AdminSessionView {
	t.Helper()
	response := accountRequest(t, h, "GET", "/auth/sessions", nil, cookies)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.NotContains(t, response.Body.String(), "token_hash")
	var result struct {
		Sessions []service.AdminSessionView `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	return result.Sessions
}

func TestAdminSessionLifecycle(t *testing.T) {
	h := setupFullApp(t, nil)
	first := setupAccount(t, h)
	second := loginAccount(t, h, "admin", accountTestPassword)
	require.NotEqual(t, first[0].Value, second[0].Value)
	sessions := listAccountSessions(t, h, first)
	require.Len(t, sessions, 2)
	var current, other string
	for _, session := range sessions {
		assert.NotEmpty(t, session.ClientIP)
		assert.Contains(t, session.UserAgent, "integration test")
		assert.True(t, session.ExpiresAt.After(session.CreatedAt))
		if session.Current {
			current = session.ID
		} else {
			other = session.ID
		}
	}
	require.NotEmpty(t, current)
	require.NotEmpty(t, other)
	response := accountRequest(t, h, "DELETE", "/auth/sessions/"+other, nil, first)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, second).Code)
	assert.Equal(t, http.StatusUnauthorized, doAdminReq(h.router, "GET", "/admin/api/v1/auth/me", nil, nil,
		map[string]string{"Authorization": "Bearer " + second[0].Value}).Code)
	assert.Equal(t, http.StatusNotFound, accountRequest(t, h, "DELETE", "/auth/sessions/"+other, nil, first).Code)
	assert.Equal(t, http.StatusOK, accountRequest(t, h, "GET", "/auth/me", nil, first).Code)
	response = accountRequest(t, h, "DELETE", "/auth/sessions/"+current, nil, first)
	require.Equal(t, http.StatusOK, response.Code)
	require.NotEmpty(t, response.Result().Cookies())
	assert.Equal(t, -1, response.Result().Cookies()[0].MaxAge)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, first).Code)
	third := loginAccount(t, h, "admin", accountTestPassword)
	require.Equal(t, http.StatusOK, accountRequest(t, h, "POST", "/auth/logout", nil, third).Code)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, third).Code)
}

func TestAdminSessionExpiryTouchAndLegacyJWT(t *testing.T) {
	h := setupFullApp(t, nil)
	cookies := setupAccount(t, h)
	sessions := listAccountSessions(t, h, cookies)
	require.Len(t, sessions, 1)
	oldSeen := time.Now().UTC().Add(-2 * time.Minute)
	_, err := h.store.DB().Exec(`UPDATE admin_sessions SET last_seen_at = ? WHERE id = ?`, oldSeen, sessions[0].ID)
	require.NoError(t, err)
	assert.True(t, listAccountSessions(t, h, cookies)[0].LastSeenAt.After(oldSeen))
	account, err := h.store.GetAdminByUsername(t.Context(), "admin")
	require.NoError(t, err)
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": account.ID, "pwd_v": middleware.PasswordVersion(account.PasswordHash), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(h.adminH.JWTSecret())
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, doAdminReq(h.router, "GET", "/admin/api/v1/auth/me", nil, nil,
		map[string]string{"Authorization": "Bearer " + legacy}).Code)
	_, err = h.store.DB().Exec(`UPDATE admin_sessions SET expires_at = ? WHERE id = ?`, time.Now().UTC().Add(-time.Minute), sessions[0].ID)
	require.NoError(t, err)
	fresh := loginAccount(t, h, "admin", accountTestPassword)
	assert.Len(t, listAccountSessions(t, h, fresh), 1)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, cookies).Code)
}

func TestAccountRolesAndPasswordVerification(t *testing.T) {
	h := setupFullApp(t, nil)
	adminCookies := setupAccount(t, h)
	response := accountRequest(t, h, "POST", "/admins", service.CreateAdminInput{Username: "viewer", Password: accountTestPassword, Role: db.RoleViewer}, adminCookies)
	assert.Equal(t, http.StatusForbidden, response.Code)
	viewer := createAccount(t, h, adminCookies, "viewer", db.RoleViewer)
	viewerCookies := loginAccount(t, h, "viewer", accountTestPassword)
	for _, path := range []string{"/dashboard/summary", "/audit-logs", "/cron-jobs", "/auth/sessions"} {
		assert.Equal(t, http.StatusOK, accountRequest(t, h, "GET", path, nil, viewerCookies).Code, path)
	}
	for _, req := range []struct{ method, path string }{
		{"GET", "/admins"}, {"POST", "/admins"}, {"PUT", "/admins/" + viewer.ID},
		{"GET", "/settings"}, {"PUT", "/settings"}, {"GET", "/settings/jwt-secret"},
		{"POST", "/settings/generate-jwt-secret"}, {"POST", "/cron-jobs/some-id/run"},
		{"PUT", "/cron-jobs/some-id"}, {"GET", "/cron-jobs/some-id/secrets"},
		{"POST", "/cron-jobs/some-id/test-safekeep"}, {"POST", "/migration/export"},
		{"POST", "/migration/import"}, {"POST", "/migration/inspect"},
	} {
		assert.Equal(t, http.StatusForbidden, accountRequest(t, h, req.method, req.path, map[string]any{}, viewerCookies).Code, req.method+req.path)
	}
	adminSessions := listAccountSessions(t, h, adminCookies)
	assert.Equal(t, http.StatusNotFound, accountRequest(t, h, "DELETE", "/auth/sessions/"+adminSessions[0].ID, nil, viewerCookies).Code)
	assert.Len(t, listAccountSessions(t, h, viewerCookies), 1)
	response = accountRequest(t, h, "PUT", "/admins/"+viewer.ID,
		service.UpdateAdminInput{Role: db.RoleAdmin, CurrentPassword: "incorrect"}, adminCookies)
	assert.Equal(t, http.StatusForbidden, response.Code)
	response = accountRequest(t, h, "PUT", "/admins/"+viewer.ID,
		service.UpdateAdminInput{Role: db.RoleAdmin, CurrentPassword: accountTestPassword}, adminCookies)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, viewerCookies).Code)
	promoted := loginAccount(t, h, "viewer", accountTestPassword)
	assert.Equal(t, http.StatusOK, accountRequest(t, h, "GET", "/admins", nil, promoted).Code)
}

func TestAccountDisableResetAndOwnPassword(t *testing.T) {
	h := setupFullApp(t, nil)
	rootCookies := setupAccount(t, h)
	account := createAccount(t, h, rootCookies, "operator", db.RoleViewer)
	cookies := loginAccount(t, h, account.Username, accountTestPassword)
	for _, active := range []bool{false, true} {
		response := accountRequest(t, h, "PUT", "/admins/"+account.ID,
			service.UpdateAdminInput{IsActive: &active, CurrentPassword: accountTestPassword}, rootCookies)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, cookies).Code)
	cookies = loginAccount(t, h, account.Username, accountTestPassword)
	response := accountRequest(t, h, "PUT", "/admins/"+account.ID,
		service.UpdateAdminInput{Password: "replacement-password-123", CurrentPassword: accountTestPassword}, rootCookies)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, cookies).Code)
	loginAccount(t, h, account.Username, "replacement-password-123")
	otherRoot := loginAccount(t, h, "admin", accountTestPassword)
	response = accountRequest(t, h, "PUT", "/auth/password", map[string]string{
		"old_password": accountTestPassword, "new_password": "new-root-password-123",
	}, rootCookies)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	freshRoot := response.Result().Cookies()
	require.NotEmpty(t, freshRoot)
	assert.Equal(t, http.StatusOK, accountRequest(t, h, "GET", "/auth/me", nil, freshRoot).Code)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, rootCookies).Code)
	assert.Equal(t, http.StatusUnauthorized, accountRequest(t, h, "GET", "/auth/me", nil, otherRoot).Code)
	assert.Len(t, listAccountSessions(t, h, freshRoot), 1)
}

func TestLastAdministratorAndConcurrentDemotions(t *testing.T) {
	h := setupFullApp(t, nil)
	cookies := setupAccount(t, h)
	root, err := h.store.GetAdminByUsername(t.Context(), "admin")
	require.NoError(t, err)
	response := accountRequest(t, h, "PUT", "/admins/"+root.ID,
		service.UpdateAdminInput{Role: db.RoleViewer, CurrentPassword: accountTestPassword}, cookies)
	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), "last_admin")
	second := createAccount(t, h, cookies, "second", db.RoleAdmin)
	secondActor, err := h.store.GetAdminByID(t.Context(), second.ID)
	require.NoError(t, err)
	// 并发降级必须至少保留一个启用的管理员。
	svc := service.NewAdminService(h.store, h.cipher)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, actor := range []*db.Admin{root, secondActor} {
		wg.Add(1)
		go func(actor *db.Admin) {
			defer wg.Done()
			_, _, err := svc.UpdateAccount(t.Context(), actor, actor.ID,
				service.UpdateAdminInput{Role: db.RoleViewer, CurrentPassword: accountTestPassword}, "127.0.0.1", "test")
			results <- err
		}(actor)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			assert.ErrorIs(t, err, db.ErrLastAdmin)
		}
	}
	assert.Equal(t, 1, successes)
	var count int
	require.NoError(t, h.store.DB().QueryRow(`SELECT COUNT(*) FROM admins WHERE is_active = 1 AND role = 'admin'`).Scan(&count))
	assert.Equal(t, 1, count)
}
