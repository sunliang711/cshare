package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) ListCronJobs(ctx context.Context) ([]CronJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_key, name, description, cron_expr, enabled, timeout_sec, config_json,
		        last_run_at, last_status, last_duration_ms, last_summary, next_run_at, updated_at
		 FROM cron_jobs ORDER BY job_key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]CronJob, 0)
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *j)
	}
	return jobs, rows.Err()
}

func (s *Store) GetCronJobByID(ctx context.Context, id string) (*CronJob, error) {
	return scanCronJob(s.db.QueryRowContext(ctx,
		`SELECT id, job_key, name, description, cron_expr, enabled, timeout_sec, config_json,
		        last_run_at, last_status, last_duration_ms, last_summary, next_run_at, updated_at
		 FROM cron_jobs WHERE id = ?`, id))
}

func (s *Store) GetCronJobByKey(ctx context.Context, key string) (*CronJob, error) {
	return scanCronJob(s.db.QueryRowContext(ctx,
		`SELECT id, job_key, name, description, cron_expr, enabled, timeout_sec, config_json,
		        last_run_at, last_status, last_duration_ms, last_summary, next_run_at, updated_at
		 FROM cron_jobs WHERE job_key = ?`, key))
}

func (s *Store) UpdateCronJob(ctx context.Context, id, cronExpr string, enabled bool, timeoutSec int, configJSON string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE cron_jobs SET cron_expr = ?, enabled = ?, timeout_sec = ?, config_json = ?, updated_at = ? WHERE id = ?`,
		cronExpr, boolToInt(enabled), timeoutSec, configJSON, time.Now().UTC(), id,
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

func (s *Store) UpdateCronJobRunState(ctx context.Context, id string, lastRunAt time.Time, lastStatus string, durationMs int64, summary string, nextRunAt *time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_jobs SET last_run_at = ?, last_status = ?, last_duration_ms = ?, last_summary = ?, next_run_at = ?, updated_at = ? WHERE id = ?`,
		lastRunAt, lastStatus, durationMs, summary, nextRunAt, time.Now().UTC(), id,
	)
	return err
}

func (s *Store) UpdateCronJobNextRun(ctx context.Context, id string, nextRunAt *time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE cron_jobs SET next_run_at = ? WHERE id = ?`, nextRunAt, id)
	return err
}

func (s *Store) InsertCronJobRun(ctx context.Context, r *CronJobRun) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cron_job_runs(id, job_id, trigger_type, triggered_by, status, affected_rows, output_log, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.JobID, r.TriggerType, r.TriggeredBy, r.Status, r.AffectedRows, r.OutputLog, r.StartedAt, r.FinishedAt,
	)
	return err
}

func (s *Store) FinishCronJobRun(ctx context.Context, id, status string, affectedRows int64, outputLog string, finishedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cron_job_runs SET status = ?, affected_rows = ?, output_log = ?, finished_at = ? WHERE id = ?`,
		status, affectedRows, outputLog, finishedAt, id,
	)
	return err
}

func (s *Store) ListCronJobRuns(ctx context.Context, jobID string, limit int) ([]CronJobRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_id, trigger_type, triggered_by, status, affected_rows, output_log, started_at, finished_at
		 FROM cron_job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT ?`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]CronJobRun, 0)
	for rows.Next() {
		var r CronJobRun
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

func (s *Store) DeleteCronJobRunsBefore(ctx context.Context, cutoff time.Time, keepPerJob int) (int64, error) {
	var totalDeleted int64
	res1, err := s.db.ExecContext(ctx, `DELETE FROM cron_job_runs WHERE started_at < ? AND status != 'running'`, cutoff)
	if err != nil {
		return 0, err
	}
	n1, _ := res1.RowsAffected()
	totalDeleted += n1

	if keepPerJob > 0 {
		res2, err := s.db.ExecContext(ctx,
			`DELETE FROM cron_job_runs
			 WHERE status != 'running' AND id NOT IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY started_at DESC) AS rn
					FROM cron_job_runs
				) WHERE rn <= ?
			 )`, keepPerJob)
		if err == nil {
			n2, _ := res2.RowsAffected()
			totalDeleted += n2
		}
	}
	return totalDeleted, nil
}

func (s *Store) MarkInterruptedRunsFailed(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE cron_job_runs SET status = 'failed', output_log = 'interrupted by server shutdown/restart', finished_at = ? WHERE status = 'running'`,
		now,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanCronJob(row rowScanner) (*CronJob, error) {
	var j CronJob
	var enabled int
	var lastRun, nextRun sql.NullTime
	err := row.Scan(
		&j.ID, &j.JobKey, &j.Name, &j.Description, &j.CronExpr, &enabled, &j.TimeoutSec, &j.ConfigJSON,
		&lastRun, &j.LastStatus, &j.LastDurationMs, &j.LastSummary, &nextRun, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	j.Enabled = enabled != 0
	if lastRun.Valid {
		t := lastRun.Time
		j.LastRunAt = &t
	}
	if nextRun.Valid {
		t := nextRun.Time
		j.NextRunAt = &t
	}
	return &j, nil
}
