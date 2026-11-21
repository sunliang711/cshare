package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/safekeep"
)

type SafekeepBackupConfig struct {
	APIBaseURL          string `json:"api_base_url"`
	APIKey              string `json:"api_key,omitempty"`
	APIKeyEncrypted     string `json:"api_key_encrypted,omitempty"`
	HasAPIKey           bool   `json:"has_api_key,omitempty"`
	Targets             string `json:"targets"`
	Path                string `json:"path"`
	Mode                string `json:"mode"`
	Rename              string `json:"rename"`
	Passphrase          string `json:"passphrase,omitempty"`
	PassphraseEncrypted string `json:"passphrase_encrypted,omitempty"`
	HasPassphrase       bool   `json:"has_passphrase,omitempty"`
	IncludeAdmins       bool   `json:"include_admins"`
	IncludeAuditLogs    bool   `json:"include_audit_logs"`
	IncludeCronRuns     bool   `json:"include_cron_runs"`
}

func defaultSafekeepBackupConfig() SafekeepBackupConfig {
	return SafekeepBackupConfig{
		Path:          "cshare-backups",
		Mode:          "sync",
		IncludeAdmins: true,
	}
}

func ParseSafekeepBackupConfig(raw string) SafekeepBackupConfig {
	cfg := defaultSafekeepBackupConfig()
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.Mode == "" {
		cfg.Mode = "sync"
	}
	if cfg.Path == "" {
		cfg.Path = "cshare-backups"
	}
	return cfg
}

func SanitizeSafekeepBackupConfigJSON(raw string) string {
	cfg := ParseSafekeepBackupConfig(raw)
	cfg.HasAPIKey = cfg.APIKeyEncrypted != "" || cfg.APIKey != ""
	cfg.HasPassphrase = cfg.PassphraseEncrypted != "" || cfg.Passphrase != ""
	cfg.APIKey = ""
	cfg.APIKeyEncrypted = ""
	cfg.Passphrase = ""
	cfg.PassphraseEncrypted = ""
	b, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func isMaskedPlaceholder(v string) bool {
	return v == "*****" || v == "••••••"
}

func MergeSafekeepBackupConfig(c *crypto.Cipher, existingRaw string, incoming json.RawMessage) (string, error) {
	if c == nil {
		return "", errors.New("cipher is required")
	}
	existing := ParseSafekeepBackupConfig(existingRaw)

	var req struct {
		APIBaseURL       string  `json:"api_base_url"`
		APIKey           *string `json:"api_key"`
		Targets          string  `json:"targets"`
		Path             string  `json:"path"`
		Mode             string  `json:"mode"`
		Rename           string  `json:"rename"`
		Passphrase       *string `json:"passphrase"`
		IncludeAdmins    *bool   `json:"include_admins"`
		IncludeAuditLogs *bool   `json:"include_audit_logs"`
		IncludeCronRuns  *bool   `json:"include_cron_runs"`
	}
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &req); err != nil {
			return "", fmt.Errorf("invalid safekeep_backup config: %w", err)
		}
	}

	baseURL := strings.TrimSpace(req.APIBaseURL)
	if baseURL != "" {
		validated, err := safekeep.ValidateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validated
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "sync"
	}
	if mode != "sync" && mode != "async" {
		return "", errors.New("mode must be 'sync' or 'async'")
	}

	rename := strings.TrimSpace(req.Rename)
	if rename == "." || rename == ".." || rename == "/" {
		return "", errors.New("rename filename is invalid")
	}

	apiKeyEnc := existing.APIKeyEncrypted
	if req.APIKey != nil {
		val := strings.TrimSpace(*req.APIKey)
		if val == "" {
			apiKeyEnc = ""
		} else if !isMaskedPlaceholder(val) {
			enc, err := c.EncryptString(val)
			if err != nil {
				return "", err
			}
			apiKeyEnc = enc
		}
	}

	passEnc := existing.PassphraseEncrypted
	if req.Passphrase != nil {
		val := *req.Passphrase
		if val == "" {
			passEnc = ""
		} else if !isMaskedPlaceholder(strings.TrimSpace(val)) {
			if len(val) < 6 {
				return "", errors.New("passphrase must be at least 6 characters (or leave empty)")
			}
			enc, err := c.EncryptString(val)
			if err != nil {
				return "", err
			}
			passEnc = enc
		}
	}

	includeAdmins := existing.IncludeAdmins
	if req.IncludeAdmins != nil {
		includeAdmins = *req.IncludeAdmins
	}
	includeAuditLogs := existing.IncludeAuditLogs
	if req.IncludeAuditLogs != nil {
		includeAuditLogs = *req.IncludeAuditLogs
	}
	includeCronRuns := existing.IncludeCronRuns
	if req.IncludeCronRuns != nil {
		includeCronRuns = *req.IncludeCronRuns
	}

	out := SafekeepBackupConfig{
		APIBaseURL:          baseURL,
		APIKeyEncrypted:     apiKeyEnc,
		Targets:             strings.TrimSpace(req.Targets),
		Path:                strings.TrimSpace(req.Path),
		Mode:                mode,
		Rename:              rename,
		PassphraseEncrypted: passEnc,
		IncludeAdmins:       includeAdmins,
		IncludeAuditLogs:    includeAuditLogs,
		IncludeCronRuns:     includeCronRuns,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func RevealSafekeepBackupSecrets(c *crypto.Cipher, raw string) (apiKey, passphrase string, err error) {
	if c == nil {
		return "", "", errors.New("cipher is required")
	}
	cfg := ParseSafekeepBackupConfig(raw)
	if cfg.APIKeyEncrypted != "" {
		apiKey, err = c.DecryptString(cfg.APIKeyEncrypted)
		if err != nil {
			return "", "", fmt.Errorf("decrypt api_key: %w", err)
		}
	}
	if cfg.PassphraseEncrypted != "" {
		passphrase, err = c.DecryptString(cfg.PassphraseEncrypted)
		if err != nil {
			return "", "", fmt.Errorf("decrypt passphrase: %w", err)
		}
	}
	return apiKey, passphrase, nil
}

func (e *Engine) TestSafekeepBackup(ctx context.Context, storedRaw, apiBaseURLOverride, apiKeyOverride string) (*safekeep.TestResult, error) {
	cfg := ParseSafekeepBackupConfig(storedRaw)
	baseURL := strings.TrimSpace(apiBaseURLOverride)
	if baseURL == "" {
		baseURL = cfg.APIBaseURL
	}
	apiKey := strings.TrimSpace(apiKeyOverride)
	if apiKey == "" || isMaskedPlaceholder(apiKey) {
		if cfg.APIKeyEncrypted != "" && e.cipher != nil {
			dec, err := e.cipher.DecryptString(cfg.APIKeyEncrypted)
			if err != nil {
				return nil, fmt.Errorf("decrypt stored api_key: %w", err)
			}
			apiKey = dec
		}
	}
	if e.skClient == nil {
		return nil, errors.New("safekeep client not configured")
	}
	return e.skClient.TestConnection(ctx, baseURL, apiKey)
}

func (e *Engine) runSafekeepBackup(ctx context.Context, rawCfg string) (int64, string, error) {
	if e.migSvc == nil || e.cipher == nil || e.skClient == nil {
		return 0, "", errors.New("safekeep backup dependencies are not configured")
	}
	cfg := ParseSafekeepBackupConfig(rawCfg)
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		return 0, "", errors.New("api_base_url is required in safekeep_backup configuration")
	}
	apiKey := ""
	if cfg.APIKeyEncrypted != "" {
		var err error
		apiKey, err = e.cipher.DecryptString(cfg.APIKeyEncrypted)
		if err != nil {
			return 0, "", fmt.Errorf("decrypt configured api_key: %w", err)
		}
	}
	if strings.TrimSpace(apiKey) == "" {
		return 0, "", errors.New("api_key is required in safekeep_backup configuration")
	}

	passphrase := ""
	if cfg.PassphraseEncrypted != "" {
		var err error
		passphrase, err = e.cipher.DecryptString(cfg.PassphraseEncrypted)
		if err != nil {
			return 0, "", fmt.Errorf("decrypt passphrase: %w", err)
		}
	}

	bundleBytes, err := e.migSvc.Export(ctx, migration.ExportOptions{
		IncludeAdmins:    cfg.IncludeAdmins,
		IncludeAuditLogs: cfg.IncludeAuditLogs,
		IncludeCronRuns:  cfg.IncludeCronRuns,
		Passphrase:       passphrase,
	})
	if err != nil {
		return 0, "", fmt.Errorf("export migration bundle: %w", err)
	}

	filename := fmt.Sprintf("cshare-migration-%s.csbundle", time.Now().UTC().Format("20060102-150405"))
	res, err := e.skClient.UploadBackup(ctx, safekeep.UploadRequest{
		APIBaseURL: cfg.APIBaseURL,
		APIKey:     apiKey,
		Mode:       cfg.Mode,
		Path:       cfg.Path,
		Targets:    cfg.Targets,
		Rename:     cfg.Rename,
		Filename:   filename,
		Data:       bundleBytes,
		Metadata: map[string]any{
			"source":             "cshare_cron_safekeep_backup",
			"include_admins":     cfg.IncludeAdmins,
			"include_audit_logs": cfg.IncludeAuditLogs,
			"include_cron_runs":  cfg.IncludeCronRuns,
			"encrypted_with_pwd": passphrase != "",
		},
	})
	if err != nil {
		return 0, "", err
	}

	modeNote := res.TaskStatus
	if res.TimedOutToAsync {
		modeNote = "timed_out_to_async"
	}
	return 1, fmt.Sprintf("uploaded %s (%d bytes) to %s -> task %s (%s)", res.StoredFilename, len(bundleBytes), res.Endpoint, res.TaskID, modeNote), nil
}
