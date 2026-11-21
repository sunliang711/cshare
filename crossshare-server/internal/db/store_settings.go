package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crossshare-server/internal/crypto"
)

func (s *Store) GetSystemSettings(ctx context.Context, cipher *crypto.Cipher) (SystemSettings, error) {
	settings := DefaultSystemSettings()
	rows, err := s.db.QueryContext(ctx, `SELECT key, value_json FROM system_settings`)
	if err != nil {
		return settings, err
	}
	defer rows.Close()

	var encSecret string
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return settings, err
		}
		switch k {
		case "default_ttl":
			_ = json.Unmarshal([]byte(v), &settings.DefaultTTL)
		case "max_ttl":
			_ = json.Unmarshal([]byte(v), &settings.MaxTTL)
		case "text_json_limit":
			_ = json.Unmarshal([]byte(v), &settings.TextJSONLimit)
		case "files_push_limit":
			_ = json.Unmarshal([]byte(v), &settings.FilesPushLimit)
		case "max_files_per_push":
			_ = json.Unmarshal([]byte(v), &settings.MaxFilesPerPush)
		case "ratelimit_enable":
			_ = json.Unmarshal([]byte(v), &settings.RateLimitEnable)
		case "ratelimit_rpm":
			_ = json.Unmarshal([]byte(v), &settings.RateLimitRPM)
		case "client_auth_enable":
			_ = json.Unmarshal([]byte(v), &settings.ClientAuthEnable)
		case "client_jwt_secret_encrypted":
			_ = json.Unmarshal([]byte(v), &encSecret)
		case "audit_client_ops":
			_ = json.Unmarshal([]byte(v), &settings.AuditClientOps)
		case "audit_retention_days":
			_ = json.Unmarshal([]byte(v), &settings.AuditRetentionDays)
		}
	}
	if err := rows.Err(); err != nil {
		return settings, err
	}

	if encSecret != "" {
		settings.HasClientJWTSecret = true
		if cipher != nil {
			plain, err := cipher.DecryptString(encSecret)
			if err != nil {
				return settings, fmt.Errorf("decrypt client_jwt_secret: %w", err)
			}
			settings.ClientJWTSecret = plain
		}
	}
	return settings, nil
}

func (s *Store) UpdateSystemSettings(ctx context.Context, cipher *crypto.Cipher, settings SystemSettings, updatedBy string) error {
	encSecret := ""
	if settings.ClientJWTSecret != "" {
		if cipher == nil {
			return fmt.Errorf("cipher required to encrypt client_jwt_secret")
		}
		var err error
		encSecret, err = cipher.EncryptString(settings.ClientJWTSecret)
		if err != nil {
			return fmt.Errorf("encrypt client_jwt_secret: %w", err)
		}
	}

	pairs := map[string]any{
		"default_ttl":                 settings.DefaultTTL,
		"max_ttl":                     settings.MaxTTL,
		"text_json_limit":             settings.TextJSONLimit,
		"files_push_limit":            settings.FilesPushLimit,
		"max_files_per_push":          settings.MaxFilesPerPush,
		"ratelimit_enable":            settings.RateLimitEnable,
		"ratelimit_rpm":               settings.RateLimitRPM,
		"client_auth_enable":          settings.ClientAuthEnable,
		"client_jwt_secret_encrypted": encSecret,
		"audit_client_ops":            settings.AuditClientOps,
		"audit_retention_days":        settings.AuditRetentionDays,
	}

	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for k, v := range pairs {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO system_settings(key, value_json, updated_at, updated_by)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json, updated_at=excluded.updated_at, updated_by=excluded.updated_by`,
			k, string(raw), now, updatedBy,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
