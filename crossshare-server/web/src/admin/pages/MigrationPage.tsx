import React, { useRef, useState } from 'react'
import { AlertTriangle, Download, FileCheck2, ShieldAlert, Upload } from 'lucide-react'
import { APIError, api, formatTime } from '../api'
import { Badge } from '../components/Badge'
import { useToast } from '../components/Toast'
import { useI18n } from '../contexts/I18nContext'
import type { InspectSummary } from '../types'

export const MigrationPage: React.FC = () => {
  const toast = useToast()
  const { t } = useI18n()

  // Export state
  const [expPass, setExpPass] = useState('')
  const [expAdmins, setExpAdmins] = useState(true)
  const [expAudit, setExpAudit] = useState(false)
  const [expCronRuns, setExpCronRuns] = useState(false)
  const [exporting, setExporting] = useState(false)

  // Import state
  const [impFile, setImpFile] = useState<File | null>(null)
  const [impPass, setImpPass] = useState('')
  const [needPassHint, setNeedPassHint] = useState(false)
  const [impMode, setImpMode] = useState<'merge' | 'replace'>('merge')
  const [impConflict, setImpConflict] = useState<'overwrite' | 'skip'>('overwrite')
  const [impAdmins, setImpAdmins] = useState(true)
  const [impAudit, setImpAudit] = useState(false)
  const [impCronRuns, setImpCronRuns] = useState(false)
  const [inspectRes, setInspectRes] = useState<InspectSummary | null>(null)
  const [inspecting, setInspecting] = useState(false)
  const [importing, setImporting] = useState(false)

  const impPassRef = useRef<HTMLInputElement | null>(null)

  const handleExport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (expPass !== '' && expPass.length < 6) {
      toast.error(t('migration.passphrasePlaceholder'))
      return
    }
    setExporting(true)
    try {
      await api.exportMigration({
        include_admins: expAdmins,
        include_audit_logs: expAudit,
        include_cron_runs: expCronRuns,
        passphrase: expPass,
      })
      toast.success(t('migration.exportSuccess'))
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setExporting(false)
    }
  }

  const handleInspect = async () => {
    if (!impFile) return
    setInspecting(true)
    setNeedPassHint(false)
    const fd = new FormData()
    fd.append('file', impFile)
    fd.append('passphrase', impPass)
    try {
      const summary = await api.inspectMigration(fd)
      setInspectRes(summary)
      toast.info(t('migration.inspectDone'))
    } catch (e: any) {
      setInspectRes(null)
      if (e instanceof APIError && e.code === 'passphrase_required') {
        setNeedPassHint(true)
        setTimeout(() => impPassRef.current?.focus(), 0)
        toast.info(t('migration.passRequiredHint'))
      } else {
        toast.error(e.message)
      }
    } finally {
      setInspecting(false)
    }
  }

  const handleImport = async () => {
    if (!impFile || !inspectRes || importing) return
    if (impMode === 'replace' && !window.confirm(t('migration.replaceConfirm'))) {
      return
    }
    setImporting(true)
    const fd = new FormData()
    fd.append('file', impFile)
    fd.append('passphrase', impPass)
    fd.append('mode', impMode)
    fd.append('conflict_strategy', impConflict)
    fd.append('include_admins', String(impAdmins))
    fd.append('include_audit_logs', String(impAudit))
    fd.append('include_cron_runs', String(impCronRuns))
    try {
      const res = await api.importMigration(fd)
      toast.success(
        t('migration.importSuccess', {
          settings: res.applied.system_settings || 0,
          cron: res.applied.cron_jobs || 0,
          admins: res.applied.admins || 0,
        }),
      )
      setInspectRes(null)
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setImporting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('migration.title')}</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('migration.subtitle')}</p>
      </div>

      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-indigo-100 dark:border-indigo-500/30 bg-indigo-50/60 dark:bg-indigo-500/10 p-4 text-xs text-indigo-800 dark:text-indigo-200">
        <span className="font-semibold">{t('migration.flowLabel')}</span>
        <span>{t('migration.flowStep1')}</span>
        <span aria-hidden="true">→</span>
        <span>{t('migration.flowStep2')}</span>
        <span aria-hidden="true">→</span>
        <span>{t('migration.flowStep3')}</span>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Export */}
        <form
          onSubmit={handleExport}
          className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4 text-xs"
        >
          <div className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
            <Download className="w-4 h-4 text-indigo-500" />
            <h2>{t('migration.exportTitle')}</h2>
          </div>
          <p className="text-slate-500 dark:text-slate-400 leading-5">{t('migration.exportDesc')}</p>

          <div>
            <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('migration.passphraseLabel')}
            </label>
            <input
              type="password"
              value={expPass}
              onChange={(e) => setExpPass(e.target.value)}
              placeholder={t('migration.passphrasePlaceholder')}
              className="w-full px-3 py-2 border rounded-xl text-xs"
            />
          </div>

          {expPass === '' && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-amber-50 dark:bg-amber-500/15 border border-amber-200 dark:border-amber-500/30 text-amber-800 dark:text-amber-200">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <span className="leading-5">{t('migration.plainWarn')}</span>
            </div>
          )}

          <div className="space-y-2 pt-1">
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={expAdmins} onChange={(e) => setExpAdmins(e.target.checked)} />
              <span>{t('migration.includeAdmins')}</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={expAudit} onChange={(e) => setExpAudit(e.target.checked)} />
              <span>{t('migration.includeAudit')}</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={expCronRuns} onChange={(e) => setExpCronRuns(e.target.checked)} />
              <span>{t('migration.includeCronRuns')}</span>
            </label>
          </div>

          <button type="submit" disabled={exporting} className="primary-button w-full py-2.5">
            <Download className="w-4 h-4" />
            <span>{t('migration.exportBtn')}</span>
          </button>
        </form>

        {/* Import */}
        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4 text-xs">
          <div className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-slate-100">
            <Upload className="w-4 h-4 text-indigo-500" />
            <h2>{t('migration.importTitle')}</h2>
          </div>
          <p className="text-slate-500 dark:text-slate-400 leading-5">{t('migration.importDesc')}</p>

          <div>
            <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('migration.fileLabel')}
            </label>
            <input
              type="file"
              accept=".csbundle,application/octet-stream"
              onChange={(e) => {
                setImpFile(e.target.files?.[0] || null)
                setInspectRes(null)
                setNeedPassHint(false)
              }}
              className="w-full text-xs"
            />
          </div>

          <div>
            <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('migration.passphraseLabel')}
            </label>
            <input
              ref={impPassRef}
              type="password"
              value={impPass}
              onChange={(e) => setImpPass(e.target.value)}
              placeholder={t('migration.importPassPlaceholder')}
              className={`w-full px-3 py-2 border rounded-xl text-xs ${
                needPassHint ? 'border-amber-400 ring-2 ring-amber-400/20' : ''
              }`}
            />
            {needPassHint && <p className="text-amber-600 dark:text-amber-400 mt-1">{t('migration.passRequiredHint')}</p>}
          </div>

          <button
            type="button"
            disabled={!impFile || inspecting}
            onClick={handleInspect}
            className="secondary-button w-full py-2"
          >
            <FileCheck2 className="w-4 h-4" />
            <span>{t('migration.inspectBtn')}</span>
          </button>

          {inspectRes && (
            <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-slate-800 dark:text-slate-100">{t('migration.summaryTitle')}</span>
                <Badge tone={inspectRes.encrypted ? 'emerald' : 'amber'}>
                  {inspectRes.encrypted ? t('migration.encryptedYes') : t('migration.encryptedNo')}
                </Badge>
              </div>
              <div className="text-slate-500">{t('migration.bundleTime', { time: formatTime(inspectRes.exported_at) })}</div>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 font-mono text-[11px]">
                <div>settings: {inspectRes.counts.system_settings}</div>
                <div>cron_jobs: {inspectRes.counts.cron_jobs}</div>
                <div>admins: {inspectRes.counts.admins}</div>
                <div>audit_logs: {inspectRes.counts.audit_logs}</div>
                <div>cron_runs: {inspectRes.counts.cron_job_runs}</div>
              </div>

              {inspectRes.admin_conflicts?.length > 0 && (
                <div className="flex items-start gap-2 p-2.5 rounded-lg bg-amber-50 dark:bg-amber-500/15 text-amber-800 dark:text-amber-200">
                  <ShieldAlert className="w-4 h-4 shrink-0 mt-0.5" />
                  <span>{t('migration.conflicts', { names: inspectRes.admin_conflicts.join(', ') })}</span>
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                <div>
                  <label className="block font-medium mb-1">{t('migration.modeLabel')}</label>
                  <select
                    value={impMode}
                    onChange={(e) => setImpMode(e.target.value === 'replace' ? 'replace' : 'merge')}
                    className="w-full px-2.5 py-1.5 border rounded-lg text-xs"
                  >
                    <option value="merge">{t('migration.modeMerge')}</option>
                    <option value="replace">{t('migration.modeReplace')}</option>
                  </select>
                </div>
                {impMode === 'merge' && (
                  <div>
                    <label className="block font-medium mb-1">{t('migration.conflictLabel')}</label>
                    <select
                      value={impConflict}
                      onChange={(e) => setImpConflict(e.target.value === 'skip' ? 'skip' : 'overwrite')}
                      className="w-full px-2.5 py-1.5 border rounded-lg text-xs"
                    >
                      <option value="overwrite">{t('migration.conflictOverwrite')}</option>
                      <option value="skip">{t('migration.conflictSkip')}</option>
                    </select>
                  </div>
                )}
              </div>

              <div className="space-y-1.5 pt-1">
                {inspectRes.includes.admins && (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={impAdmins} onChange={(e) => setImpAdmins(e.target.checked)} />
                    <span>{t('migration.includeAdmins')}</span>
                  </label>
                )}
                {inspectRes.includes.audit_logs && (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={impAudit} onChange={(e) => setImpAudit(e.target.checked)} />
                    <span>{t('migration.includeAudit')}</span>
                  </label>
                )}
                {inspectRes.includes.cron_runs && (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={impCronRuns} onChange={(e) => setImpCronRuns(e.target.checked)} />
                    <span>{t('migration.includeCronRuns')}</span>
                  </label>
                )}
              </div>

              <button
                type="button"
                disabled={importing}
                onClick={handleImport}
                className="primary-button w-full py-2"
              >
                {t('migration.importBtn')}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
