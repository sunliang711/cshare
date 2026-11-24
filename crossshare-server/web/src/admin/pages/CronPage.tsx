import React, { useCallback, useEffect, useState } from 'react'
import { History, Loader2, Play, PlugZap, RefreshCw, Sliders } from 'lucide-react'
import { api, formatTime } from '../api'
import { Badge, StatusBadge } from '../components/Badge'
import { Drawer } from '../components/Drawer'
import { LoadingState } from '../components/Feedback'
import { SecretField } from '../components/SecretField'
import { useToast } from '../components/Toast'
import { useI18n } from '../contexts/I18nContext'
import type { TranslationKey } from '../i18n/zh'
import type { CronJob, CronJobRun } from '../types'
import { describeCron } from '../utils/cron'

const JOB_I18N: Record<string, { name: TranslationKey; desc: TranslationKey }> = {
  cleanup_audit_logs: { name: 'cron.job.cleanup_audit_logs.name', desc: 'cron.job.cleanup_audit_logs.desc' },
  cleanup_cron_runs: { name: 'cron.job.cleanup_cron_runs.name', desc: 'cron.job.cleanup_cron_runs.desc' },
  storage_health_check: { name: 'cron.job.storage_health_check.name', desc: 'cron.job.storage_health_check.desc' },
  stats_snapshot: { name: 'cron.job.stats_snapshot.name', desc: 'cron.job.stats_snapshot.desc' },
  safekeep_backup: { name: 'cron.job.safekeep_backup.name', desc: 'cron.job.safekeep_backup.desc' },
}

const PRESETS: Array<{ labelKey: TranslationKey; value: string }> = [
  { labelKey: 'cron.preset.every5m', value: '*/5 * * * *' },
  { labelKey: 'cron.preset.hourly', value: '0 * * * *' },
  { labelKey: 'cron.preset.daily02', value: '0 2 * * *' },
  { labelKey: 'cron.preset.daily03', value: '0 3 * * *' },
  { labelKey: 'cron.preset.weeklySun03', value: '0 3 * * 0' },
  { labelKey: 'cron.preset.weeklyMon05', value: '0 5 * * 1' },
  { labelKey: 'cron.preset.monthly', value: '0 4 1 * *' },
  { labelKey: 'cron.preset.custom', value: 'custom' },
]

interface SafekeepForm {
  api_base_url: string
  api_key: string
  has_api_key: boolean
  clear_api_key: boolean
  targets: string
  path: string
  mode: 'sync' | 'async'
  rename: string
  passphrase: string
  has_passphrase: boolean
  clear_passphrase: boolean
  include_admins: boolean
  include_audit_logs: boolean
  include_cron_runs: boolean
}

function safeParse(raw?: string): Record<string, any> {
  try {
    return JSON.parse(raw || '{}') || {}
  } catch {
    return {}
  }
}

export const CronPage: React.FC = () => {
  const { t, lang } = useI18n()
  const toast = useToast()

  const [jobs, setJobs] = useState<CronJob[]>([])
  const [loading, setLoading] = useState(true)
  const [runningId, setRunningId] = useState<string | null>(null)

  // Edit drawer
  const [editingJob, setEditingJob] = useState<CronJob | null>(null)
  const [cronExpr, setCronExpr] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [timeoutSec, setTimeoutSec] = useState(120)
  const [previewRuns, setPreviewRuns] = useState<string[]>([])
  const [previewErr, setPreviewErr] = useState('')
  const [saving, setSaving] = useState(false)

  // Job-specific configs
  const [retentionDays, setRetentionDays] = useState(90)
  const [keepPerJob, setKeepPerJob] = useState(200)
  const [skForm, setSkForm] = useState<SafekeepForm>({
    api_base_url: '',
    api_key: '',
    has_api_key: false,
    clear_api_key: false,
    targets: '',
    path: 'cshare-backups',
    mode: 'sync',
    rename: '',
    passphrase: '',
    has_passphrase: false,
    clear_passphrase: false,
    include_admins: true,
    include_audit_logs: false,
    include_cron_runs: false,
  })
  const [testingSK, setTestingSK] = useState(false)

  // Runs history drawer
  const [historyJob, setHistoryJob] = useState<CronJob | null>(null)
  const [runs, setRuns] = useState<CronJobRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.listCronJobs()
      setJobs(res.items || [])
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setLoading(false)
    }
  }, [toast])

  useEffect(() => {
    load()
  }, [load])

  const refreshPreview = useCallback(async (expr: string) => {
    if (!expr.trim()) {
      setPreviewRuns([])
      setPreviewErr('')
      return
    }
    try {
      const res = await api.previewCron(expr)
      setPreviewRuns(res.next_runs || [])
      setPreviewErr('')
    } catch (e: any) {
      setPreviewRuns([])
      setPreviewErr(e?.message || 'invalid cron')
    }
  }, [])

  useEffect(() => {
    if (!editingJob) return
    const timer = setTimeout(() => {
      refreshPreview(cronExpr)
    }, 250)
    return () => clearTimeout(timer)
  }, [cronExpr, editingJob, refreshPreview])

  const openEdit = (job: CronJob) => {
    setEditingJob(job)
    setCronExpr(job.cron_expr)
    setEnabled(job.enabled)
    setTimeoutSec(job.timeout_sec || 120)
    setPreviewRuns(job.next_runs || [])
    setPreviewErr('')

    const cfg = safeParse(job.config_json)
    if (job.job_key === 'cleanup_audit_logs') {
      setRetentionDays(Number(cfg.retention_days) || 90)
    } else if (job.job_key === 'cleanup_cron_runs') {
      setRetentionDays(Number(cfg.retention_days) || 30)
      setKeepPerJob(Number(cfg.keep_per_job) || 200)
    } else if (job.job_key === 'safekeep_backup') {
      setSkForm({
        api_base_url: String(cfg.api_base_url || ''),
        api_key: '',
        has_api_key: Boolean(cfg.has_api_key),
        clear_api_key: false,
        targets: String(cfg.targets || ''),
        path: String(cfg.path || 'cshare-backups'),
        mode: cfg.mode === 'async' ? 'async' : 'sync',
        rename: String(cfg.rename || ''),
        passphrase: '',
        has_passphrase: Boolean(cfg.has_passphrase),
        clear_passphrase: false,
        include_admins: cfg.include_admins !== false,
        include_audit_logs: Boolean(cfg.include_audit_logs),
        include_cron_runs: Boolean(cfg.include_cron_runs),
      })
    }
  }

  const openHistory = async (job: CronJob) => {
    setHistoryJob(job)
    setRunsLoading(true)
    try {
      const res = await api.listCronRuns(job.id, 30)
      setRuns(res.items || [])
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setRunsLoading(false)
    }
  }

  const handleToggleEnabled = async (job: CronJob) => {
    try {
      await api.updateCronJob(job.id, {
        cron_expr: job.cron_expr,
        enabled: !job.enabled,
        timeout_sec: job.timeout_sec,
      })
      toast.success(t('cron.saved'))
      load()
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const handleRunNow = async (job: CronJob) => {
    setRunningId(job.id)
    try {
      const run = await api.runCronJob(job.id)
      if (run.status === 'succeeded') {
        toast.success(t('cron.runTriggered', { status: run.output_log || run.status }))
      } else {
        toast.error(run.output_log || t('cron.runFailed'))
      }
      load()
    } catch (e: any) {
      toast.error(e.message || t('cron.runFailed'))
    } finally {
      setRunningId(null)
    }
  }

  const handleTestSafekeep = async () => {
    if (!editingJob) return
    setTestingSK(true)
    try {
      const res = await api.testSafekeep(editingJob.id, {
        api_base_url: skForm.api_base_url,
        api_key: skForm.api_key || undefined,
      })
      if (res.upstreams && res.upstreams.length > 0) {
        toast.success(
          t('cron.skTestSuccess', { upstreams: res.upstreams.join(', '), ms: res.latency_ms }),
        )
      } else {
        toast.success(t('cron.skTestNoUpstream', { ms: res.latency_ms }))
      }
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setTestingSK(false)
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingJob) return
    setSaving(true)
    try {
      let config: unknown = undefined
      if (editingJob.job_key === 'cleanup_audit_logs') {
        config = { retention_days: Number(retentionDays) || 90 }
      } else if (editingJob.job_key === 'cleanup_cron_runs') {
        config = {
          retention_days: Number(retentionDays) || 30,
          keep_per_job: Number(keepPerJob) || 200,
        }
      } else if (editingJob.job_key === 'safekeep_backup') {
        const cfgPayload: Record<string, unknown> = {
          api_base_url: skForm.api_base_url,
          targets: skForm.targets,
          path: skForm.path,
          mode: skForm.mode,
          rename: skForm.rename,
          include_admins: skForm.include_admins,
          include_audit_logs: skForm.include_audit_logs,
          include_cron_runs: skForm.include_cron_runs,
        }
        if (skForm.clear_api_key) {
          cfgPayload.api_key = ''
        } else if (skForm.api_key) {
          cfgPayload.api_key = skForm.api_key
        }
        if (skForm.clear_passphrase) {
          cfgPayload.passphrase = ''
        } else if (skForm.passphrase) {
          cfgPayload.passphrase = skForm.passphrase
        }
        config = cfgPayload
      }

      await api.updateCronJob(editingJob.id, {
        cron_expr: cronExpr,
        enabled,
        timeout_sec: Number(timeoutSec) || 120,
        config,
      })
      toast.success(t('cron.saved'))
      setEditingJob(null)
      load()
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setSaving(false)
    }
  }

  const selectedPreset = PRESETS.some((p) => p.value === cronExpr) ? cronExpr : 'custom'

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('cron.title')}</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('cron.subtitle')}</p>
        </div>
        <button type="button" onClick={load} className="secondary-button">
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          <span>{t('common.refresh')}</span>
        </button>
      </div>

      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 overflow-hidden">
        {loading && jobs.length === 0 ? (
          <LoadingState label={t('common.loading')} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">{t('cron.colJob')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('cron.colSchedule')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('cron.colStatus')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('cron.colLastRun')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('cron.colNextRun')}</th>
                  <th className="px-4 py-3 text-right font-medium">{t('cron.colActions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {jobs.map((job) => {
                  const labels = JOB_I18N[job.job_key]
                  const name = labels ? t(labels.name) : job.name
                  const desc = labels ? t(labels.desc) : job.description
                  const humanCron = describeCron(job.cron_expr, lang)
                  return (
                    <tr key={job.id} className="hover:bg-slate-50/70 dark:hover:bg-slate-800/40">
                      <td className="px-4 py-3.5 min-w-[180px] max-w-[260px]">
                        <div className="font-semibold text-slate-900 dark:text-slate-100">{name}</div>
                        <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">{desc}</div>
                        {job.last_summary && (
                          <div className="text-[11px] text-slate-400 font-mono mt-1 truncate max-w-xs" title={job.last_summary}>
                            {job.last_summary}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3.5 whitespace-nowrap">
                        <div className="font-mono text-slate-800 dark:text-slate-200">{job.cron_expr}</div>
                        {humanCron && <div className="text-[11px] text-slate-400 mt-0.5">{humanCron}</div>}
                      </td>
                      <td className="px-4 py-3.5 whitespace-nowrap">
                        <button
                          type="button"
                          role="switch"
                          aria-checked={job.enabled}
                          onClick={() => handleToggleEnabled(job)}
                          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium ${
                            job.enabled
                              ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                              : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
                          }`}
                        >
                          <span className={`w-1.5 h-1.5 rounded-full ${job.enabled ? 'bg-emerald-500' : 'bg-slate-400'}`} />
                          {job.enabled ? t('common.enabled') : t('common.disabled')}
                        </button>
                      </td>
                      <td className="px-4 py-3.5 whitespace-nowrap">
                        {job.last_run_at ? (
                          <div className="space-y-1">
                            <div className="flex items-center gap-1.5">
                              <StatusBadge status={job.last_status} />
                              <span className="text-slate-400 font-mono">{job.last_duration_ms}ms</span>
                            </div>
                            <div className="text-[11px] text-slate-400">{formatTime(job.last_run_at)}</div>
                          </div>
                        ) : (
                          <span className="text-slate-400">{t('cron.neverRun')}</span>
                        )}
                      </td>
                      <td className="px-4 py-3.5 whitespace-nowrap text-slate-500">
                        {job.enabled && job.next_run_at ? formatTime(job.next_run_at) : '—'}
                      </td>
                      <td className="px-4 py-3.5 text-right">
                        <div className="flex flex-wrap items-center justify-end gap-1.5">
                          <button
                            type="button"
                            disabled={runningId === job.id}
                            onClick={() => handleRunNow(job)}
                            className="secondary-button py-1.5 px-2.5 text-xs"
                          >
                            {runningId === job.id ? (
                              <Loader2 className="w-3.5 h-3.5 animate-spin" />
                            ) : (
                              <Play className="w-3.5 h-3.5" />
                            )}
                            <span>{t('common.runNow')}</span>
                          </button>
                          <button
                            type="button"
                            onClick={() => openEdit(job)}
                            className="secondary-button py-1.5 px-2.5 text-xs"
                          >
                            <Sliders className="w-3.5 h-3.5" />
                            <span>{t('common.edit')}</span>
                          </button>
                          <button
                            type="button"
                            onClick={() => openHistory(job)}
                            className="secondary-button py-1.5 px-2.5 text-xs"
                          >
                            <History className="w-3.5 h-3.5" />
                            <span>{t('common.history')}</span>
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Edit Cron Drawer */}
      <Drawer
        open={Boolean(editingJob)}
        onClose={() => setEditingJob(null)}
        title={
          editingJob
            ? `${t('cron.editTitle')} — ${
                JOB_I18N[editingJob.job_key] ? t(JOB_I18N[editingJob.job_key].name) : editingJob.name
              }`
            : t('cron.editTitle')
        }
        widthClass="max-w-2xl"
      >
        {editingJob && (
          <form onSubmit={handleSave} className="space-y-5 text-xs">
            <div className="flex items-center justify-between p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/60">
              <label className="flex items-center gap-2 font-medium text-slate-800 dark:text-slate-200 cursor-pointer">
                <input
                  type="checkbox"
                  checked={enabled}
                  onChange={(e) => setEnabled(e.target.checked)}
                />
                <span>{t('common.enable')}</span>
              </label>
              <Badge tone="indigo">{editingJob.job_key}</Badge>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('cron.presetLabel')}
                </label>
                <select
                  value={selectedPreset}
                  onChange={(e) => {
                    if (e.target.value !== 'custom') setCronExpr(e.target.value)
                  }}
                  className="w-full px-3 py-2 border rounded-lg text-xs"
                >
                  {PRESETS.map((p) => (
                    <option key={p.value} value={p.value}>
                      {t(p.labelKey)}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('cron.timeoutSec')}
                </label>
                <input
                  type="number"
                  min={1}
                  max={86400}
                  value={timeoutSec}
                  onChange={(e) => setTimeoutSec(Number(e.target.value))}
                  className="w-full px-3 py-2 border rounded-lg text-xs"
                />
              </div>
            </div>

            <div>
              <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('cron.cronExpr')}
              </label>
              <input
                type="text"
                required
                value={cronExpr}
                onChange={(e) => setCronExpr(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg font-mono text-xs"
              />
              {previewErr ? (
                <p className="text-rose-600 dark:text-rose-400 mt-1">{previewErr}</p>
              ) : (
                previewRuns.length > 0 && (
                  <div className="mt-2 p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 space-y-1">
                    <div className="text-[11px] font-medium text-slate-500">{t('cron.nextRunsPreview')}</div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-1 font-mono text-[11px] text-slate-600 dark:text-slate-300">
                      {previewRuns.map((iso) => (
                        <div key={iso}>{formatTime(iso)}</div>
                      ))}
                    </div>
                  </div>
                )
              )}
            </div>

            {editingJob.job_key === 'cleanup_audit_logs' && (
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('cron.retentionDays')}
                </label>
                <input
                  type="number"
                  min={1}
                  max={36500}
                  value={retentionDays}
                  onChange={(e) => setRetentionDays(Number(e.target.value))}
                  className="w-full px-3 py-2 border rounded-lg text-xs"
                />
              </div>
            )}

            {editingJob.job_key === 'cleanup_cron_runs' && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                    {t('cron.retentionDays')}
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={36500}
                    value={retentionDays}
                    onChange={(e) => setRetentionDays(Number(e.target.value))}
                    className="w-full px-3 py-2 border rounded-lg text-xs"
                  />
                </div>
                <div>
                  <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                    {t('cron.keepPerJob')}
                  </label>
                  <input
                    type="number"
                    min={1}
                    max={10000}
                    value={keepPerJob}
                    onChange={(e) => setKeepPerJob(Number(e.target.value))}
                    className="w-full px-3 py-2 border rounded-lg text-xs"
                  />
                </div>
              </div>
            )}

            {editingJob.job_key === 'safekeep_backup' && (
              <div className="space-y-4 border-t border-slate-200 dark:border-slate-800 pt-4">
                <div>
                  <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                    {t('cron.skApiUrl')}
                  </label>
                  <input
                    type="url"
                    placeholder="https://safekeep.example.com"
                    value={skForm.api_base_url}
                    onChange={(e) => setSkForm((f) => ({ ...f, api_base_url: e.target.value }))}
                    className="w-full px-3 py-2 border rounded-lg font-mono text-xs"
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="font-medium text-slate-700 dark:text-slate-300">{t('cron.skApiKey')}</label>
                    <button
                      type="button"
                      disabled={testingSK}
                      onClick={handleTestSafekeep}
                      className="inline-flex items-center gap-1 text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
                    >
                      {testingSK ? <Loader2 className="w-3 h-3 animate-spin" /> : <PlugZap className="w-3 h-3" />}
                      <span>{t('cron.skTestConn')}</span>
                    </button>
                  </div>
                  <SecretField
                    value={skForm.api_key}
                    hasSaved={skForm.has_api_key && !skForm.clear_api_key}
                    onChange={(v) => setSkForm((f) => ({ ...f, api_key: v, clear_api_key: false }))}
                    onReveal={async () => {
                      const sec = await api.revealCronSecrets(editingJob.id)
                      return sec.api_key
                    }}
                  />
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      {t('cron.skPath')}
                    </label>
                    <input
                      type="text"
                      value={skForm.path}
                      onChange={(e) => setSkForm((f) => ({ ...f, path: e.target.value }))}
                      className="w-full px-3 py-2 border rounded-lg font-mono text-xs"
                    />
                  </div>
                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      {t('cron.skMode')}
                    </label>
                    <select
                      value={skForm.mode}
                      onChange={(e) => setSkForm((f) => ({ ...f, mode: e.target.value === 'async' ? 'async' : 'sync' }))}
                      className="w-full px-3 py-2 border rounded-lg text-xs"
                    >
                      <option value="sync">sync</option>
                      <option value="async">async</option>
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      {t('cron.skTargets')}
                    </label>
                    <input
                      type="text"
                      placeholder="prod-s3,dropbox"
                      value={skForm.targets}
                      onChange={(e) => setSkForm((f) => ({ ...f, targets: e.target.value }))}
                      className="w-full px-3 py-2 border rounded-lg font-mono text-xs"
                    />
                  </div>
                  <div>
                    <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                      {t('cron.skRename')}
                    </label>
                    <input
                      type="text"
                      placeholder="cshare-latest.csbundle"
                      value={skForm.rename}
                      onChange={(e) => setSkForm((f) => ({ ...f, rename: e.target.value }))}
                      className="w-full px-3 py-2 border rounded-lg font-mono text-xs"
                    />
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="font-medium text-slate-700 dark:text-slate-300">{t('cron.skPassphrase')}</label>
                    {skForm.has_passphrase && !skForm.clear_passphrase && (
                      <button
                        type="button"
                        onClick={() => setSkForm((f) => ({ ...f, passphrase: '', clear_passphrase: true }))}
                        className="text-[11px] text-rose-600 hover:underline"
                      >
                        {t('settings.clearSecret')}
                      </button>
                    )}
                  </div>
                  <SecretField
                    value={skForm.passphrase}
                    hasSaved={skForm.has_passphrase && !skForm.clear_passphrase}
                    onChange={(v) => setSkForm((f) => ({ ...f, passphrase: v, clear_passphrase: false }))}
                    onReveal={async () => {
                      const sec = await api.revealCronSecrets(editingJob.id)
                      return sec.passphrase
                    }}
                  />
                </div>

                <div className="space-y-2 pt-1">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={skForm.include_admins}
                      onChange={(e) => setSkForm((f) => ({ ...f, include_admins: e.target.checked }))}
                    />
                    <span>{t('cron.skIncludeAdmins')}</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={skForm.include_audit_logs}
                      onChange={(e) => setSkForm((f) => ({ ...f, include_audit_logs: e.target.checked }))}
                    />
                    <span>{t('cron.skIncludeAudit')}</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={skForm.include_cron_runs}
                      onChange={(e) => setSkForm((f) => ({ ...f, include_cron_runs: e.target.checked }))}
                    />
                    <span>{t('cron.skIncludeCronRuns')}</span>
                  </label>
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-4 border-t border-slate-200 dark:border-slate-800">
              <button type="button" onClick={() => setEditingJob(null)} className="secondary-button">
                {t('common.cancel')}
              </button>
              <button type="submit" disabled={saving || Boolean(previewErr)} className="primary-button">
                {saving ? t('common.saving') : t('common.save')}
              </button>
            </div>
          </form>
        )}
      </Drawer>

      {/* Runs History Drawer */}
      <Drawer
        open={Boolean(historyJob)}
        onClose={() => setHistoryJob(null)}
        title={
          historyJob
            ? `${t('cron.runsTitle')} — ${
                JOB_I18N[historyJob.job_key] ? t(JOB_I18N[historyJob.job_key].name) : historyJob.name
              }`
            : t('cron.runsTitle')
        }
        widthClass="max-w-2xl"
      >
        {runsLoading ? (
          <LoadingState label={t('common.loading')} />
        ) : runs.length === 0 ? (
          <div className="py-12 text-center text-xs text-slate-400">{t('common.noData')}</div>
        ) : (
          <div className="space-y-3">
            {runs.map((r) => (
              <div
                key={r.id}
                className="p-3.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40 text-xs space-y-1.5"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <StatusBadge status={r.status} />
                    <Badge tone="slate">{r.trigger_type}</Badge>
                    <span className="text-slate-500">{r.triggered_by}</span>
                  </div>
                  <span className="text-slate-400">{formatTime(r.started_at)}</span>
                </div>
                {r.output_log && (
                  <pre className="text-[11px] font-mono text-slate-700 dark:text-slate-300 whitespace-pre-wrap break-words bg-white dark:bg-slate-900 p-2.5 rounded-lg border border-slate-200/70 dark:border-slate-800">
                    {r.output_log}
                  </pre>
                )}
              </div>
            ))}
          </div>
        )}
      </Drawer>
    </div>
  )
}
