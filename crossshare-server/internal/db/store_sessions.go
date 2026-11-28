package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

type AdminSession struct {
	ID         string
	AdminID    string
	TokenHash  string
	ClientIP   string
	UserAgent  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

func AdminTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func insertAdminSession(ctx context.Context, tx *sql.Tx, session *AdminSession) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO admin_sessions
		(id, admin_id, token_hash, client_ip, user_agent, created_at, last_seen_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, session.ID, session.AdminID, session.TokenHash,
		session.ClientIP, session.UserAgent, session.CreatedAt, session.LastSeenAt, session.ExpiresAt)
	return err
}

func (s *Store) CreateAdminSession(ctx context.Context, admin *Admin, session *AdminSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 将账号版本校验和会话写入放在同一事务，避免改密或停用时产生旧凭证。
	var id string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM admins WHERE id = ? AND password_hash = ? AND is_active = 1 AND role = ?`,
		admin.ID, admin.PasswordHash, admin.Role).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions WHERE expires_at <= ?`, time.Now().UTC()); err != nil {
		return err
	}
	if err := insertAdminSession(ctx, tx, session); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetAdminSession(ctx context.Context, token string) (*AdminSession, error) {
	var session AdminSession
	err := s.db.QueryRowContext(ctx, `SELECT id, admin_id, token_hash, client_ip, user_agent, created_at, last_seen_at, expires_at
		FROM admin_sessions WHERE token_hash = ?`, AdminTokenHash(token)).Scan(&session.ID, &session.AdminID,
		&session.TokenHash, &session.ClientIP, &session.UserAgent, &session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &session, err
}

func (s *Store) ListAdminSessions(ctx context.Context, adminID string, now time.Time) ([]AdminSession, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, admin_id, client_ip, user_agent, created_at, last_seen_at, expires_at
		FROM admin_sessions WHERE admin_id = ? AND expires_at > ? ORDER BY created_at DESC`, adminID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AdminSession, 0)
	for rows.Next() {
		var session AdminSession
		if err := rows.Scan(&session.ID, &session.AdminID, &session.ClientIP, &session.UserAgent,
			&session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

func (s *Store) TouchAdminSession(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE admin_sessions SET last_seen_at = ? WHERE id = ?`, now, id)
	return err
}

func (s *Store) DeleteAdminSession(ctx context.Context, adminID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE id = ? AND admin_id = ?`, id, adminID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ChangeAdminPasswordWithSession(ctx context.Context, admin *Admin, hash string, session *AdminSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE admins SET password_hash = ?, must_change_password = 0, updated_at = ?
		WHERE id = ? AND password_hash = ? AND is_active = 1`, hash, time.Now().UTC(), admin.ID, admin.PasswordHash)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions WHERE admin_id = ?`, admin.ID); err != nil {
		return err
	}
	if err := insertAdminSession(ctx, tx, session); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateAdminAccount(ctx context.Context, actor *Admin, id, role string, active bool, hash string, replacement *AdminSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 在同一事务中重读操作者并保护最后一个启用的管理员，避免并发操作绕过约束。
	var actorID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM admins WHERE id = ? AND role = 'admin' AND is_active = 1 AND password_hash = ?`,
		actor.ID, actor.PasswordHash).Scan(&actorID); err != nil {
		return err
	}
	var oldRole, oldHash string
	var oldActive bool
	if err := tx.QueryRowContext(ctx, `SELECT role, is_active, password_hash FROM admins WHERE id = ?`, id).Scan(&oldRole, &oldActive, &oldHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if oldRole == RoleAdmin && oldActive && (role != RoleAdmin || !active) {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins WHERE role = 'admin' AND is_active = 1`).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}
	if hash == "" {
		hash = oldHash
	}
	if _, err := tx.ExecContext(ctx, `UPDATE admins SET role = ?, is_active = ?, password_hash = ?,
		must_change_password = CASE WHEN password_hash != ? THEN 0 ELSE must_change_password END, updated_at = ? WHERE id = ?`,
		role, boolToInt(active), hash, hash, time.Now().UTC(), id); err != nil {
		return err
	}
	if oldRole != role || !active || oldHash != hash {
		if _, err := tx.ExecContext(ctx, `DELETE FROM admin_sessions WHERE admin_id = ?`, id); err != nil {
			return err
		}
		if replacement != nil {
			if err := insertAdminSession(ctx, tx, replacement); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) CreateAdminByActor(ctx context.Context, actor, account *Admin) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var actorID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM admins WHERE id = ? AND role = 'admin' AND is_active = 1 AND password_hash = ?`,
		actor.ID, actor.PasswordHash).Scan(&actorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins WHERE username = ?`, account.Username).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return ErrUsernameTaken
	}
	now := time.Now().UTC()
	account.CreatedAt, account.UpdatedAt = now, now
	if _, err := tx.ExecContext(ctx, `INSERT INTO admins(id, username, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, account.ID, account.Username, account.PasswordHash, account.Role,
		boolToInt(account.IsActive), account.CreatedAt, account.UpdatedAt); err != nil {
		return err
	}
	return tx.Commit()
}
