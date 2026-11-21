package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&n)
	return n, err
}

func (s *Store) CreateInitialAdminIfEmpty(ctx context.Context, a *Admin) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var n int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrAlreadyInitialized
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO admins(id, username, password_hash, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.Username, a.PasswordHash, boolToInt(a.IsActive), a.CreatedAt, a.UpdatedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateAdmin(ctx context.Context, a *Admin) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admins(id, username, password_hash, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.Username, a.PasswordHash, boolToInt(a.IsActive), a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	return s.scanAdmin(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, is_active, last_login_at, created_at, updated_at
		 FROM admins WHERE username = ?`, username))
}

func (s *Store) GetAdminByID(ctx context.Context, id string) (*Admin, error) {
	return s.scanAdmin(s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, is_active, last_login_at, created_at, updated_at
		 FROM admins WHERE id = ?`, id))
}

func (s *Store) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, is_active, last_login_at, created_at, updated_at
		 FROM admins ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Admin, 0)
	for rows.Next() {
		a, err := s.scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *a)
	}
	return list, rows.Err()
}

func (s *Store) UpdateAdmin(ctx context.Context, id string, isActive bool, passwordHash string) error {
	now := time.Now().UTC()
	var res sql.Result
	var err error
	if passwordHash != "" {
		res, err = s.db.ExecContext(ctx,
			`UPDATE admins SET is_active = ?, password_hash = ?, updated_at = ? WHERE id = ?`,
			boolToInt(isActive), passwordHash, now, id,
		)
	} else {
		res, err = s.db.ExecContext(ctx,
			`UPDATE admins SET is_active = ?, updated_at = ? WHERE id = ?`,
			boolToInt(isActive), now, id,
		)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateAdminLogin(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE admins SET last_login_at = ?, updated_at = ? WHERE id = ?`,
		now, now, id,
	)
	return err
}

func (s *Store) UpdateAdminPassword(ctx context.Context, id, passwordHash string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE admins SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, now, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) scanAdmin(row rowScanner) (*Admin, error) {
	var a Admin
	var isActive int
	var lastLogin sql.NullTime
	err := row.Scan(
		&a.ID, &a.Username, &a.PasswordHash, &isActive,
		&lastLogin, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.IsActive = isActive != 0
	if lastLogin.Valid {
		t := lastLogin.Time
		a.LastLoginAt = &t
	}
	return &a, nil
}
