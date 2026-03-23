package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type APIResponse struct {
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

func NewClient(baseURL, token string, noProxy bool) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout: 5 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	if noProxy {
		transport.Proxy = nil
	}
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) url(path string) string {
	return c.BaseURL + "/api/v1" + path
}

func (c *Client) setAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) doJSON(method, path string, body interface{}, headers map[string]string) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.url(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuth(req)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("server returned non-JSON (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return &apiResp, nil
}

func (c *Client) doRaw(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.url(path), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}
