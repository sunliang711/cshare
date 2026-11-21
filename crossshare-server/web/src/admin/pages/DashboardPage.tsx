import React, { useCallback, useEffect, useState } from 'react'
import { Activity, ArrowRight, Database, HardDrive, Radio, RefreshCw, Share2 } from 'lucide-react'
import { api, formatBytes, formatTime } from '../api'
import { Badge, StatusBadge } from '../components/Badge'
import type { NavTab } from '../components/Layout'
import { LoadingState } from '../components/Feedback'
import { useI18n } from '../contexts/I18nContext'
import type { DashboardSummary } from '../types'

interface Props {
  onNavigate: (tab: NavTab) => void
}

export const DashboardPage: React.FC<Props> = ({ onNavigate }) => {
  const { t } = useI18n()
  const [data, setData] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.dashboardSummary()
      setData(res)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  if (loading && !data) {
    return <LoadingState label={t('common.loading')} />
  }
  if (!data) return null

  const maxTraffic = Math.max(
    1,
    ...data.snapshots.map((s) => Math.max(s.push_count, s.pull_count, s.active_shares)),
  )

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('dash.title')}</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('dash.subtitle')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={data.client_auth_enabled ? 'indigo' : 'slate'}>
            {t('dash.authBadge', { state: data.client_auth_enabled ? t('common.enabled') : t('common.disabled') })}
          </Badge>
          <Badge tone={data.ratelimit_enabled ? 'emerald' : 'slate'}>
            {t('dash.rateBadge', {
              state: data.ratelimit_enabled ? t('dash.rateRpm', { rpm: data.ratelimit_rpm }) : t('common.disabled'),
            })}
          </Badge>
          <button type="button" onClick={load} className="secondary-button">
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>{t('common.refresh')}</span>
          </button>
        </div>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{t('dash.activeShares')}</span>
            <span className="w-9 h-9 rounded-xl bg-indigo-50 dark:bg-indigo-500/15 text-indigo-600 dark:text-indigo-400 flex items-center justify-center">
              <Share2 className="w-4 h-4" />
            </span>
          </div>
          <div className="text-3xl font-bold text-slate-900 dark:text-slate-100 mt-3">
            {data.storage.active_shares}
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            {t('dash.storedBytes', { size: formatBytes(data.storage.total_bytes) })}
          </div>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{t('dash.todayTraffic')}</span>
            <span className="w-9 h-9 rounded-xl bg-emerald-50 dark:bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center">
              <Activity className="w-4 h-4" />
            </span>
          </div>
          <div className="text-3xl font-bold text-slate-900 dark:text-slate-100 mt-3">
            {data.today.push_count + data.today.pull_count}
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            {t('dash.todayDetail', {
              push: data.today.push_count,
              pull: data.today.pull_count,
              del: data.today.delete_count,
            })}
          </div>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{t('dash.p2pSessions')}</span>
            <span className="w-9 h-9 rounded-xl bg-sky-50 dark:bg-sky-500/15 text-sky-600 dark:text-sky-400 flex items-center justify-center">
              <Radio className="w-4 h-4" />
            </span>
          </div>
          <div className="text-3xl font-bold text-slate-900 dark:text-slate-100 mt-3">
            {data.p2p_sessions}
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">{t('dash.p2pDesc')}</div>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{t('dash.storageHealth')}</span>
            <span className="w-9 h-9 rounded-xl bg-amber-50 dark:bg-amber-500/15 text-amber-600 dark:text-amber-400 flex items-center justify-center">
              {data.storage.backend === 'redis' ? <Database className="w-4 h-4" /> : <HardDrive className="w-4 h-4" />}
            </span>
          </div>
          <div className="flex items-center gap-2 mt-3">
            <span className="text-2xl font-bold text-slate-900 dark:text-slate-100 uppercase">
              {data.storage.backend || 'memory'}
            </span>
            <Badge tone={data.storage.healthy ? 'emerald' : 'rose'}>
              {data.storage.healthy ? 'OK' : 'Error'}
            </Badge>
          </div>
          <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 truncate" title={data.storage.error || ''}>
            {data.storage.healthy ? t('dash.latency', { ms: data.storage.latency_ms }) : data.storage.error}
          </div>
        </div>
      </div>

      {/* Hourly Snapshots Trend */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
        <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('dash.trendTitle')}</h2>
          <div className="flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400">
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-indigo-500" />
              {t('dash.trendPush')}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-emerald-500" />
              {t('dash.trendPull')}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-amber-400" />
              {t('dash.trendShares')}
            </span>
          </div>
        </div>

        {data.snapshots.length === 0 ? (
          <div className="py-10 text-center text-xs text-slate-500 dark:text-slate-400">
            {t('dash.trendEmpty')}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <div className="flex items-end gap-2 h-40 min-w-[420px] pt-4 pb-1 px-1">
              {data.snapshots.map((snap) => {
                const label = new Date(snap.bucket_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
                const pushH = Math.max(4, Math.round((snap.push_count / maxTraffic) * 100))
                const pullH = Math.max(4, Math.round((snap.pull_count / maxTraffic) * 100))
                const shareH = Math.max(4, Math.round((snap.active_shares / maxTraffic) * 100))
                return (
                  <div
                    key={snap.id}
                    className="flex-1 min-w-[26px] flex flex-col items-center gap-1.5"
                    title={`${label} | Push: ${snap.push_count}, Pull: ${snap.pull_count}, Shares: ${snap.active_shares}`}
                  >
                    <div className="w-full h-28 flex items-end justify-center gap-0.5">
                      <div className="w-2 rounded-t bg-indigo-500/85" style={{ height: `${pushH}%` }} />
                      <div className="w-2 rounded-t bg-emerald-500/85" style={{ height: `${pullH}%` }} />
                      <div className="w-2 rounded-t bg-amber-400/85" style={{ height: `${shareH}%` }} />
                    </div>
                    <span className="text-[10px] font-mono text-slate-400">{label}</span>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Bottom 2 columns: Cron & Audit */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('dash.cronTitle')}</h2>
            <button
              type="button"
              onClick={() => onNavigate('cron')}
              className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline inline-flex items-center gap-1"
            >
              <span>{t('dash.viewAll')}</span>
              <ArrowRight className="w-3 h-3" />
            </button>
          </div>
          <div className="divide-y divide-slate-100 dark:divide-slate-800">
            {data.cron_jobs.map((job) => (
              <div key={job.id} className="py-2.5 flex items-center justify-between gap-3 text-xs">
                <div className="min-w-0">
                  <div className="font-medium text-slate-800 dark:text-slate-200 truncate">{job.name}</div>
                  <div className="text-slate-400 font-mono text-[11px]">{job.cron_expr}</div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge tone={job.enabled ? 'emerald' : 'slate'}>
                    {job.enabled ? t('common.enabled') : t('common.disabled')}
                  </Badge>
                  {job.last_status && <StatusBadge status={job.last_status} />}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('dash.auditTitle')}</h2>
            <button
              type="button"
              onClick={() => onNavigate('audit')}
              className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline inline-flex items-center gap-1"
            >
              <span>{t('dash.viewAll')}</span>
              <ArrowRight className="w-3 h-3" />
            </button>
          </div>
          {data.recent_audit_logs.length === 0 ? (
            <div className="py-8 text-center text-xs text-slate-400">{t('common.noData')}</div>
          ) : (
            <div className="divide-y divide-slate-100 dark:divide-slate-800">
              {data.recent_audit_logs.map((log) => (
                <div key={log.id} className="py-2.5 flex items-center justify-between gap-3 text-xs">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Badge tone={log.actor_type === 'client' ? 'indigo' : 'slate'}>{log.actor_type}</Badge>
                      <span className="font-mono font-medium text-slate-800 dark:text-slate-200">{log.action}</span>
                      {log.share_key && <span className="font-mono text-indigo-500">#{log.share_key}</span>}
                    </div>
                    <div className="text-[11px] text-slate-400 mt-0.5">
                      {log.ip_address || '—'} · {formatTime(log.created_at)}
                    </div>
                  </div>
                  <StatusBadge status={log.status} />
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
