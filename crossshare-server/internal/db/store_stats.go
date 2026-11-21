package db

import (
	"context"
	"time"
)

func (s *Store) UpsertStatsSnapshot(ctx context.Context, snap *StatsSnapshot) error {
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stats_snapshots(bucket_at, active_shares, total_stored_bytes, push_count, pull_count, delete_count, p2p_sessions, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(bucket_at) DO UPDATE SET
			active_shares=excluded.active_shares,
			total_stored_bytes=excluded.total_stored_bytes,
			push_count=excluded.push_count,
			pull_count=excluded.pull_count,
			delete_count=excluded.delete_count,
			p2p_sessions=excluded.p2p_sessions`,
		snap.BucketAt, snap.ActiveShares, snap.TotalStoredBytes,
		snap.PushCount, snap.PullCount, snap.DeleteCount, snap.P2PSessions, snap.CreatedAt,
	)
	return err
}

func (s *Store) ListStatsSnapshots(ctx context.Context, limit int) ([]StatsSnapshot, error) {
	if limit <= 0 || limit > 720 {
		limit = 24
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, bucket_at, active_shares, total_stored_bytes, push_count, pull_count, delete_count, p2p_sessions, created_at
		 FROM stats_snapshots ORDER BY bucket_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]StatsSnapshot, 0)
	for rows.Next() {
		var sn StatsSnapshot
		if err := rows.Scan(
			&sn.ID, &sn.BucketAt, &sn.ActiveShares, &sn.TotalStoredBytes,
			&sn.PushCount, &sn.PullCount, &sn.DeleteCount, &sn.P2PSessions, &sn.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, sn)
	}
	// Reverse to chronological order for charts
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, rows.Err()
}
