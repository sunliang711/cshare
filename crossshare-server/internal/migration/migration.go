package migration

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/fx"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
)

const FormatVersion = 1

var Module = fx.Options(
	fx.Provide(NewService),
)

type ExportOptions struct {
	IncludeAdmins    bool   `json:"include_admins"`
	IncludeAuditLogs bool   `json:"include_audit_logs"`
	IncludeCronRuns  bool   `json:"include_cron_runs"`
	Passphrase       string `json:"passphrase"`
}

type ImportOptions struct {
	Passphrase       string `json:"passphrase"`
	Mode             string `json:"mode"`              // replace | merge
	ConflictStrategy string `json:"conflict_strategy"` // skip | overwrite
	IncludeAdmins    bool   `json:"include_admins"`
	IncludeAuditLogs bool   `json:"include_audit_logs"`
	IncludeCronRuns  bool   `json:"include_cron_runs"`
}

type Bundle struct {
	FormatVersion int           `json:"format_version"`
	ExportedAt    time.Time     `json:"exported_at"`
	Encrypted     bool          `json:"encrypted"`
	Includes      BundleInclude `json:"includes"`
	Checksum      string        `json:"checksum_sha256"`
	Data          BundleData    `json:"data"`
}

type BundleInclude struct {
	Admins    bool `json:"admins"`
	AuditLogs bool `json:"audit_logs"`
	CronRuns  bool `json:"cron_runs"`
}

type BundleData struct {
	SystemSettings []SettingRecord `json:"system_settings"`
	CronJobs       []db.CronJob    `json:"cron_jobs"`
	Admins         []AdminRecord   `json:"admins,omitempty"`
	AuditLogs      []db.AuditLog   `json:"audit_logs,omitempty"`
	CronJobRuns    []db.CronJobRun `json:"cron_job_runs,omitempty"`
}

type SettingRecord struct {
	Key       string    `json:"key"`
	ValueJSON string    `json:"value_json"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

type AdminRecord struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"password_hash"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type BundleCounts struct {
	SystemSettings int `json:"system_settings"`
	CronJobs       int `json:"cron_jobs"`
	Admins         int `json:"admins"`
	AuditLogs      int `json:"audit_logs"`
	CronJobRuns    int `json:"cron_job_runs"`
}

type InspectSummary struct {
	FormatVersion  int           `json:"format_version"`
	ExportedAt     time.Time     `json:"exported_at"`
	Encrypted      bool          `json:"encrypted"`
	Includes       BundleInclude `json:"includes"`
	Counts         BundleCounts  `json:"counts"`
	AdminConflicts []string      `json:"admin_conflicts"`
}

type Service struct {
	store  *db.Store
	cipher *crypto.Cipher
}

func NewService(store *db.Store, cipher *crypto.Cipher) *Service {
	return &Service{store: store, cipher: cipher}
}

func (s *Service) Export(ctx context.Context, opts ExportOptions) ([]byte, error) {
	if opts.Passphrase != "" && len(opts.Passphrase) < 6 {
		return nil, errors.New("migration passphrase must be at least 6 characters (or empty)")
	}

	var data BundleData

	// 1. System settings (decrypt client_jwt_secret_encrypted into portable plaintext)
	rows, err := s.store.DB().QueryContext(ctx, `SELECT key, value_json, updated_at, updated_by FROM system_settings ORDER BY key ASC`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r SettingRecord
		if err := rows.Scan(&r.Key, &r.ValueJSON, &r.UpdatedAt, &r.UpdatedBy); err != nil {
			rows.Close()
			return nil, err
		}
		if r.Key == "client_jwt_secret_encrypted" {
			var enc string
			if json.Unmarshal([]byte(r.ValueJSON), &enc) == nil && enc != "" && s.cipher != nil {
				plain, decErr := s.cipher.DecryptString(enc)
				if decErr != nil {
					rows.Close()
					return nil, fmt.Errorf("decrypt client_jwt_secret: %w", decErr)
				}
				r.Key = "client_jwt_secret_plain"
				if b, mErr := json.Marshal(plain); mErr == nil {
					r.ValueJSON = string(b)
				}
			} else {
				r.Key = "client_jwt_secret_plain"
				r.ValueJSON = `""`
			}
		}
		data.SystemSettings = append(data.SystemSettings, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Cron jobs (decrypt safekeep_backup secrets for portable export)
	jobs, err := s.store.ListCronJobs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].JobKey == "safekeep_backup" {
			cfgJSON, cErr := decryptSafekeepConfigForExport(s.cipher, jobs[i].ConfigJSON)
			if cErr != nil {
				return nil, fmt.Errorf("decrypt safekeep_backup config: %w", cErr)
			}
			jobs[i].ConfigJSON = cfgJSON
		}
	}
	data.CronJobs = jobs

	// 3. Optional Admins
	if opts.IncludeAdmins {
		admins, err := s.store.ListAdmins(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range admins {
			data.Admins = append(data.Admins, AdminRecord{
				ID:           a.ID,
				Username:     a.Username,
				PasswordHash: a.PasswordHash,
				IsActive:     a.IsActive,
				LastLoginAt:  a.LastLoginAt,
				CreatedAt:    a.CreatedAt,
				UpdatedAt:    a.UpdatedAt,
			})
		}
	}

	// 4. Optional Audit Logs
	if opts.IncludeAuditLogs {
		logs, err := s.store.ExportAllAuditLogs(ctx, db.AuditFilter{})
		if err != nil {
			return nil, err
		}
		data.AuditLogs = logs
	}

	// 5. Optional Cron Job Runs
	if opts.IncludeCronRuns {
		runs, err := s.exportCronJobRuns(ctx)
		if err != nil {
			return nil, err
		}
		data.CronJobRuns = runs
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(dataBytes)

	bundle := Bundle{
		FormatVersion: FormatVersion,
		ExportedAt:    time.Now().UTC(),
		Encrypted:     opts.Passphrase != "",
		Includes: BundleInclude{
			Admins:    opts.IncludeAdmins,
			AuditLogs: opts.IncludeAuditLogs,
			CronRuns:  opts.IncludeCronRuns,
		},
		Checksum: hex.EncodeToString(sum[:]),
		Data:     data,
	}

	rawBundle, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := gw.Write(rawBundle); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return crypto.EncodeBundle(gzBuf.Bytes(), opts.Passphrase)
}

func (s *Service) exportCronJobRuns(ctx context.Context) ([]db.CronJobRun, error) {
	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT id, job_id, trigger_type, triggered_by, status, affected_rows, output_log, started_at, finished_at
		 FROM cron_job_runs ORDER BY started_at DESC LIMIT 5000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]db.CronJobRun, 0)
	for rows.Next() {
		var r db.CronJobRun
		var finished sql.NullTime
		if err := rows.Scan(&r.ID, &r.JobID, &r.TriggerType, &r.TriggeredBy, &r.Status, &r.AffectedRows, &r.OutputLog, &r.StartedAt, &finished); err != nil {
			return nil, err
		}
		if finished.Valid {
			t := finished.Time
			r.FinishedAt = &t
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func decodeBundle(blob []byte, passphrase string) (*Bundle, error) {
	encrypted, err := crypto.IsBundleEncrypted(blob)
	if err != nil {
		return nil, err
	}
	gzBytes, err := crypto.DecodeBundle(blob, passphrase)
	if err != nil {
		return nil, err
	}

	gr, err := gzip.NewReader(bytes.NewReader(gzBytes))
	if err != nil {
		return nil, fmt.Errorf("decompress migration bundle: %w", err)
	}
	defer gr.Close()

	raw, err := io.ReadAll(io.LimitReader(gr, 128<<20))
	if err != nil {
		return nil, fmt.Errorf("read migration bundle: %w", err)
	}

	var envelope struct {
		FormatVersion int             `json:"format_version"`
		ExportedAt    time.Time       `json:"exported_at"`
		Encrypted     bool            `json:"encrypted"`
		Includes      BundleInclude   `json:"includes"`
		Checksum      string          `json:"checksum_sha256"`
		Data          json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse bundle json: %w", err)
	}
	if envelope.FormatVersion != FormatVersion {
		return nil, fmt.Errorf("unsupported migration format_version %d", envelope.FormatVersion)
	}
	if envelope.Checksum != "" {
		sum := sha256.Sum256(envelope.Data)
		if hex.EncodeToString(sum[:]) != envelope.Checksum {
			return nil, errors.New("bundle checksum verification failed")
		}
	}

	var data BundleData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("parse bundle data: %w", err)
	}
	return &Bundle{
		FormatVersion: envelope.FormatVersion,
		ExportedAt:    envelope.ExportedAt,
		Encrypted:     encrypted,
		Includes:      envelope.Includes,
		Checksum:      envelope.Checksum,
		Data:          data,
	}, nil
}

func (s *Service) Inspect(ctx context.Context, blob []byte, passphrase string) (*InspectSummary, error) {
	b, err := decodeBundle(blob, passphrase)
	if err != nil {
		return nil, err
	}

	summary := &InspectSummary{
		FormatVersion: b.FormatVersion,
		ExportedAt:    b.ExportedAt,
		Encrypted:     b.Encrypted,
		Includes:      b.Includes,
		Counts: BundleCounts{
			SystemSettings: len(b.Data.SystemSettings),
			CronJobs:       len(b.Data.CronJobs),
			Admins:         len(b.Data.Admins),
			AuditLogs:      len(b.Data.AuditLogs),
			CronJobRuns:    len(b.Data.CronJobRuns),
		},
		AdminConflicts: []string{},
	}

	existingAdmins, _ := s.store.ListAdmins(ctx)
	adSet := make(map[string]bool)
	for _, a := range existingAdmins {
		adSet[a.ID] = true
		adSet[a.Username] = true
	}
	for _, a := range b.Data.Admins {
		if adSet[a.ID] || adSet[a.Username] {
			summary.AdminConflicts = append(summary.AdminConflicts, a.Username)
		}
	}
	return summary, nil
}

func (s *Service) Import(ctx context.Context, blob []byte, opts ImportOptions) (*BundleCounts, error) {
	b, err := decodeBundle(blob, opts.Passphrase)
	if err != nil {
		return nil, err
	}

	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if mode == "" {
		mode = "merge"
	}
	if mode != "replace" && mode != "merge" {
		return nil, fmt.Errorf("invalid import mode %q (must be replace or merge)", mode)
	}
	conflict := strings.ToLower(strings.TrimSpace(opts.ConflictStrategy))
	if conflict == "" {
		conflict = "overwrite"
	}
	if conflict != "skip" && conflict != "overwrite" {
		return nil, fmt.Errorf("invalid conflict_strategy %q (must be skip or overwrite)", conflict)
	}

	tx, err := s.store.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	applied := &BundleCounts{}

	// 1. System settings
	for _, sr := range b.Data.SystemSettings {
		key := sr.Key
		valJSON := sr.ValueJSON
		if key == "client_jwt_secret_plain" {
			key = "client_jwt_secret_encrypted"
			var plain string
			if json.Unmarshal([]byte(sr.ValueJSON), &plain) == nil && plain != "" {
				if s.cipher == nil {
					return nil, errors.New("cipher required to re-encrypt client_jwt_secret")
				}
				enc, err := s.cipher.EncryptString(plain)
				if err != nil {
					return nil, fmt.Errorf("encrypt client_jwt_secret: %w", err)
				}
				bVal, _ := json.Marshal(enc)
				valJSON = string(bVal)
			} else {
				valJSON = `""`
			}
		}

		if mode == "merge" && conflict == "skip" {
			res, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO system_settings(key, value_json, updated_at, updated_by) VALUES (?, ?, ?, ?)`,
				key, valJSON, sr.UpdatedAt, sr.UpdatedBy,
			)
			if err != nil {
				return nil, err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				applied.SystemSettings++
			}
		} else {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO system_settings(key, value_json, updated_at, updated_by)
				 VALUES (?, ?, ?, ?)
				 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json, updated_at=excluded.updated_at, updated_by=excluded.updated_by`,
				key, valJSON, sr.UpdatedAt, sr.UpdatedBy,
			); err != nil {
				return nil, err
			}
			applied.SystemSettings++
		}
	}

	// 2. Cron jobs (re-encrypt safekeep_backup secrets)
	cronIDMap := make(map[string]string) // exported job id -> local job id
	for _, cj := range b.Data.CronJobs {
		cfgJSON := cj.ConfigJSON
		if cj.JobKey == "safekeep_backup" {
			reEnc, err := reencryptSafekeepConfigForImport(s.cipher, cfgJSON)
			if err != nil {
				return nil, fmt.Errorf("re-encrypt safekeep_backup config: %w", err)
			}
			cfgJSON = reEnc
		}

		var localID string
		err := tx.QueryRowContext(ctx, `SELECT id FROM cron_jobs WHERE job_key = ?`, cj.JobKey).Scan(&localID)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO cron_jobs(id, job_key, name, description, cron_expr, enabled, timeout_sec, config_json, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				cj.ID, cj.JobKey, cj.Name, cj.Description, cj.CronExpr, boolToInt(cj.Enabled), cj.TimeoutSec, cfgJSON, cj.UpdatedAt,
			); err != nil {
				return nil, err
			}
			cronIDMap[cj.ID] = cj.ID
			applied.CronJobs++
			continue
		} else if err != nil {
			return nil, err
		}

		cronIDMap[cj.ID] = localID
		if mode == "merge" && conflict == "skip" {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE cron_jobs SET cron_expr = ?, enabled = ?, timeout_sec = ?, config_json = ?, updated_at = ? WHERE id = ?`,
			cj.CronExpr, boolToInt(cj.Enabled), cj.TimeoutSec, cfgJSON, time.Now().UTC(), localID,
		); err != nil {
			return nil, err
		}
		applied.CronJobs++
	}

	// 3. Admins
	if opts.IncludeAdmins && len(b.Data.Admins) > 0 {
		if mode == "replace" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM admins`); err != nil {
				return nil, err
			}
		}
		for _, a := range b.Data.Admins {
			if mode == "merge" && conflict == "skip" {
				res, err := tx.ExecContext(ctx,
					`INSERT OR IGNORE INTO admins(id, username, password_hash, is_active, last_login_at, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?)`,
					a.ID, a.Username, a.PasswordHash, boolToInt(a.IsActive), a.LastLoginAt, a.CreatedAt, a.UpdatedAt,
				)
				if err != nil {
					return nil, err
				}
				if n, _ := res.RowsAffected(); n > 0 {
					applied.Admins++
				}
			} else {
				var existingID string
				err := tx.QueryRowContext(ctx, `SELECT id FROM admins WHERE id = ? OR username = ? LIMIT 1`, a.ID, a.Username).Scan(&existingID)
				if errors.Is(err, sql.ErrNoRows) {
					if _, err := tx.ExecContext(ctx,
						`INSERT INTO admins(id, username, password_hash, is_active, last_login_at, created_at, updated_at)
						 VALUES (?, ?, ?, ?, ?, ?, ?)`,
						a.ID, a.Username, a.PasswordHash, boolToInt(a.IsActive), a.LastLoginAt, a.CreatedAt, a.UpdatedAt,
					); err != nil {
						return nil, err
					}
					applied.Admins++
				} else if err != nil {
					return nil, err
				} else {
					if _, err := tx.ExecContext(ctx,
						`UPDATE admins SET username = ?, password_hash = ?, is_active = ?, last_login_at = ?, updated_at = ? WHERE id = ?`,
						a.Username, a.PasswordHash, boolToInt(a.IsActive), a.LastLoginAt, a.UpdatedAt, existingID,
					); err != nil {
						return nil, err
					}
					applied.Admins++
				}
			}
		}
	}

	// 4. Audit logs
	if opts.IncludeAuditLogs && len(b.Data.AuditLogs) > 0 {
		if mode == "replace" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM audit_logs`); err != nil {
				return nil, err
			}
		}
		for _, l := range b.Data.AuditLogs {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO audit_logs(actor_type, actor_id, actor_name, action, resource_type, resource_id, share_key, ip_address, user_agent, request_id, status, detail_json, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				l.ActorType, l.ActorID, l.ActorName, l.Action, l.ResourceType, l.ResourceID, l.ShareKey, l.IPAddress, l.UserAgent, l.RequestID, l.Status, l.DetailJSON, l.CreatedAt,
			); err != nil {
				return nil, err
			}
			applied.AuditLogs++
		}
	}

	// 5. Cron job runs
	if opts.IncludeCronRuns && len(b.Data.CronJobRuns) > 0 {
		if mode == "replace" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM cron_job_runs`); err != nil {
				return nil, err
			}
		}
		for _, r := range b.Data.CronJobRuns {
			targetJobID, ok := cronIDMap[r.JobID]
			if !ok {
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO cron_job_runs(id, job_id, trigger_type, triggered_by, status, affected_rows, output_log, started_at, finished_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				r.ID, targetJobID, r.TriggerType, r.TriggeredBy, r.Status, r.AffectedRows, r.OutputLog, r.StartedAt, r.FinishedAt,
			); err != nil {
				return nil, err
			}
			applied.CronJobRuns++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return applied, nil
}

type safekeepPortableCfg struct {
	APIBaseURL          string `json:"api_base_url"`
	APIKey              string `json:"api_key,omitempty"`
	APIKeyEncrypted     string `json:"api_key_encrypted,omitempty"`
	Targets             string `json:"targets"`
	Path                string `json:"path"`
	Mode                string `json:"mode"`
	Rename              string `json:"rename"`
	Passphrase          string `json:"passphrase,omitempty"`
	PassphraseEncrypted string `json:"passphrase_encrypted,omitempty"`
	IncludeAdmins       bool   `json:"include_admins"`
	IncludeAuditLogs    bool   `json:"include_audit_logs"`
	IncludeCronRuns     bool   `json:"include_cron_runs"`
}

func decryptSafekeepConfigForExport(c *crypto.Cipher, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "{}", nil
	}
	var cfg safekeepPortableCfg
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return raw, nil
	}
	if cfg.APIKeyEncrypted != "" && c != nil {
		plain, err := c.DecryptString(cfg.APIKeyEncrypted)
		if err != nil {
			return "", err
		}
		cfg.APIKey = plain
	}
	cfg.APIKeyEncrypted = ""
	if cfg.PassphraseEncrypted != "" && c != nil {
		plain, err := c.DecryptString(cfg.PassphraseEncrypted)
		if err != nil {
			return "", err
		}
		cfg.Passphrase = plain
	}
	cfg.PassphraseEncrypted = ""
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func reencryptSafekeepConfigForImport(c *crypto.Cipher, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "{}", nil
	}
	var cfg safekeepPortableCfg
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return raw, nil
	}
	if cfg.APIKey != "" && c != nil {
		enc, err := c.EncryptString(cfg.APIKey)
		if err != nil {
			return "", err
		}
		cfg.APIKeyEncrypted = enc
	}
	cfg.APIKey = ""
	if cfg.Passphrase != "" && c != nil {
		enc, err := c.EncryptString(cfg.Passphrase)
		if err != nil {
			return "", err
		}
		cfg.PassphraseEncrypted = enc
	}
	cfg.Passphrase = ""
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
