package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/safekeep"
	"crossshare-server/internal/service"
	"crossshare-server/internal/settings"
	"crossshare-server/internal/storage"
)

var Module = fx.Options(
	fx.Provide(ProvideEngine),
)

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

type jobMeta struct {
	DefaultSchedule string
	DefaultTimeout  int
}

var builtinJobMeta = map[string]jobMeta{
	"cleanup_audit_logs":   {DefaultSchedule: "0 2 * * *", DefaultTimeout: 120},
	"cleanup_cron_runs":    {DefaultSchedule: "0 3 * * *", DefaultTimeout: 120},
	"storage_health_check": {DefaultSchedule: "*/5 * * * *", DefaultTimeout: 30},
	"stats_snapshot":       {DefaultSchedule: "0 * * * *", DefaultTimeout: 60},
	"safekeep_backup":      {DefaultSchedule: "0 3 * * 0", DefaultTimeout: 300},
}

func DefaultScheduleFor(jobKey string) string {
	if m, ok := builtinJobMeta[jobKey]; ok {
		return m.DefaultSchedule
	}
	return "0 3 * * *"
}

func DefaultTimeoutFor(jobKey string) int {
	if m, ok := builtinJobMeta[jobKey]; ok && m.DefaultTimeout > 0 {
		return m.DefaultTimeout
	}
	return 120
}

func ValidateCron(expr string) error {
	_, err := PreviewCron(expr, 1)
	return err
}

func PreviewCron(expr string, count int) ([]time.Time, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, errors.New("cron expression is required")
	}
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 5
	}
	out := make([]time.Time, 0, count)
	at := time.Now()
	for i := 0; i < count; i++ {
		at = sched.Next(at)
		if at.IsZero() {
			break
		}
		out = append(out, at)
	}
	return out, nil
}

func NormalizeJobConfig(jobKey string, incoming json.RawMessage) (string, error) {
	var raw map[string]any
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &raw); err != nil {
			return "", fmt.Errorf("invalid config JSON: %w", err)
		}
	}
	switch jobKey {
	case "cleanup_audit_logs":
		days := intFromMap(raw, "retention_days", 90)
		if days < 1 || days > 36500 {
			return "", errors.New("retention_days must be between 1 and 36500")
		}
		b, _ := json.Marshal(map[string]int{"retention_days": days})
		return string(b), nil
	case "cleanup_cron_runs":
		days := intFromMap(raw, "retention_days", 30)
		keepPerJob := intFromMap(raw, "keep_per_job", 200)
		if days < 1 || days > 36500 {
			return "", errors.New("retention_days must be between 1 and 36500")
		}
		if keepPerJob < 1 || keepPerJob > 10000 {
			return "", errors.New("keep_per_job must be between 1 and 10000")
		}
		b, _ := json.Marshal(map[string]int{
			"retention_days": days,
			"keep_per_job":   keepPerJob,
		})
		return string(b), nil
	case "storage_health_check", "stats_snapshot":
		return "{}", nil
	default:
		if len(incoming) == 0 {
			return "{}", nil
		}
		return string(incoming), nil
	}
}

type Engine struct {
	store    *db.Store
	storage  storage.Storage
	p2p      *service.P2PService
	settings *settings.Manager
	audit    *audit.Recorder
	cipher   *crypto.Cipher
	migSvc   *migration.Service
	skClient *safekeep.Client
	logger   zerolog.Logger

	mu         sync.Mutex
	c          *cron.Cron
	entryIDs   map[string]cron.EntryID
	running    map[string]bool
	appCtx     context.Context
	cancelFunc context.CancelFunc
}

type EngineParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Store     *db.Store
	Storage   storage.Storage
	P2P       *service.P2PService
	Settings  *settings.Manager
	Audit     *audit.Recorder
	Cipher    *crypto.Cipher
	Migration *migration.Service
	Logger    zerolog.Logger
}

func ProvideEngine(p EngineParams) *Engine {
	eng := NewEngine(p.Store, p.Storage, p.P2P, p.Settings, p.Audit, p.Cipher, p.Migration, safekeep.NewClient(), p.Logger)
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return eng.Start()
		},
		OnStop: func(ctx context.Context) error {
			eng.Stop()
			return nil
		},
	})
	return eng
}

func NewEngine(
	store *db.Store,
	st storage.Storage,
	p2p *service.P2PService,
	settingsMgr *settings.Manager,
	auditRec *audit.Recorder,
	cipher *crypto.Cipher,
	migSvc *migration.Service,
	skClient *safekeep.Client,
	logger zerolog.Logger,
) *Engine {
	appCtx, cancel := context.WithCancel(context.Background())
	return &Engine{
		store:      store,
		storage:    st,
		p2p:        p2p,
		settings:   settingsMgr,
		audit:      auditRec,
		cipher:     cipher,
		migSvc:     migSvc,
		skClient:   skClient,
		logger:     logger.With().Str("component", "scheduler").Logger(),
		entryIDs:   make(map[string]cron.EntryID),
		running:    make(map[string]bool),
		appCtx:     appCtx,
		cancelFunc: cancel,
	}
}

func (e *Engine) Start() error {
	e.mu.Lock()
	e.c = cron.New(cron.WithParser(cronParser))
	e.c.Start()
	e.mu.Unlock()

	return e.ReloadAll(e.appCtx)
}

func (e *Engine) Stop() {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	e.mu.Lock()
	c := e.c
	e.mu.Unlock()
	if c != nil {
		ctx := c.Stop()
		<-ctx.Done()
	}
}

func (e *Engine) ReloadAll(ctx context.Context) error {
	jobs, err := e.store.ListCronJobs(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.c == nil {
		return nil
	}
	for id, entryID := range e.entryIDs {
		e.c.Remove(entryID)
		delete(e.entryIDs, id)
	}

	for _, job := range jobs {
		if !job.Enabled {
			_ = e.store.UpdateCronJobNextRun(ctx, job.ID, nil)
			continue
		}
		j := job
		entryID, err := e.c.AddFunc(j.CronExpr, func() {
			e.executeJob(e.appCtx, j.ID, "schedule", "scheduler")
		})
		if err != nil {
			e.logger.Error().Err(err).Str("job_key", j.JobKey).Str("expr", j.CronExpr).Msg("register cron job failed")
			continue
		}
		e.entryIDs[j.ID] = entryID
		entry := e.c.Entry(entryID)
		if !entry.Next.IsZero() {
			nxt := entry.Next.UTC()
			_ = e.store.UpdateCronJobNextRun(ctx, j.ID, &nxt)
		}
	}
	return nil
}

func (e *Engine) RunNow(_ context.Context, jobID, actorName string) (*db.CronJobRun, error) {
	if _, err := e.store.GetCronJobByID(e.appCtx, jobID); err != nil {
		return nil, err
	}
	run := e.executeJob(e.appCtx, jobID, "manual", actorName)
	return run, nil
}

func (e *Engine) executeJob(ctx context.Context, jobID, triggerType, triggeredBy string) *db.CronJobRun {
	e.mu.Lock()
	if e.running[jobID] {
		e.mu.Unlock()
		return &db.CronJobRun{
			JobID:       jobID,
			TriggerType: triggerType,
			TriggeredBy: triggeredBy,
			Status:      "failed",
			OutputLog:   "job is already running",
			StartedAt:   time.Now().UTC(),
		}
	}
	e.running[jobID] = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, jobID)
		e.mu.Unlock()
	}()

	job, err := e.store.GetCronJobByID(ctx, jobID)
	if err != nil {
		return &db.CronJobRun{
			JobID:       jobID,
			TriggerType: triggerType,
			TriggeredBy: triggeredBy,
			Status:      "failed",
			OutputLog:   err.Error(),
			StartedAt:   time.Now().UTC(),
		}
	}

	timeoutSec := job.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = DefaultTimeoutFor(job.JobKey)
	}
	jobCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	startedAt := time.Now().UTC()
	run := &db.CronJobRun{
		ID:          crypto.NewULID(),
		JobID:       job.ID,
		TriggerType: triggerType,
		TriggeredBy: triggeredBy,
		Status:      "running",
		StartedAt:   startedAt,
	}
	_ = e.store.InsertCronJobRun(ctx, run)

	affected, summary, jobErr := e.dispatchJob(jobCtx, job)
	finishedAt := time.Now().UTC()
	durationMs := finishedAt.Sub(startedAt).Milliseconds()
	status := "succeeded"
	output := summary
	if jobErr != nil {
		status = "failed"
		output = jobErr.Error()
		e.logger.Error().Err(jobErr).Str("job_key", job.JobKey).Msg("cron job failed")
	}

	_ = e.store.FinishCronJobRun(ctx, run.ID, status, affected, output, finishedAt)

	var nextRunAt *time.Time
	e.mu.Lock()
	if entryID, ok := e.entryIDs[job.ID]; ok && e.c != nil {
		entry := e.c.Entry(entryID)
		if !entry.Next.IsZero() {
			nxt := entry.Next.UTC()
			nextRunAt = &nxt
		}
	}
	e.mu.Unlock()

	_ = e.store.UpdateCronJobRunState(ctx, job.ID, startedAt, status, durationMs, output, nextRunAt)

	if e.audit != nil {
		actorType := "system"
		if triggerType == "manual" {
			actorType = "admin"
		}
		auditStatus := "success"
		if status == "failed" {
			auditStatus = "failure"
		}
		e.audit.Record(audit.Event{
			ActorType:    actorType,
			ActorName:    triggeredBy,
			Action:       "cron.run",
			ResourceType: "cron_job",
			ResourceID:   job.ID,
			Status:       auditStatus,
			Detail: map[string]any{
				"job_key":       job.JobKey,
				"trigger_type":  triggerType,
				"affected_rows": affected,
				"duration_ms":   durationMs,
				"output":        output,
			},
		})
	}

	run.Status = status
	run.AffectedRows = affected
	run.OutputLog = output
	run.FinishedAt = &finishedAt
	return run
}

func (e *Engine) dispatchJob(ctx context.Context, job *db.CronJob) (int64, string, error) {
	var cfg map[string]any
	if job.ConfigJSON != "" {
		_ = json.Unmarshal([]byte(job.ConfigJSON), &cfg)
	}

	switch job.JobKey {
	case "cleanup_audit_logs":
		defaultDays := 90
		if e.settings != nil && e.settings.Get().AuditRetentionDays > 0 {
			defaultDays = e.settings.Get().AuditRetentionDays
		}
		days := intFromMap(cfg, "retention_days", defaultDays)
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		n, err := e.store.DeleteAuditLogsBefore(ctx, cutoff)
		if err != nil {
			return 0, "", err
		}
		return n, fmt.Sprintf("deleted %d audit logs older than %d days", n, days), nil

	case "cleanup_cron_runs":
		days := intFromMap(cfg, "retention_days", 30)
		keep := intFromMap(cfg, "keep_per_job", 200)
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		n, err := e.store.DeleteCronJobRunsBefore(ctx, cutoff, keep)
		if err != nil {
			return 0, "", err
		}
		return n, fmt.Sprintf("deleted %d old cron runs (retention=%dd, keep_per_job=%d)", n, days, keep), nil

	case "storage_health_check":
		if e.storage == nil {
			return 0, "", errors.New("storage not configured")
		}
		latency, err := e.storage.HealthCheck(ctx)
		if err != nil {
			return 0, "", err
		}
		stats, statsErr := e.storage.Stats(ctx)
		if statsErr != nil {
			return 1, fmt.Sprintf("storage healthy (latency=%dms)", latency), nil
		}
		return 1, fmt.Sprintf("backend=%s healthy (latency=%dms, active_shares=%d, bytes=%d)", stats.Backend, latency, stats.ActiveShares, stats.TotalBytes), nil

	case "stats_snapshot":
		var activeShares, totalBytes int64
		if e.storage != nil {
			if st, err := e.storage.Stats(ctx); err == nil {
				activeShares = st.ActiveShares
				totalBytes = st.TotalBytes
			}
		}
		now := time.Now().UTC()
		bucket := now.Truncate(time.Hour)
		counts, err := e.store.CountClientActionsSince(ctx, bucket)
		if err != nil {
			return 0, "", err
		}
		p2pCount := int64(0)
		if e.p2p != nil {
			p2pCount = int64(e.p2p.ActiveSessionCount())
		}
		snap := &db.StatsSnapshot{
			BucketAt:         bucket,
			ActiveShares:     activeShares,
			TotalStoredBytes: totalBytes,
			PushCount:        counts.PushCount,
			PullCount:        counts.PullCount,
			DeleteCount:      counts.DeleteCount,
			P2PSessions:      p2pCount,
			CreatedAt:        now,
		}
		if err := e.store.UpsertStatsSnapshot(ctx, snap); err != nil {
			return 0, "", err
		}
		return 1, fmt.Sprintf("bucket=%s shares=%d bytes=%d push=%d pull=%d p2p=%d", bucket.Format("15:04"), activeShares, totalBytes, counts.PushCount, counts.PullCount, p2pCount), nil

	case "safekeep_backup":
		return e.runSafekeepBackup(ctx, job.ConfigJSON)

	default:
		return 0, "", fmt.Errorf("unknown job_key %q", job.JobKey)
	}
}

func intFromMap(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	v, ok := m[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		if int(n) > 0 {
			return int(n)
		}
	case int:
		if n > 0 {
			return n
		}
	}
	return fallback
}
