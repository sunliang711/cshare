export interface Admin {
  id: string
  username: string
  role: 'admin' | 'viewer'
  is_active: boolean
  must_change_password?: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface AdminSession {
  id: string
  client_ip: string
  user_agent: string
  created_at: string
  last_seen_at: string
  expires_at: string
  current: boolean
}

export interface SystemSettings {
  default_ttl: number
  max_ttl: number
  text_json_limit: number
  files_push_limit: number
  max_files_per_push: number
  ratelimit_enable: boolean
  ratelimit_rpm: number
  client_auth_enable: boolean
  has_client_jwt_secret: boolean
  audit_client_ops: boolean
  audit_retention_days: number
}

export interface AuditLog {
  id: number
  actor_type: 'admin' | 'client' | 'system'
  actor_id: string
  actor_name: string
  action: string
  resource_type: string
  resource_id: string
  share_key: string
  ip_address: string
  user_agent: string
  request_id: string
  status: 'success' | 'failure'
  detail_json: string
  created_at: string
}

export interface CronJob {
  id: string
  job_key: string
  name: string
  description: string
  cron_expr: string
  default_schedule?: string
  enabled: boolean
  timeout_sec: number
  config_json: string
  last_run_at?: string
  last_status: string
  last_duration_ms: number
  last_summary: string
  next_run_at?: string
  next_runs?: string[]
  updated_at: string
}

export interface CronJobRun {
  id: string
  job_id: string
  trigger_type: 'schedule' | 'manual'
  triggered_by: string
  status: 'running' | 'succeeded' | 'failed'
  affected_rows: number
  output_log: string
  started_at: string
  finished_at?: string
}

export interface StatsSnapshot {
  id: number
  bucket_at: string
  active_shares: number
  total_stored_bytes: number
  push_count: number
  pull_count: number
  delete_count: number
  p2p_sessions: number
}

export interface DashboardSummary {
  storage: {
    backend: string
    healthy: boolean
    latency_ms: number
    error?: string
    active_shares: number
    total_bytes: number
  }
  today: {
    push_count: number
    pull_count: number
    delete_count: number
  }
  p2p_sessions: number
  snapshots: StatsSnapshot[]
  client_auth_enabled: boolean
  ratelimit_enabled: boolean
  ratelimit_rpm: number
  cron_jobs: CronJob[]
  recent_audit_logs: AuditLog[]
}

export interface InspectSummary {
  format_version: number
  exported_at: string
  encrypted: boolean
  includes: {
    admins: boolean
    audit_logs: boolean
    cron_runs: boolean
  }
  counts: {
    system_settings: number
    cron_jobs: number
    admins: number
    audit_logs: number
    cron_job_runs: number
  }
  admin_conflicts: string[]
}

export interface SafekeepTestResult {
  reachable: boolean
  upstreams: string[]
  latency_ms: number
}
