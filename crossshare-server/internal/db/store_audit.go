package db

import (
	"context"
	"strings"
	"time"
)

func (s *Store) InsertAuditLog(ctx context.Context, l *AuditLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	if l.Status == "" {
		l.Status = "success"
	}
	if l.DetailJSON == "" {
		l.DetailJSON = "{}"
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs(actor_type, actor_id, actor_name, action, resource_type, resource_id, share_key, ip_address, user_agent, request_id, status, detail_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ActorType, l.ActorID, l.ActorName, l.Action, l.ResourceType, l.ResourceID,
		l.ShareKey, l.IPAddress, l.UserAgent, l.RequestID, l.Status, l.DetailJSON, l.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	l.ID = id
	return nil
}

func buildAuditWhere(f AuditFilter) (string, []any) {
	var conds []string
	var args []any

	if f.ActorType != "" {
		conds = append(conds, "actor_type = ?")
		args = append(args, f.ActorType)
	}
	if f.Action != "" {
		conds = append(conds, "action = ?")
		args = append(args, f.Action)
	}
	if f.ShareKey != "" {
		conds = append(conds, "share_key = ?")
		args = append(args, f.ShareKey)
	}
	if f.IPAddress != "" {
		conds = append(conds, "ip_address LIKE ?")
		args = append(args, "%"+f.IPAddress+"%")
	}
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, f.Status)
	}
	if f.Query != "" {
		q := "%" + f.Query + "%"
		conds = append(conds, "(actor_name LIKE ? OR action LIKE ? OR share_key LIKE ? OR resource_id LIKE ? OR ip_address LIKE ? OR request_id LIKE ? OR detail_json LIKE ?)")
		args = append(args, q, q, q, q, q, q, q)
	}
	if f.From != nil {
		conds = append(conds, "created_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		conds = append(conds, "created_at <= ?")
		args = append(args, *f.To)
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (s *Store) ListAuditLogs(ctx context.Context, f AuditFilter) ([]AuditLog, int64, error) {
	where, args := buildAuditWhere(f)

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, actor_type, actor_id, actor_name, action, resource_type, resource_id, share_key, ip_address, user_agent, request_id, status, detail_json, created_at
		FROM audit_logs` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any(nil), args...), limit, offset)

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]AuditLog, 0)
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID, &l.ActorType, &l.ActorID, &l.ActorName, &l.Action,
			&l.ResourceType, &l.ResourceID, &l.ShareKey, &l.IPAddress,
			&l.UserAgent, &l.RequestID, &l.Status, &l.DetailJSON, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, l)
	}
	return list, total, rows.Err()
}

func (s *Store) ExportAllAuditLogs(ctx context.Context, f AuditFilter) ([]AuditLog, error) {
	where, args := buildAuditWhere(f)
	query := `SELECT id, actor_type, actor_id, actor_name, action, resource_type, resource_id, share_key, ip_address, user_agent, request_id, status, detail_json, created_at
		FROM audit_logs` + where + ` ORDER BY id DESC LIMIT 10000`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]AuditLog, 0)
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID, &l.ActorType, &l.ActorID, &l.ActorName, &l.Action,
			&l.ResourceType, &l.ResourceID, &l.ShareKey, &l.IPAddress,
			&l.UserAgent, &l.RequestID, &l.Status, &l.DetailJSON, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (s *Store) DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM audit_logs WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) CountClientActionsSince(ctx context.Context, since time.Time) (ActionCounts, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT action, COUNT(*) FROM audit_logs
		 WHERE actor_type = 'client' AND status = 'success' AND created_at >= ?
		 GROUP BY action`, since)
	if err != nil {
		return ActionCounts{}, err
	}
	defer rows.Close()

	var counts ActionCounts
	for rows.Next() {
		var action string
		var cnt int64
		if err := rows.Scan(&action, &cnt); err != nil {
			return counts, err
		}
		switch {
		case strings.HasPrefix(action, "push."):
			counts.PushCount += cnt
		case action == "pull":
			counts.PullCount += cnt
		case action == "delete":
			counts.DeleteCount += cnt
		}
	}
	return counts, rows.Err()
}
