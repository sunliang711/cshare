import type {
  Admin,
  AuditLog,
  CronJob,
  CronJobRun,
  DashboardSummary,
  InspectSummary,
  SafekeepTestResult,
  SystemSettings,
} from './types'

export class APIError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData
  const res = await fetch(`/admin/api/v1${path}`, {
    credentials: 'include',
    ...init,
    headers: {
      ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
      ...(init?.headers || {}),
    },
  })
  if (!res.ok) {
    let errBody: any = null
    try {
      errBody = await res.json()
    } catch {
      // ignore
    }
    const code = errBody?.error?.code || 'http_error'
    const message = errBody?.error?.message || `HTTP ${res.status}`
    throw new APIError(res.status, code, message)
  }
  return (await res.json()) as T
}

export function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}

export function formatTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

export const api = {
  setupStatus: () => request<{ initialized: boolean; version: string }>('/setup/status'),
  setupInit: (username: string, password: string) =>
    request<{ admin: Admin; token: string }>('/setup/init', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  login: (username: string, password: string) =>
    request<{ admin: Admin; token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<{ status: string }>('/auth/logout', { method: 'POST' }),
  me: () => request<{ admin: Admin; version: string }>('/auth/me'),
  changePassword: (old_password: string, new_password: string) =>
    request<{ status: string }>('/auth/password', {
      method: 'PUT',
      body: JSON.stringify({ old_password, new_password }),
    }),

  dashboardSummary: () => request<DashboardSummary>('/dashboard/summary'),

  getSettings: () => request<SystemSettings>('/settings'),
  updateSettings: (payload: Record<string, unknown>) =>
    request<SystemSettings>('/settings', {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),
  revealJWTSecret: () => request<{ client_jwt_secret: string }>('/settings/jwt-secret'),
  generateJWTSecret: () =>
    request<{ client_jwt_secret: string }>('/settings/generate-jwt-secret', { method: 'POST' }),

  listAdmins: () => request<{ items: Admin[] }>('/admins'),
  createAdmin: (payload: { username: string; password: string; is_active: boolean }) =>
    request<Admin>('/admins', { method: 'POST', body: JSON.stringify(payload) }),
  updateAdmin: (id: string, payload: { is_active: boolean; password?: string }) =>
    request<Admin>(`/admins/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),

  listAuditLogs: (params: Record<string, string | number | undefined>) => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') qs.set(k, String(v))
    }
    return request<{ items: AuditLog[]; total: number }>(`/audit-logs?${qs.toString()}`)
  },
  auditExportURL: (params: Record<string, string | undefined>, format: 'csv' | 'json') => {
    const qs = new URLSearchParams({ format })
    for (const [k, v] of Object.entries(params)) {
      if (v) qs.set(k, v)
    }
    return `/admin/api/v1/audit-logs/export?${qs.toString()}`
  },

  listCronJobs: () => request<{ items: CronJob[] }>('/cron-jobs'),
  previewCron: (schedule: string) =>
    request<{ next_runs: string[] }>('/cron-jobs/preview', {
      method: 'POST',
      body: JSON.stringify({ schedule }),
    }),
  revealCronSecrets: (id: string) =>
    request<{ api_key: string; passphrase: string }>(`/cron-jobs/${id}/secrets`),
  updateCronJob: (
    id: string,
    payload: { cron_expr: string; enabled: boolean; timeout_sec?: number; config?: unknown },
  ) =>
    request<CronJob>(`/cron-jobs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),
  runCronJob: (id: string) => request<CronJobRun>(`/cron-jobs/${id}/run`, { method: 'POST' }),
  listCronRuns: (id: string, limit = 30) =>
    request<{ items: CronJobRun[] }>(`/cron-jobs/${id}/runs?limit=${limit}`),
  testSafekeep: (id: string, payload: { api_base_url?: string; api_key?: string }) =>
    request<SafekeepTestResult>(`/cron-jobs/${id}/test-safekeep`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  exportMigration: async (payload: {
    include_admins: boolean
    include_audit_logs: boolean
    include_cron_runs: boolean
    passphrase: string
  }) => {
    const res = await fetch('/admin/api/v1/migration/export', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) {
      let msg = `HTTP ${res.status}`
      try {
        const j = await res.json()
        if (j?.error?.message) msg = j.error.message
      } catch {
        // ignore
      }
      throw new Error(msg)
    }
    const blob = await res.blob()
    const disp = res.headers.get('Content-Disposition') || ''
    const match = /filename="([^"]+)"/.exec(disp)
    const filename = match?.[1] || 'cshare-migration.csbundle'
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },
  inspectMigration: (fd: FormData) =>
    request<InspectSummary>('/migration/inspect', { method: 'POST', body: fd }),
  importMigration: (fd: FormData) =>
    request<{ applied: Record<string, number> }>('/migration/import', { method: 'POST', body: fd }),
}
