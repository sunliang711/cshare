package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"crossshare-server/internal/audit"
	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/middleware"
	"crossshare-server/internal/migration"
	"crossshare-server/internal/scheduler"
	"crossshare-server/internal/storage"
)

// --- Dashboard ---

func (h *AdminHandler) DashboardSummary(c *gin.Context) {
	ctx := c.Request.Context()

	var st storage.StorageStats
	var latencyMs int64
	healthy := true
	storageErr := ""
	if h.storage != nil {
		lat, hErr := h.storage.HealthCheck(ctx)
		latencyMs = lat
		if hErr != nil {
			healthy = false
			storageErr = hErr.Error()
		}
		if stats, sErr := h.storage.Stats(ctx); sErr == nil {
			st = stats
		}
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC()
	todayCounts, _ := h.store.CountClientActionsSince(ctx, startOfDay)

	p2pCount := 0
	if h.p2p != nil {
		p2pCount = h.p2p.ActiveSessionCount()
	}

	snapshots, _ := h.store.ListStatsSnapshots(ctx, 24)
	if snapshots == nil {
		snapshots = []db.StatsSnapshot{}
	}

	jobs, _ := h.store.ListCronJobs(ctx)
	for i := range jobs {
		jobs[i] = sanitizeCronJobDTO(jobs[i])
	}

	recentLogs, _, _ := h.store.ListAuditLogs(ctx, db.AuditFilter{Limit: 8})
	cur := h.settings.Get()

	c.JSON(http.StatusOK, gin.H{
		"storage": gin.H{
			"backend":       st.Backend,
			"healthy":       healthy,
			"latency_ms":    latencyMs,
			"error":         storageErr,
			"active_shares": st.ActiveShares,
			"total_bytes":   st.TotalBytes,
		},
		"today":               todayCounts,
		"p2p_sessions":        p2pCount,
		"snapshots":           snapshots,
		"client_auth_enabled": cur.ClientAuthEnable,
		"ratelimit_enabled":   cur.RateLimitEnable,
		"ratelimit_rpm":       cur.RateLimitRPM,
		"cron_jobs":           jobs,
		"recent_audit_logs":   recentLogs,
	})
}

// --- Settings ---

func (h *AdminHandler) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, h.settings.Get())
}

func (h *AdminHandler) RevealClientJWTSecret(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	cur := h.settings.Get()
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "settings.reveal_jwt_secret",
		ResourceType: "system_settings",
	})
	c.JSON(http.StatusOK, gin.H{"client_jwt_secret": cur.ClientJWTSecret})
}

func (h *AdminHandler) GenerateClientJWTSecret(c *gin.Context) {
	secret, err := crypto.RandomToken(32)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "generate_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"client_jwt_secret": secret})
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	var req struct {
		DefaultTTL           int     `json:"default_ttl"`
		MaxTTL               int     `json:"max_ttl"`
		TextJSONLimit        int64   `json:"text_json_limit"`
		FilesPushLimit       int64   `json:"files_push_limit"`
		MaxFilesPerPush      int     `json:"max_files_per_push"`
		RateLimitEnable      bool    `json:"ratelimit_enable"`
		RateLimitRPM         int     `json:"ratelimit_rpm"`
		ClientAuthEnable     bool    `json:"client_auth_enable"`
		ClientJWTSecret      *string `json:"client_jwt_secret"`
		ClearClientJWTSecret bool    `json:"clear_client_jwt_secret"`
		AuditClientOps       bool    `json:"audit_client_ops"`
		AuditRetentionDays   int     `json:"audit_retention_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	existing := h.settings.Get()
	next := db.SystemSettings{
		DefaultTTL:         req.DefaultTTL,
		MaxTTL:             req.MaxTTL,
		TextJSONLimit:      req.TextJSONLimit,
		FilesPushLimit:     req.FilesPushLimit,
		MaxFilesPerPush:    req.MaxFilesPerPush,
		RateLimitEnable:    req.RateLimitEnable,
		RateLimitRPM:       req.RateLimitRPM,
		ClientAuthEnable:   req.ClientAuthEnable,
		ClientJWTSecret:    existing.ClientJWTSecret,
		AuditClientOps:     req.AuditClientOps,
		AuditRetentionDays: req.AuditRetentionDays,
	}

	if req.ClearClientJWTSecret {
		next.ClientJWTSecret = ""
	} else if req.ClientJWTSecret != nil {
		val := strings.TrimSpace(*req.ClientJWTSecret)
		if val != "" && val != "*****" && val != "••••••" {
			next.ClientJWTSecret = val
		} else if val == "" {
			next.ClientJWTSecret = ""
		}
	}

	if err := h.settings.Update(c.Request.Context(), next, admin.Username); err != nil {
		writeAdminError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Keep cleanup_audit_logs retention_days in sync if user didn't customize differently, or audit change
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "settings.update",
		ResourceType: "system_settings",
		Detail: gin.H{
			"default_ttl":           next.DefaultTTL,
			"max_ttl":               next.MaxTTL,
			"text_json_limit":       next.TextJSONLimit,
			"files_push_limit":      next.FilesPushLimit,
			"max_files_per_push":    next.MaxFilesPerPush,
			"ratelimit_enable":      next.RateLimitEnable,
			"ratelimit_rpm":         next.RateLimitRPM,
			"client_auth_enable":    next.ClientAuthEnable,
			"has_client_jwt_secret": next.ClientJWTSecret != "",
			"audit_client_ops":      next.AuditClientOps,
			"audit_retention_days":  next.AuditRetentionDays,
		},
	})
	c.JSON(http.StatusOK, h.settings.Get())
}

// --- Audit Logs ---

func parseAuditFilter(c *gin.Context) db.AuditFilter {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	f := db.AuditFilter{
		ActorType: strings.TrimSpace(c.Query("actor_type")),
		Action:    strings.TrimSpace(c.Query("action")),
		ShareKey:  strings.TrimSpace(c.Query("share_key")),
		IPAddress: strings.TrimSpace(c.Query("ip")),
		Status:    strings.TrimSpace(c.Query("status")),
		Query:     strings.TrimSpace(c.Query("q")),
		Limit:     limit,
		Offset:    offset,
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			f.From = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			f.To = &t
		}
	}
	return f
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	f := parseAuditFilter(c)
	logs, total, err := h.store.ListAuditLogs(c.Request.Context(), f)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": logs, "total": total})
}

func (h *AdminHandler) ExportAuditLogs(c *gin.Context) {
	f := parseAuditFilter(c)
	logs, err := h.store.ExportAllAuditLogs(c.Request.Context(), f)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}
	format := strings.ToLower(c.Query("format"))
	if format == "json" {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="crossshare-audit-logs.json"`)
		_ = json.NewEncoder(c.Writer).Encode(logs)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="crossshare-audit-logs.csv"`)
	cw := csv.NewWriter(c.Writer)
	_ = cw.Write([]string{"id", "created_at", "actor_type", "actor_name", "action", "resource_type", "resource_id", "share_key", "status", "ip_address", "request_id", "detail_json"})
	for _, l := range logs {
		_ = cw.Write([]string{
			strconv.FormatInt(l.ID, 10),
			l.CreatedAt.Format(time.RFC3339),
			l.ActorType,
			l.ActorName,
			l.Action,
			l.ResourceType,
			l.ResourceID,
			l.ShareKey,
			l.Status,
			l.IPAddress,
			l.RequestID,
			l.DetailJSON,
		})
	}
	cw.Flush()
}

// --- Cron Jobs ---

func sanitizeCronJobDTO(j db.CronJob) db.CronJob {
	j.DefaultSchedule = scheduler.DefaultScheduleFor(j.JobKey)
	if j.TimeoutSec <= 0 {
		j.TimeoutSec = scheduler.DefaultTimeoutFor(j.JobKey)
	}
	if next, err := scheduler.PreviewCron(j.CronExpr, 5); err == nil {
		j.NextRuns = next
	} else {
		j.NextRuns = []time.Time{}
	}
	if j.JobKey == "safekeep_backup" {
		j.ConfigJSON = scheduler.SanitizeSafekeepBackupConfigJSON(j.ConfigJSON)
	}
	return j
}

func (h *AdminHandler) ListCronJobs(c *gin.Context) {
	jobs, err := h.store.ListCronJobs(c.Request.Context())
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	for i := range jobs {
		jobs[i] = sanitizeCronJobDTO(jobs[i])
	}
	c.JSON(http.StatusOK, gin.H{"items": jobs})
}

func (h *AdminHandler) PreviewCronSchedule(c *gin.Context) {
	var req struct {
		Schedule string `json:"schedule"`
		CronExpr string `json:"cron_expr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	expr := strings.TrimSpace(req.Schedule)
	if expr == "" {
		expr = strings.TrimSpace(req.CronExpr)
	}
	times, err := scheduler.PreviewCron(expr, 5)
	if err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_cron", fmt.Sprintf("invalid cron expression: %v", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"next_runs": times})
}

func (h *AdminHandler) RevealCronJobSecrets(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	id := c.Param("id")
	job, err := h.store.GetCronJobByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeAdminError(c, http.StatusNotFound, "not_found", "cron job not found")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	var apiKey, passphrase string
	if job.JobKey == "safekeep_backup" {
		apiKey, passphrase, err = scheduler.RevealSafekeepBackupSecrets(h.cipher, job.ConfigJSON)
		if err != nil {
			writeAdminError(c, http.StatusInternalServerError, "decrypt_failed", err.Error())
			return
		}
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "cron.reveal_secrets",
		ResourceType: "cron_job",
		ResourceID:   job.ID,
		Detail:       gin.H{"job_key": job.JobKey},
	})
	c.JSON(http.StatusOK, gin.H{
		"api_key":    apiKey,
		"passphrase": passphrase,
	})
}

func (h *AdminHandler) UpdateCronJob(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	id := c.Param("id")
	existing, err := h.store.GetCronJobByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeAdminError(c, http.StatusNotFound, "not_found", "cron job not found")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	var req struct {
		CronExpr   string          `json:"cron_expr"`
		Enabled    bool            `json:"enabled"`
		TimeoutSec *int            `json:"timeout_sec"`
		Config     json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := scheduler.ValidateCron(req.CronExpr); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_cron", fmt.Sprintf("invalid cron expression: %v", err))
		return
	}
	timeoutSec := existing.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = scheduler.DefaultTimeoutFor(existing.JobKey)
	}
	if req.TimeoutSec != nil {
		if *req.TimeoutSec < 1 || *req.TimeoutSec > 86400 {
			writeAdminError(c, http.StatusBadRequest, "invalid_timeout", "timeout_sec must be between 1 and 86400")
			return
		}
		timeoutSec = *req.TimeoutSec
	}

	rawCfg := strings.TrimSpace(string(req.Config))
	cfgStr := existing.ConfigJSON
	if rawCfg != "" && rawCfg != "null" {
		if existing.JobKey == "safekeep_backup" {
			cfgStr, err = scheduler.MergeSafekeepBackupConfig(h.cipher, existing.ConfigJSON, req.Config)
		} else {
			cfgStr, err = scheduler.NormalizeJobConfig(existing.JobKey, req.Config)
		}
		if err != nil {
			writeAdminError(c, http.StatusBadRequest, "invalid_config", err.Error())
			return
		}
	}

	if err := h.store.UpdateCronJob(c.Request.Context(), id, strings.TrimSpace(req.CronExpr), req.Enabled, timeoutSec, cfgStr); err != nil {
		writeAdminError(c, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	if h.sched != nil {
		_ = h.sched.ReloadAll(c.Request.Context())
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "cron.update",
		ResourceType: "cron_job",
		ResourceID:   id,
		Detail: gin.H{
			"job_key":     existing.JobKey,
			"cron_expr":   strings.TrimSpace(req.CronExpr),
			"enabled":     req.Enabled,
			"timeout_sec": timeoutSec,
		},
	})

	updated, _ := h.store.GetCronJobByID(c.Request.Context(), id)
	if updated == nil {
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
		return
	}
	c.JSON(http.StatusOK, sanitizeCronJobDTO(*updated))
}

func (h *AdminHandler) RunCronJob(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	id := c.Param("id")
	if h.sched == nil {
		writeAdminError(c, http.StatusServiceUnavailable, "no_scheduler", "scheduler not running")
		return
	}
	run, err := h.sched.RunNow(c.Request.Context(), id, admin.Username)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeAdminError(c, http.StatusNotFound, "not_found", "cron job not found")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "run_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *AdminHandler) ListCronJobRuns(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	runs, err := h.store.ListCronJobRuns(c.Request.Context(), id, limit)
	if err != nil {
		writeAdminError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": runs})
}

func (h *AdminHandler) TestSafekeepConnection(c *gin.Context) {
	id := c.Param("id")
	job, err := h.store.GetCronJobByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeAdminError(c, http.StatusNotFound, "not_found", "cron job not found")
			return
		}
		writeAdminError(c, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	var req struct {
		APIBaseURL string `json:"api_base_url"`
		APIKey     string `json:"api_key"`
	}
	_ = c.ShouldBindJSON(&req)
	res, err := h.sched.TestSafekeepBackup(c.Request.Context(), job.ConfigJSON, req.APIBaseURL, req.APIKey)
	if err != nil {
		writeAdminError(c, http.StatusBadGateway, "safekeep_test_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, res)
}

// --- Migration ---

func (h *AdminHandler) MigrationExport(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	var req migration.ExportOptions
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	blob, err := h.mig.Export(c.Request.Context(), req)
	if err != nil {
		writeAdminError(c, http.StatusBadRequest, "export_failed", err.Error())
		return
	}
	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "migration.export",
		ResourceType: "migration",
		Detail: gin.H{
			"encrypted":          req.Passphrase != "",
			"include_admins":     req.IncludeAdmins,
			"include_audit_logs": req.IncludeAuditLogs,
			"include_cron_runs":  req.IncludeCronRuns,
			"size_bytes":         len(blob),
		},
	})
	filename := fmt.Sprintf("cshare-migration-%s.csbundle", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/octet-stream", blob)
}

func readUploadedBundle(c *gin.Context) ([]byte, error) {
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, errors.New("bundle file is required")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 64<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("uploaded bundle is empty")
	}
	return data, nil
}

func (h *AdminHandler) MigrationInspect(c *gin.Context) {
	blob, err := readUploadedBundle(c)
	if err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_bundle", err.Error())
		return
	}
	passphrase := c.PostForm("passphrase")
	summary, err := h.mig.Inspect(c.Request.Context(), blob, passphrase)
	if err != nil {
		if errors.Is(err, crypto.ErrPassphraseRequired) {
			writeAdminError(c, http.StatusBadRequest, "passphrase_required", err.Error())
			return
		}
		writeAdminError(c, http.StatusBadRequest, "inspect_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *AdminHandler) MigrationImport(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	blob, err := readUploadedBundle(c)
	if err != nil {
		writeAdminError(c, http.StatusBadRequest, "invalid_bundle", err.Error())
		return
	}

	opts := migration.ImportOptions{
		Passphrase:       c.PostForm("passphrase"),
		Mode:             c.PostForm("mode"),
		ConflictStrategy: c.PostForm("conflict_strategy"),
		IncludeAdmins:    c.PostForm("include_admins") == "true",
		IncludeAuditLogs: c.PostForm("include_audit_logs") == "true",
		IncludeCronRuns:  c.PostForm("include_cron_runs") == "true",
	}

	counts, err := h.mig.Import(c.Request.Context(), blob, opts)
	if err != nil {
		if errors.Is(err, crypto.ErrPassphraseRequired) {
			writeAdminError(c, http.StatusBadRequest, "passphrase_required", err.Error())
			return
		}
		writeAdminError(c, http.StatusBadRequest, "import_failed", err.Error())
		return
	}

	_ = h.settings.Reload(c.Request.Context())
	if h.sched != nil {
		_ = h.sched.ReloadAll(c.Request.Context())
	}

	h.audit.RecordFromGin(c, audit.Event{
		ActorType:    "admin",
		ActorID:      admin.ID,
		ActorName:    admin.Username,
		Action:       "migration.import",
		ResourceType: "migration",
		Detail: gin.H{
			"mode":              opts.Mode,
			"conflict_strategy": opts.ConflictStrategy,
			"counts":            counts,
		},
	})
	c.JSON(http.StatusOK, gin.H{"applied": counts})
}
