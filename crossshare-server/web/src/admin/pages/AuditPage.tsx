import React, { useCallback, useEffect, useState } from 'react'
import { Download, RefreshCw, ScrollText, Search } from 'lucide-react'
import { api, formatTime } from '../api'
import { Badge, StatusBadge } from '../components/Badge'
import { Drawer } from '../components/Drawer'
import { EmptyState, LoadingState } from '../components/Feedback'
import { useToast } from '../components/Toast'
import { useI18n } from '../contexts/I18nContext'
import type { AuditLog } from '../types'

const PAGE_SIZE = 30

export const AuditPage: React.FC = () => {
  const { t } = useI18n()
  const toast = useToast()
  const [items, setItems] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState<'csv' | 'json' | null>(null)
  const [page, setPage] = useState(0)

  // Filters
  const [actorType, setActorType] = useState('')
  const [action, setAction] = useState('')
  const [shareKey, setShareKey] = useState('')
  const [ip, setIp] = useState('')
  const [status, setStatus] = useState('')
  const [query, setQuery] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')

  const [selected, setSelected] = useState<AuditLog | null>(null)

  const buildFilterParams = useCallback(
    (includePaging: boolean) => {
      const params: Record<string, string> = {}
      if (actorType) params.actor_type = actorType
      if (action.trim()) params.action = action.trim()
      if (shareKey.trim()) params.share_key = shareKey.trim()
      if (ip.trim()) params.ip = ip.trim()
      if (status) params.status = status
      if (query.trim()) params.q = query.trim()
      if (fromDate) params.from = new Date(fromDate).toISOString()
      if (toDate) params.to = new Date(toDate).toISOString()
      if (includePaging) {
        params.limit = String(PAGE_SIZE)
        params.offset = String(page * PAGE_SIZE)
      }
      return params
    },
    [action, actorType, fromDate, ip, page, query, shareKey, status, toDate],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.listAuditLogs(buildFilterParams(true))
      setItems(res.items || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setLoading(false)
    }
  }, [buildFilterParams, toast])

  const handleExport = async (format: 'csv' | 'json') => {
    setExporting(format)
    try {
      await api.exportAuditLogs(buildFilterParams(false), format)
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setExporting(null)
    }
  }

  useEffect(() => {
    load()
  }, [load])

  const handleReset = () => {
    setActorType('')
    setAction('')
    setShareKey('')
    setIp('')
    setStatus('')
    setQuery('')
    setFromDate('')
    setToDate('')
    setPage(0)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('audit.title')}</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('audit.subtitle')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            disabled={exporting !== null}
            onClick={() => handleExport('csv')}
            className="secondary-button"
          >
            <Download className="w-3.5 h-3.5" />
            <span>{t('audit.exportCsv')}</span>
          </button>
          <button
            type="button"
            disabled={exporting !== null}
            onClick={() => handleExport('json')}
            className="secondary-button"
          >
            <Download className="w-3.5 h-3.5" />
            <span>{t('audit.exportJson')}</span>
          </button>
          <button type="button" onClick={load} className="secondary-button">
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>{t('common.refresh')}</span>
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-4 space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.actorType')}</label>
            <select
              value={actorType}
              onChange={(e) => {
                setActorType(e.target.value)
                setPage(0)
              }}
              className="w-full px-3 py-1.5 border rounded-lg text-xs"
            >
              <option value="">{t('common.all')}</option>
              <option value="client">{t('audit.actorClient')}</option>
              <option value="admin">{t('audit.actorAdmin')}</option>
              <option value="system">{t('audit.actorSystem')}</option>
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.action')}</label>
            <select
              value={action}
              onChange={(e) => {
                setAction(e.target.value)
                setPage(0)
              }}
              className="w-full px-3 py-1.5 border rounded-lg text-xs"
            >
              <option value="">{t('common.all')}</option>
              <option value="push.text">push.text</option>
              <option value="push.binary">push.binary</option>
              <option value="push.files">push.files</option>
              <option value="pull">pull</option>
              <option value="delete">delete</option>
              <option value="p2p.session.create">p2p.session.create</option>
              <option value="p2p.session.close">p2p.session.close</option>
              <option value="admin.login">admin.login</option>
              <option value="settings.update">settings.update</option>
              <option value="cron.update">cron.update</option>
              <option value="cron.run">cron.run</option>
              <option value="migration.export">migration.export</option>
              <option value="migration.import">migration.import</option>
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.status')}</label>
            <select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value)
                setPage(0)
              }}
              className="w-full px-3 py-1.5 border rounded-lg text-xs"
            >
              <option value="">{t('common.all')}</option>
              <option value="success">{t('audit.statusSuccess')}</option>
              <option value="failure">{t('audit.statusFailure')}</option>
            </select>
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.shareKey')}</label>
            <input
              type="text"
              value={shareKey}
              onChange={(e) => {
                setShareKey(e.target.value)
                setPage(0)
              }}
              placeholder="e.g. A8k2dP"
              className="w-full px-3 py-1.5 border rounded-lg text-xs font-mono"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 items-end">
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.ip')}</label>
            <input
              type="text"
              value={ip}
              onChange={(e) => {
                setIp(e.target.value)
                setPage(0)
              }}
              placeholder="192.168..."
              className="w-full px-3 py-1.5 border rounded-lg text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.from')}</label>
            <input
              type="datetime-local"
              value={fromDate}
              onChange={(e) => {
                setFromDate(e.target.value)
                setPage(0)
              }}
              className="w-full px-3 py-1.5 border rounded-lg text-xs"
            />
          </div>
          <div>
            <label className="block text-[11px] font-medium text-slate-500 mb-1">{t('audit.to')}</label>
            <input
              type="datetime-local"
              value={toDate}
              onChange={(e) => {
                setToDate(e.target.value)
                setPage(0)
              }}
              className="w-full px-3 py-1.5 border rounded-lg text-xs"
            />
          </div>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  setPage(0)
                }}
                placeholder={t('audit.keyword')}
                className="w-full pl-8 pr-3 py-1.5 border rounded-lg text-xs"
              />
            </div>
            <button type="button" onClick={handleReset} className="secondary-button text-xs py-1.5">
              {t('common.reset')}
            </button>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 overflow-hidden">
        {loading && items.length === 0 ? (
          <LoadingState label={t('common.loading')} />
        ) : items.length === 0 ? (
          <EmptyState icon={<ScrollText className="w-6 h-6" />} title={t('common.noData')} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colTime')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colActor')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colAction')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colTarget')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colIP')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('audit.colStatus')}</th>
                  <th className="px-4 py-3 text-right font-medium" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {items.map((log) => (
                  <tr
                    key={log.id}
                    onClick={() => setSelected(log)}
                    className="cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50"
                  >
                    <td className="px-4 py-3 whitespace-nowrap text-slate-500">{formatTime(log.created_at)}</td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex items-center gap-1.5">
                        <Badge tone={log.actor_type === 'client' ? 'indigo' : log.actor_type === 'admin' ? 'amber' : 'slate'}>
                          {log.actor_type}
                        </Badge>
                        <span className="text-slate-700 dark:text-slate-200">{log.actor_name || '—'}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 font-mono font-medium text-slate-800 dark:text-slate-200 whitespace-nowrap">
                      {log.action}
                    </td>
                    <td className="px-4 py-3 font-mono whitespace-nowrap text-slate-600 dark:text-slate-300">
                      {log.share_key ? `#${log.share_key}` : log.resource_id || '—'}
                    </td>
                    <td className="px-4 py-3 font-mono whitespace-nowrap text-slate-500">{log.ip_address || '—'}</td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <StatusBadge status={log.status} />
                    </td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelected(log)
                        }}
                        className="text-indigo-600 dark:text-indigo-400 hover:underline"
                      >
                        {t('common.detail')}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 dark:border-slate-800 text-xs text-slate-500">
          <span>{t('common.total', { count: total })}</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={page <= 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              className="secondary-button py-1 px-2.5 text-xs"
            >
              {t('common.prevPage')}
            </button>
            <span className="font-mono">
              {page + 1} / {totalPages}
            </span>
            <button
              type="button"
              disabled={page + 1 >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="secondary-button py-1 px-2.5 text-xs"
            >
              {t('common.nextPage')}
            </button>
          </div>
        </div>
      </div>

      <Drawer open={Boolean(selected)} onClose={() => setSelected(null)} title={t('audit.detailTitle')}>
        {selected && (
          <div className="space-y-4 text-xs">
            <div className="grid grid-cols-2 gap-3 p-4 rounded-xl bg-slate-50 dark:bg-slate-800/60">
              <div>
                <div className="text-slate-400">{t('audit.colTime')}</div>
                <div className="font-medium mt-0.5">{formatTime(selected.created_at)}</div>
              </div>
              <div>
                <div className="text-slate-400">{t('audit.colStatus')}</div>
                <div className="mt-0.5">
                  <StatusBadge status={selected.status} />
                </div>
              </div>
              <div>
                <div className="text-slate-400">{t('audit.colActor')}</div>
                <div className="font-medium mt-0.5">
                  {selected.actor_type} {selected.actor_name ? `(${selected.actor_name})` : ''}
                </div>
              </div>
              <div>
                <div className="text-slate-400">{t('audit.colAction')}</div>
                <div className="font-mono font-medium mt-0.5">{selected.action}</div>
              </div>
              <div>
                <div className="text-slate-400">{t('audit.shareKey')} / Resource</div>
                <div className="font-mono mt-0.5">{selected.share_key || selected.resource_id || '—'}</div>
              </div>
              <div>
                <div className="text-slate-400">{t('audit.colIP')}</div>
                <div className="font-mono mt-0.5">{selected.ip_address || '—'}</div>
              </div>
              <div className="col-span-2">
                <div className="text-slate-400">Request ID</div>
                <div className="font-mono mt-0.5 break-all">{selected.request_id || '—'}</div>
              </div>
              <div className="col-span-2">
                <div className="text-slate-400">User-Agent</div>
                <div className="font-mono mt-0.5 break-all">{selected.user_agent || '—'}</div>
              </div>
            </div>
            <div>
              <div className="text-slate-500 font-medium mb-1.5">Detail JSON</div>
              <pre className="p-4 rounded-xl bg-slate-900 text-slate-100 font-mono text-[11px] overflow-x-auto">
                {(() => {
                  try {
                    return JSON.stringify(JSON.parse(selected.detail_json || '{}'), null, 2)
                  } catch {
                    return selected.detail_json
                  }
                })()}
              </pre>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  )
}
