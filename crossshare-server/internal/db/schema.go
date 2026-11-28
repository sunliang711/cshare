package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	_ "modernc.org/sqlite"

	"crossshare-server/internal/config"
	"crossshare-server/internal/crypto"
)

var Module = fx.Options(
	fx.Provide(ProvideStore),
)

type Store struct {
	db      *sql.DB
	dataDir string
}

func ProvideStore(lc fx.Lifecycle, cfg *config.Config, logger zerolog.Logger) (*Store, error) {
	dataDir := strings.TrimSpace(cfg.Server.DataDir)
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data_dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "crossshare.db")
	store, err := Open(dbPath, dataDir)
	if err != nil {
		return nil, err
	}
	_, _ = store.MarkInterruptedRunsFailed(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info().Str("db_path", dbPath).Msg("sqlite store ready")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return store.Close()
		},
	})
	return store, nil
}

func Open(dbPath, dataDir string) (*Store, error) {
	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqldb.SetMaxOpenConns(1)
	sqldb.SetMaxIdleConns(1)
	sqldb.SetConnMaxLifetime(0)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, p := range pragmas {
		if _, err := sqldb.Exec(p); err != nil {
			_ = sqldb.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	s := &Store{db: sqldb, dataDir: dataDir}
	if err := s.migrate(context.Background()); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	if err := s.seedDefaults(context.Background()); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("seed defaults: %w", err)
	}
	return s, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) DataDir() string {
	return s.dataDir
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			must_change_password INTEGER NOT NULL DEFAULT 0,
			last_login_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS admin_sessions (
			id TEXT PRIMARY KEY,
			admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			client_ip TEXT NOT NULL,
			user_agent TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			last_seen_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_sessions_admin ON admin_sessions(admin_id, expires_at);`,
		`CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at DATETIME NOT NULL,
			updated_by TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			actor_type TEXT NOT NULL,
			actor_id TEXT NOT NULL DEFAULT '',
			actor_name TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL DEFAULT '',
			resource_id TEXT NOT NULL DEFAULT '',
			share_key TEXT NOT NULL DEFAULT '',
			ip_address TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'success',
			detail_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_action ON audit_logs(actor_type, action);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_share_key ON audit_logs(share_key);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_ip ON audit_logs(ip_address);`,
		`CREATE TABLE IF NOT EXISTS cron_jobs (
			id TEXT PRIMARY KEY,
			job_key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			cron_expr TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			timeout_sec INTEGER NOT NULL DEFAULT 60,
			config_json TEXT NOT NULL DEFAULT '{}',
			last_run_at DATETIME,
			last_status TEXT NOT NULL DEFAULT '',
			last_duration_ms INTEGER NOT NULL DEFAULT 0,
			last_summary TEXT NOT NULL DEFAULT '',
			next_run_at DATETIME,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS cron_job_runs (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES cron_jobs(id) ON DELETE CASCADE,
			trigger_type TEXT NOT NULL,
			triggered_by TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			affected_rows INTEGER NOT NULL DEFAULT 0,
			output_log TEXT NOT NULL DEFAULT '',
			started_at DATETIME NOT NULL,
			finished_at DATETIME
		);`,
		`CREATE INDEX IF NOT EXISTS idx_cron_job_runs_job_id ON cron_job_runs(job_id, started_at DESC);`,
		`CREATE TABLE IF NOT EXISTS stats_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bucket_at DATETIME NOT NULL UNIQUE,
			active_shares INTEGER NOT NULL DEFAULT 0,
			total_stored_bytes INTEGER NOT NULL DEFAULT 0,
			push_count INTEGER NOT NULL DEFAULT 0,
			pull_count INTEGER NOT NULL DEFAULT 0,
			delete_count INTEGER NOT NULL DEFAULT 0,
			p2p_sessions INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(40, len(stmt))], err)
		}
	}
	if err := s.ensureColumn(ctx, "admins", "must_change_password", `ALTER TABLE admins ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("migrate admins.must_change_password: %w", err)
	}
	if err := s.ensureColumn(ctx, "admins", "role", `ALTER TABLE admins ADD COLUMN role TEXT NOT NULL DEFAULT 'admin' CHECK(role IN ('admin', 'viewer'))`); err != nil {
		return fmt.Errorf("migrate admins.role: %w", err)
	}
	return nil
}

func (s *Store) ensureColumn(ctx context.Context, table, column, alterDDL string) error {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &primaryKey); err != nil {
			return err
		}
		if strings.EqualFold(name, column) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, alterDDL)
	return err
}

func (s *Store) seedDefaults(ctx context.Context) error {
	now := time.Now().UTC()
	d := DefaultSystemSettings()
	defaults := map[string]any{
		"default_ttl":                 d.DefaultTTL,
		"max_ttl":                     d.MaxTTL,
		"text_json_limit":             d.TextJSONLimit,
		"files_push_limit":            d.FilesPushLimit,
		"max_files_per_push":          d.MaxFilesPerPush,
		"ratelimit_enable":            d.RateLimitEnable,
		"ratelimit_rpm":               d.RateLimitRPM,
		"client_auth_enable":          d.ClientAuthEnable,
		"client_jwt_secret_encrypted": "",
		"audit_client_ops":            d.AuditClientOps,
		"audit_retention_days":        d.AuditRetentionDays,
	}
	for k, v := range defaults {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO system_settings(key, value_json, updated_at, updated_by) VALUES (?, ?, ?, 'system')`,
			k, string(raw), now,
		); err != nil {
			return err
		}
	}

	type cronSeed struct {
		key        string
		name       string
		desc       string
		expr       string
		enabled    int
		timeoutSec int
		configJSON string
	}
	cronSeeds := []cronSeed{
		{
			key:        "cleanup_audit_logs",
			name:       "清理过期审计日志",
			desc:       "定期清理超过保留天数的历史审计日志",
			expr:       "0 2 * * *",
			enabled:    1,
			timeoutSec: 120,
			configJSON: `{"retention_days":90}`,
		},
		{
			key:        "cleanup_cron_runs",
			name:       "清理旧定时任务记录",
			desc:       "清理过期或超出保留条数的定时任务执行日志",
			expr:       "0 3 * * *",
			enabled:    1,
			timeoutSec: 120,
			configJSON: `{"retention_days":30,"keep_per_job":200}`,
		},
		{
			key:        "storage_health_check",
			name:       "存储健康巡检",
			desc:       "定期检测当前存储后端（Redis / 内存）连通性与响应延迟",
			expr:       "*/5 * * * *",
			enabled:    1,
			timeoutSec: 30,
			configJSON: `{}`,
		},
		{
			key:        "stats_snapshot",
			name:       "统计数据快照",
			desc:       "每小时汇总活跃分享数、占用容量与推送/拉取/删除次数，供仪表盘趋势图展示",
			expr:       "0 * * * *",
			enabled:    1,
			timeoutSec: 60,
			configJSON: `{}`,
		},
		{
			key:        "safekeep_backup",
			name:       "备份环境到 SafeKeep",
			desc:       "按计划生成环境迁移包（系统设置、定时任务及可选管理员/审计日志）并自动上传至 SafeKeep",
			expr:       "0 3 * * 0",
			enabled:    0,
			timeoutSec: 300,
			configJSON: `{"api_base_url":"","targets":"","path":"cshare-backups","mode":"sync","include_admins":true,"include_audit_logs":false,"include_cron_runs":false}`,
		},
	}

	for _, cs := range cronSeeds {
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO cron_jobs(id, job_key, name, description, cron_expr, enabled, timeout_sec, config_json, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			crypto.NewULID(), cs.key, cs.name, cs.desc, cs.expr, cs.enabled, cs.timeoutSec, cs.configJSON, now,
		); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
