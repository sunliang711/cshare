package safekeep

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

func NewClientWithHTTP(hc *http.Client) *Client {
	return &Client{httpClient: hc}
}

type UploadRequest struct {
	APIBaseURL string
	APIKey     string
	Mode       string
	Path       string
	Targets    string
	Rename     string
	Metadata   map[string]any
	Filename   string
	Data       []byte
}

type UploadResult struct {
	StatusCode      int    `json:"status_code"`
	TaskID          string `json:"task_id"`
	TaskStatus      string `json:"task_status"`
	StoredFilename  string `json:"stored_filename"`
	TimedOutToAsync bool   `json:"timed_out_to_async"`
	Endpoint        string `json:"endpoint"`
}

type TestResult struct {
	Reachable bool     `json:"reachable"`
	Upstreams []string `json:"upstreams"`
	LatencyMs int64    `json:"latency_ms"`
}

func ValidateBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("api_base_url is required")
	}
	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("api_base_url must be a valid http or https URL")
	}
	return strings.TrimRight(trimmed, "/"), nil
}

func (c *Client) TestConnection(ctx context.Context, apiBaseURL, apiKey string) (*TestResult, error) {
	baseURL, err := ValidateBaseURL(apiBaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("api_key is required")
	}

	endpoint := baseURL + "/api/v1/upstreams"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", strings.TrimSpace(apiKey))

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("connect safekeep: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, formatAPIError(resp.StatusCode, body)
	}

	var parsed struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Upstreams []struct {
			Name string `json:"name"`
		} `json:"upstreams"`
	}
	_ = json.Unmarshal(body, &parsed)
	names := make([]string, 0)
	for _, it := range parsed.Items {
		if it.Name != "" {
			names = append(names, it.Name)
		}
	}
	if len(names) == 0 {
		for _, it := range parsed.Upstreams {
			if it.Name != "" {
				names = append(names, it.Name)
			}
		}
	}

	return &TestResult{
		Reachable: true,
		Upstreams: names,
		LatencyMs: latency,
	}, nil
}

func (c *Client) UploadBackup(ctx context.Context, req UploadRequest) (*UploadResult, error) {
	baseURL, err := ValidateBaseURL(req.APIBaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return nil, errors.New("api_key is required")
	}
	if len(req.Data) == 0 {
		return nil, errors.New("backup payload is empty")
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "sync"
	}
	if mode != "sync" && mode != "async" {
		return nil, errors.New("mode must be 'sync' or 'async'")
	}

	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = fmt.Sprintf("cshare-migration-%s.csbundle", time.Now().UTC().Format("20060102-150405"))
	}

	var bodyBuf bytes.Buffer
	mw := multipart.NewWriter(&bodyBuf)
	_ = mw.WriteField("mode", mode)
	if strings.TrimSpace(req.Path) != "" {
		_ = mw.WriteField("path", strings.TrimSpace(req.Path))
	}
	if strings.TrimSpace(req.Targets) != "" {
		_ = mw.WriteField("targets", strings.TrimSpace(req.Targets))
	}
	if strings.TrimSpace(req.Rename) != "" {
		_ = mw.WriteField("rename", strings.TrimSpace(req.Rename))
	}
	_ = mw.WriteField("dedup", "false")
	if req.Metadata != nil {
		if metaJSON, mErr := json.Marshal(req.Metadata); mErr == nil {
			_ = mw.WriteField("metadata", string(metaJSON))
		}
	}

	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create multipart form file: %w", err)
	}
	if _, err := fw.Write(req.Data); err != nil {
		return nil, fmt.Errorf("write bundle to multipart body: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	endpoint := baseURL + "/api/v1/backups"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("build backup request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("X-API-Key", strings.TrimSpace(req.APIKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		var out struct {
			TimedOutToAsync bool `json:"timed_out_to_async"`
			Task            struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				OriginalFilename string `json:"original_filename"`
				RemoteFilename   string `json:"remote_filename"`
			} `json:"task"`
		}
		_ = json.Unmarshal(respBytes, &out)
		storedName := out.Task.RemoteFilename
		if storedName == "" {
			if req.Rename != "" {
				storedName = req.Rename
			} else {
				storedName = filename
			}
		}
		return &UploadResult{
			StatusCode:      resp.StatusCode,
			TaskID:          out.Task.ID,
			TaskStatus:      out.Task.Status,
			StoredFilename:  storedName,
			TimedOutToAsync: out.TimedOutToAsync,
			Endpoint:        endpoint,
		}, nil
	}

	return nil, formatAPIError(resp.StatusCode, respBytes)
}

func formatAPIError(status int, respBytes []byte) error {
	var errOut struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Task struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			ErrorSummary string `json:"error_summary"`
		} `json:"task"`
	}
	_ = json.Unmarshal(respBytes, &errOut)
	if errOut.Task.ID != "" {
		return fmt.Errorf("safekeep API %d (task %s, %s): %s", status, errOut.Task.ID, errOut.Task.Status, errOut.Task.ErrorSummary)
	}
	if errOut.Error.Message != "" {
		return fmt.Errorf("safekeep API %d (%s): %s", status, errOut.Error.Code, errOut.Error.Message)
	}
	return fmt.Errorf("safekeep API returned HTTP %d: %s", status, strings.TrimSpace(string(respBytes)))
}
