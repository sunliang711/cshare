package db

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("record not found")
	ErrAlreadyInitialized = errors.New("admin already initialized")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrLastAdmin          = errors.New("cannot disable or demote the last active administrator")
)

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

type Admin struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	PasswordHash       string     `json:"-"`
	Role               string     `json:"role"`
	IsActive           bool       `json:"is_active"`
	MustChangePassword bool       `json:"must_change_password"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type SystemSettings struct {
	DefaultTTL         int    `json:"default_ttl"`
	MaxTTL             int    `json:"max_ttl"`
	TextJSONLimit      int64  `json:"text_json_limit"`
	FilesPushLimit     int64  `json:"files_push_limit"`
	MaxFilesPerPush    int    `json:"max_files_per_push"`
	RateLimitEnable    bool   `json:"ratelimit_enable"`
	RateLimitRPM       int    `json:"ratelimit_rpm"`
	ClientAuthEnable   bool   `json:"client_auth_enable"`
	ClientJWTSecret    string `json:"-"`
	HasClientJWTSecret bool   `json:"has_client_jwt_secret"`
	AuditClientOps     bool   `json:"audit_client_ops"`
	AuditRetentionDays int    `json:"audit_retention_days"`
}

func DefaultSystemSettings() SystemSettings {
	return SystemSettings{
		DefaultTTL:         600,
		MaxTTL:             86400,
		TextJSONLimit:      1 << 20,  // 1 MiB
		FilesPushLimit:     20 << 20, // 20 MiB
		MaxFilesPerPush:    20,
		RateLimitEnable:    true,
		RateLimitRPM:       20,
		ClientAuthEnable:   false,
		ClientJWTSecret:    "",
		HasClientJWTSecret: false,
		AuditClientOps:     true,
		AuditRetentionDays: 90,
	}
}

type AuditLog struct {
	ID           int64     `json:"id"`
	ActorType    string    `json:"actor_type"` // admin | client | system
	ActorID      string    `json:"actor_id"`
	ActorName    string    `json:"actor_name"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	ShareKey     string    `json:"share_key"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	RequestID    string    `json:"request_id"`
	Status       string    `json:"status"` // success | failure
	DetailJSON   string    `json:"detail_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditFilter struct {
	ActorType string
	Action    string
	ShareKey  string
	IPAddress string
	Status    string
	Query     string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type CronJob struct {
	ID              string      `json:"id"`
	JobKey          string      `json:"job_key"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	CronExpr        string      `json:"cron_expr"`
	DefaultSchedule string      `json:"default_schedule,omitempty"`
	Enabled         bool        `json:"enabled"`
	TimeoutSec      int         `json:"timeout_sec"`
	ConfigJSON      string      `json:"config_json"`
	LastRunAt       *time.Time  `json:"last_run_at,omitempty"`
	LastStatus      string      `json:"last_status"`
	LastDurationMs  int64       `json:"last_duration_ms"`
	LastSummary     string      `json:"last_summary"`
	NextRunAt       *time.Time  `json:"next_run_at,omitempty"`
	NextRuns        []time.Time `json:"next_runs,omitempty"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type CronJobRun struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	TriggerType  string     `json:"trigger_type"` // schedule | manual
	TriggeredBy  string     `json:"triggered_by"`
	Status       string     `json:"status"` // running | succeeded | failed
	AffectedRows int64      `json:"affected_rows"`
	OutputLog    string     `json:"output_log"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type StatsSnapshot struct {
	ID               int64     `json:"id"`
	BucketAt         time.Time `json:"bucket_at"`
	ActiveShares     int64     `json:"active_shares"`
	TotalStoredBytes int64     `json:"total_stored_bytes"`
	PushCount        int64     `json:"push_count"`
	PullCount        int64     `json:"pull_count"`
	DeleteCount      int64     `json:"delete_count"`
	P2PSessions      int64     `json:"p2p_sessions"`
	CreatedAt        time.Time `json:"created_at"`
}

type ActionCounts struct {
	PushCount   int64 `json:"push_count"`
	PullCount   int64 `json:"pull_count"`
	DeleteCount int64 `json:"delete_count"`
}
