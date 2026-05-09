package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crossshare-server/internal/config"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/model"
	"crossshare-server/internal/service"
	"crossshare-server/internal/storage"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type apiResponse struct {
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

func defaultConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Port: 10431},
		Auth: config.AuthConfig{
			Enable:        false,
			JWTSecret:     "test-secret",
			JWTHeaderName: "Authorization",
		},
		Business: config.BusinessConfig{
			DefaultTTL:      300,
			MaxTTL:          2592000,
			TextJSONLimit:   1 << 20,
			FilesPushLimit:  20 << 20,
			MaxFilesPerPush: 20,
		},
		RateLimit: config.RateLimitConfig{Enable: false},
		CORS: config.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
			AllowHeaders: []string{
				"Content-Type", "Authorization", "X-Request-Id",
				"Filename", "X-Content-Type", "X-TTL",
				"Accept", "Delete-After-Pull",
			},
		},
		Redis: config.RedisConfig{Addr: "localhost:6379", DB: 1},
	}
}

func setupRouter(t *testing.T, opts ...func(*config.Config)) *gin.Engine {
	t.Helper()
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	logger := zerolog.Nop()

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", cfg.Redis.Addr, err)
	}
	client.FlushDB(ctx)
	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})

	store := storage.New(client, logger)
	svc := service.NewShareService(store, cfg, logger)

	healthH := handler.NewHealthHandler()
	pushH := handler.NewPushHandler(svc, cfg, logger)
	pullH := handler.NewPullHandler(svc, logger)
	rl := middleware.NewRateLimiter(cfg)
	authMw := middleware.Auth(cfg)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(cfg))
	r.Use(rl.Middleware())

	v2 := r.Group("/api/v1")
	{
		v2.GET("/health", healthH.Health)

		push := v2.Group("/push")
		push.Use(authMw)
		{
			push.POST("/text", pushH.PushText)
			push.POST("/binary", pushH.PushBinary)
			push.POST("/files", pushH.PushFiles)
			push.POST("", pushH.PushUnified)
		}

		pull := v2.Group("/pull")
		pull.Use(authMw)
		{
			pull.GET("/:key", pullH.Pull)
			pull.DELETE("/:key", pullH.Delete)
		}
	}
	return r
}

func doRequest(r *gin.Engine, method, path string, body *bytes.Reader, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body: %s", w.Body.String())
	return resp
}

func pushTextHelper(t *testing.T, r *gin.Engine, text string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"text": text, "ttl": 300})
	w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var data struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.NotEmpty(t, data.Key)
	return data.Key
}

func pushBinaryHelper(t *testing.T, r *gin.Engine, data []byte, filename string) string {
	t.Helper()
	headers := map[string]string{"Content-Type": "application/octet-stream"}
	if filename != "" {
		headers["Filename"] = filename
	}
	w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader(data), headers)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var result struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	require.NotEmpty(t, result.Key)
	return result.Key
}

func pushFilesHelper(t *testing.T, r *gin.Engine, files map[string][]byte) string {
	t.Helper()
	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	for name, data := range files {
		part, err := writer.CreateFormFile("files", name)
		require.NoError(t, err)
		_, err = part.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.WriteField("ttl", "300"))
	require.NoError(t, writer.Close())

	w := doRequest(r, "POST", "/api/v1/push/files", bytes.NewReader(body.Bytes()),
		map[string]string{"Content-Type": writer.FormDataContentType()})
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var result struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	require.NotEmpty(t, result.Key)
	return result.Key
}

func createJWTToken(secret, sub string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, _ := token.SignedString([]byte(secret))
	return s
}

// ---------------------------------------------------------------------------
// 1. Health Check
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	r := setupRouter(t)

	w := doRequest(r, "GET", "/api/v1/health", nil, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "OK", resp.Msg)
	assert.NotEmpty(t, resp.RequestID)

	var data model.HealthResult
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "crossshare-server", data.Service)
	assert.Equal(t, "up", data.Status)
	assert.NotEmpty(t, data.Time)
}

// ---------------------------------------------------------------------------
// 2. Push Text
// ---------------------------------------------------------------------------

func TestPushText(t *testing.T) {
	r := setupRouter(t)

	t.Run("success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"text": "hello world",
			"ttl":  3600,
		})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "push success", resp.Msg)

		var data model.PushResult
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.NotEmpty(t, data.Key)
		assert.Equal(t, 3600, data.TTL)
		assert.Equal(t, 11, data.Size)
		assert.Equal(t, "text", data.Type)
		assert.Greater(t, data.ExpireAt, int64(0))
	})

	t.Run("default ttl", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "no ttl"})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusOK, w.Code)
		var data model.PushResult
		resp := parseResponse(t, w)
		json.Unmarshal(resp.Data, &data)
		assert.Equal(t, 300, data.TTL)
	})

	t.Run("with filename and content_type", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"text":         "markdown content",
			"ttl":          300,
			"filename":     "readme.md",
			"content_type": "text/markdown; charset=utf-8",
		})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty text rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": ""})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1001, resp.Code)
	})

	t.Run("whitespace only text rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "   "})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1001, resp.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader([]byte("not json")),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1001, resp.Code)
	})

	t.Run("ttl below minimum", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "hello", "ttl": 10})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1003, resp.Code)
	})

	t.Run("ttl above maximum", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "hello", "ttl": 9999999})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1003, resp.Code)
	})
}

func TestPushText_PayloadTooLarge(t *testing.T) {
	r := setupRouter(t, func(cfg *config.Config) {
		cfg.Business.TextJSONLimit = 100
	})

	largeText := strings.Repeat("x", 200)
	body, _ := json.Marshal(map[string]any{"text": largeText})
	w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 1002, resp.Code)
}

// ---------------------------------------------------------------------------
// 3. Push Binary
// ---------------------------------------------------------------------------

func TestPushBinary(t *testing.T) {
	r := setupRouter(t)

	t.Run("success", func(t *testing.T) {
		data := []byte("binary content here")
		w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader(data),
			map[string]string{"Content-Type": "application/octet-stream"})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "push success", resp.Msg)

		var result model.PushResult
		json.Unmarshal(resp.Data, &result)
		assert.NotEmpty(t, result.Key)
		assert.Equal(t, len(data), result.Size)
		assert.Equal(t, "files", result.Type)
		assert.Equal(t, 1, result.FileCount)
		assert.Greater(t, result.StoredSize, 0)
	})

	t.Run("with filename and custom headers", func(t *testing.T) {
		data := []byte("zip payload")
		w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader(data),
			map[string]string{
				"Content-Type":   "application/octet-stream",
				"Filename":       "archive.zip",
				"X-Content-Type": "application/zip",
				"X-TTL":          "600",
			})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		var result model.PushResult
		json.Unmarshal(resp.Data, &result)
		assert.Equal(t, "archive.zip", result.Filename)
		assert.Equal(t, 600, result.TTL)
	})

	t.Run("empty body rejected", func(t *testing.T) {
		w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader([]byte{}),
			map[string]string{"Content-Type": "application/octet-stream"})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1001, resp.Code)
	})
}

func TestPushBinary_PayloadTooLarge(t *testing.T) {
	r := setupRouter(t, func(cfg *config.Config) {
		cfg.Business.FilesPushLimit = 100
	})

	data := make([]byte, 200)
	rand.Read(data)
	w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader(data),
		map[string]string{"Content-Type": "application/octet-stream"})

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 1002, resp.Code)
}

func TestPushFiles(t *testing.T) {
	r := setupRouter(t)

	t.Run("single file returns files type and pulls raw content", func(t *testing.T) {
		key := pushFilesHelper(t, r, map[string][]byte{
			"single.txt": []byte("single file content"),
		})

		w := doRequest(r, "GET", "/api/v1/pull/"+key, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "File", w.Header().Get("Crossshare-Type"))
		assert.Equal(t, "single.txt", w.Header().Get("Crossshare-Filename"))
		assert.Equal(t, "1", w.Header().Get("Crossshare-File-Count"))
		assert.Equal(t, "single file content", w.Body.String())
	})

	t.Run("multiple files pull as zip bundle", func(t *testing.T) {
		body := bytes.NewBuffer(nil)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("files", "a.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("alpha"))
		require.NoError(t, err)
		part, err = writer.CreateFormFile("files", "b.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("bravo"))
		require.NoError(t, err)
		require.NoError(t, writer.WriteField("ttl", "600"))
		require.NoError(t, writer.WriteField("name", "docs"))
		require.NoError(t, writer.Close())

		w := doRequest(r, "POST", "/api/v1/push/files", bytes.NewReader(body.Bytes()),
			map[string]string{"Content-Type": writer.FormDataContentType()})
		require.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		var result model.PushResult
		require.NoError(t, json.Unmarshal(resp.Data, &result))
		assert.Equal(t, "files", result.Type)
		assert.Equal(t, 2, result.FileCount)
		assert.Equal(t, 10, result.Size)
		assert.Greater(t, result.StoredSize, result.Size)
		assert.Equal(t, "docs.zip", result.Filename)
		assert.Len(t, result.Files, 2)

		w = doRequest(r, "GET", "/api/v1/pull/"+result.Key, nil,
			map[string]string{"Accept": "application/json"})
		require.Equal(t, http.StatusOK, w.Code)
		resp = parseResponse(t, w)
		var meta model.PullFilesResult
		require.NoError(t, json.Unmarshal(resp.Data, &meta))
		assert.Equal(t, 2, meta.FileCount)
		assert.Equal(t, result.StoredSize, meta.StoredSize)

		w = doRequest(r, "GET", "/api/v1/pull/"+result.Key, nil, nil)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Bundle", w.Header().Get("Crossshare-Type"))
		assert.Equal(t, "docs.zip", w.Header().Get("Crossshare-Filename"))
		assert.Equal(t, "2", w.Header().Get("Crossshare-File-Count"))

		zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
		require.NoError(t, err)
		got := map[string]string{}
		for _, zf := range zr.File {
			rc, err := zf.Open()
			require.NoError(t, err)
			data, err := io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
			got[zf.Name] = string(data)
		}
		assert.Equal(t, map[string]string{
			"a.txt": "alpha",
			"b.txt": "bravo",
		}, got)
	})
}

// ---------------------------------------------------------------------------
// 4. Push Unified
// ---------------------------------------------------------------------------

func TestPushUnified(t *testing.T) {
	r := setupRouter(t)

	t.Run("json dispatches to text", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "unified text", "ttl": 300})
		w := doRequest(r, "POST", "/api/v1/push", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		var result model.PushResult
		json.Unmarshal(resp.Data, &result)
		assert.Equal(t, "text", result.Type)
	})

	t.Run("json with charset", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "charset", "ttl": 300})
		w := doRequest(r, "POST", "/api/v1/push", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json; charset=utf-8"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("octet-stream dispatches to binary", func(t *testing.T) {
		w := doRequest(r, "POST", "/api/v1/push", bytes.NewReader([]byte("binary data")),
			map[string]string{"Content-Type": "application/octet-stream"})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		var result model.PushResult
		json.Unmarshal(resp.Data, &result)
		assert.Equal(t, "files", result.Type)
		assert.Equal(t, 1, result.FileCount)
	})

	t.Run("unsupported content type returns 415", func(t *testing.T) {
		w := doRequest(r, "POST", "/api/v1/push", bytes.NewReader([]byte("<xml/>")),
			map[string]string{"Content-Type": "application/xml"})

		assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1004, resp.Code)
	})
}

// ---------------------------------------------------------------------------
// 5. Pull
// ---------------------------------------------------------------------------

func TestPullText_JSONResponse(t *testing.T) {
	r := setupRouter(t)
	key := pushTextHelper(t, r, "hello world")

	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil,
		map[string]string{"Accept": "application/json"})

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "pull success", resp.Msg)

	var data model.PullTextResult
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, key, data.Key)
	assert.Equal(t, "hello world", data.Text)
	assert.Equal(t, "text/plain; charset=utf-8", data.ContentType)
	assert.Equal(t, 11, data.Size)
	assert.False(t, data.Deleted)
}

func TestPullText_StreamResponse(t *testing.T) {
	r := setupRouter(t)
	key := pushTextHelper(t, r, "streamed text")

	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "streamed text", w.Body.String())
	assert.Equal(t, "Text", w.Header().Get("Crossshare-Type"))
	assert.Equal(t, "false", w.Header().Get("Key-Deleted"))
	assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "Crossshare-Type")
}

func TestPullBinary_StreamAndHashIntegrity(t *testing.T) {
	r := setupRouter(t)

	originalData := make([]byte, 4096)
	rand.Read(originalData)
	expectedHash := sha256.Sum256(originalData)

	key := pushBinaryHelper(t, r, originalData, "test.bin")

	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "File", w.Header().Get("Crossshare-Type"))
	assert.Equal(t, "test.bin", w.Header().Get("Crossshare-Filename"))

	pulledHash := sha256.Sum256(w.Body.Bytes())
	assert.Equal(t, expectedHash, pulledHash, "binary content hash mismatch")
	assert.Equal(t, len(originalData), w.Body.Len())
}

func TestPull_NotFound(t *testing.T) {
	r := setupRouter(t)

	w := doRequest(r, "GET", "/api/v1/pull/abcdefgh", nil,
		map[string]string{"Accept": "application/json"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 1404, resp.Code)
	assert.Equal(t, "share not found", resp.Msg)
}

func TestPull_InvalidKey(t *testing.T) {
	r := setupRouter(t)

	t.Run("too short", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/pull/abc", nil,
			map[string]string{"Accept": "application/json"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1101, resp.Code)
	})

	t.Run("too long", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/pull/abcdefghijklmno", nil,
			map[string]string{"Accept": "application/json"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1101, resp.Code)
	})

	t.Run("contains special chars", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/pull/abc_defgh", nil,
			map[string]string{"Accept": "application/json"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1101, resp.Code)
	})
}

// ---------------------------------------------------------------------------
// 6. Delete-After-Pull
// ---------------------------------------------------------------------------

func TestDeleteAfterPull(t *testing.T) {
	r := setupRouter(t)
	key := pushTextHelper(t, r, "ephemeral content")

	// First pull with Delete-After-Pull: true
	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil,
		map[string]string{
			"Accept":            "application/json",
			"Delete-After-Pull": "true",
		})
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var data model.PullTextResult
	json.Unmarshal(resp.Data, &data)
	assert.Equal(t, "ephemeral content", data.Text)
	assert.True(t, data.Deleted)

	// Second pull → 404
	w2 := doRequest(r, "GET", "/api/v1/pull/"+key, nil,
		map[string]string{"Accept": "application/json"})
	assert.Equal(t, http.StatusNotFound, w2.Code)
	resp2 := parseResponse(t, w2)
	assert.Equal(t, 1404, resp2.Code)
}

// ---------------------------------------------------------------------------
// 7. Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	r := setupRouter(t)

	t.Run("success", func(t *testing.T) {
		key := pushTextHelper(t, r, "to be deleted")

		w := doRequest(r, "DELETE", "/api/v1/pull/"+key, nil, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "delete success", resp.Msg)

		var data model.DeleteResult
		json.Unmarshal(resp.Data, &data)
		assert.Equal(t, key, data.Key)
		assert.True(t, data.Deleted)
	})

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, "DELETE", "/api/v1/pull/abcdefgh", nil, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1404, resp.Code)
	})

	t.Run("idempotent - second delete returns 404", func(t *testing.T) {
		key := pushTextHelper(t, r, "double delete")

		w1 := doRequest(r, "DELETE", "/api/v1/pull/"+key, nil, nil)
		assert.Equal(t, http.StatusOK, w1.Code)

		w2 := doRequest(r, "DELETE", "/api/v1/pull/"+key, nil, nil)
		assert.Equal(t, http.StatusNotFound, w2.Code)
		resp := parseResponse(t, w2)
		assert.Equal(t, 1404, resp.Code)
	})
}

// ---------------------------------------------------------------------------
// 8. Auth
// ---------------------------------------------------------------------------

func TestAuth(t *testing.T) {
	secret := "test-jwt-secret-key"
	r := setupRouter(t, func(cfg *config.Config) {
		cfg.Auth.Enable = true
		cfg.Auth.JWTSecret = secret
	})

	t.Run("health does not require auth", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/health", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("reject push without token", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "hello"})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1601, resp.Code)
	})

	t.Run("reject pull without token", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/pull/abcdefgh", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("reject invalid token", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "hello"})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer invalid.token.value",
			})

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1601, resp.Code)
	})

	t.Run("reject wrong auth scheme", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"text": "hello"})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Basic dXNlcjpwYXNz",
			})

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 1601, resp.Code)
	})

	t.Run("accept valid token", func(t *testing.T) {
		token := createJWTToken(secret, "testuser")
		body, _ := json.Marshal(map[string]any{"text": "authenticated", "ttl": 300})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body),
			map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			})

		assert.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		assert.Equal(t, 0, resp.Code)
	})

	t.Run("full flow with auth", func(t *testing.T) {
		token := createJWTToken(secret, "testuser")
		authHeader := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + token,
		}

		// Push
		body, _ := json.Marshal(map[string]any{"text": "auth flow", "ttl": 300})
		w := doRequest(r, "POST", "/api/v1/push/text", bytes.NewReader(body), authHeader)
		require.Equal(t, http.StatusOK, w.Code)
		resp := parseResponse(t, w)
		var pushData struct {
			Key string `json:"key"`
		}
		json.Unmarshal(resp.Data, &pushData)

		// Pull
		w = doRequest(r, "GET", "/api/v1/pull/"+pushData.Key, nil,
			map[string]string{
				"Accept":        "application/json",
				"Authorization": "Bearer " + token,
			})
		assert.Equal(t, http.StatusOK, w.Code)

		// Delete
		w = doRequest(r, "DELETE", "/api/v1/pull/"+pushData.Key, nil,
			map[string]string{"Authorization": "Bearer " + token})
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// 9. Filename Sanitization
// ---------------------------------------------------------------------------

func TestFilenameSanitization(t *testing.T) {
	r := setupRouter(t)

	data := []byte("file data")
	w := doRequest(r, "POST", "/api/v1/push/binary", bytes.NewReader(data),
		map[string]string{
			"Content-Type": "application/octet-stream",
			"Filename":     "../../etc/passwd",
		})

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var result struct {
		Key string `json:"key"`
	}
	json.Unmarshal(resp.Data, &result)

	w2 := doRequest(r, "GET", "/api/v1/pull/"+result.Key, nil, nil)
	assert.Equal(t, http.StatusOK, w2.Code)

	filename := w2.Header().Get("Crossshare-Filename")
	assert.NotContains(t, filename, "/")
	assert.NotContains(t, filename, "\\")
}

// ---------------------------------------------------------------------------
// 10. Request ID
// ---------------------------------------------------------------------------

func TestRequestID(t *testing.T) {
	r := setupRouter(t)

	t.Run("auto generates request_id", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/health", nil, nil)
		rid := w.Header().Get("X-Request-Id")
		assert.NotEmpty(t, rid)
		assert.True(t, strings.HasPrefix(rid, "req-"))

		resp := parseResponse(t, w)
		assert.Equal(t, rid, resp.RequestID)
	})

	t.Run("preserves client request_id", func(t *testing.T) {
		w := doRequest(r, "GET", "/api/v1/health", nil,
			map[string]string{"X-Request-Id": "my-custom-id-123"})

		assert.Equal(t, "my-custom-id-123", w.Header().Get("X-Request-Id"))
		resp := parseResponse(t, w)
		assert.Equal(t, "my-custom-id-123", resp.RequestID)
	})
}

// ---------------------------------------------------------------------------
// 11. Full E2E Workflows
// ---------------------------------------------------------------------------

func TestFullWorkflow_TextPushPullDelete(t *testing.T) {
	r := setupRouter(t)

	// 1. Push text
	key := pushTextHelper(t, r, "end to end test")

	// 2. Pull as JSON
	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil,
		map[string]string{"Accept": "application/json"})
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var pullData model.PullTextResult
	json.Unmarshal(resp.Data, &pullData)
	assert.Equal(t, "end to end test", pullData.Text)
	assert.Equal(t, 15, pullData.Size)

	// 3. Pull as stream
	w = doRequest(r, "GET", "/api/v1/pull/"+key, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "end to end test", w.Body.String())

	// 4. Delete
	w = doRequest(r, "DELETE", "/api/v1/pull/"+key, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Confirm gone
	w = doRequest(r, "GET", "/api/v1/pull/"+key, nil,
		map[string]string{"Accept": "application/json"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFullWorkflow_BinaryPushPullIntegrity(t *testing.T) {
	r := setupRouter(t)

	originalData := make([]byte, 8192)
	rand.Read(originalData)

	key := pushBinaryHelper(t, r, originalData, "random.bin")

	w := doRequest(r, "GET", "/api/v1/pull/"+key, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originalData, w.Body.Bytes(), "binary content must match exactly")
}
