import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { KeyRound, Lock, Plus, RefreshCw, Save, Shield, Users } from 'lucide-react'
import { api, formatBytes, formatTime } from '../api'
import { Badge } from '../components/Badge'
import { LoadingState } from '../components/Feedback'
import { Modal } from '../components/Modal'
import { SecretField } from '../components/SecretField'
import { useToast } from '../components/Toast'
import { useI18n } from '../contexts/I18nContext'
import type { Admin, SystemSettings } from '../types'

interface UnitOption {
  key: string
  label: string
  factor: number
}

function pickBestUnit(raw: number, units: UnitOption[]): UnitOption {
  if (raw > 0) {
    for (let i = units.length - 1; i >= 0; i--) {
      if (raw >= units[i].factor && raw % units[i].factor === 0) {
        return units[i]
      }
    }
  }
  return units[0]
}

const UnitNumberInput: React.FC<{
  value: number
  syncVersion: number
  units: UnitOption[]
  onChange: (rawValue: number) => void
  preview: string
}> = ({ value, syncVersion, units, onChange, preview }) => {
  const [unitKey, setUnitKey] = useState<string>(() => pickBestUnit(value, units).key)
  const [text, setText] = useState<string>(() => {
    const best = pickBestUnit(value, units)
    return String(value / best.factor)
  })

  useEffect(() => {
    const best = pickBestUnit(value, units)
    setUnitKey(best.key)
    setText(String(value / best.factor))
    // Re-sync display unit only when backend data is reloaded/saved
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [syncVersion])

  const currentUnit = units.find((u) => u.key === unitKey) || units[0]

  return (
    <div>
      <div className="flex items-stretch gap-1.5">
        <input
          type="number"
          step="any"
          min={0}
          required
          value={text}
          onChange={(e) => {
            const rawText = e.target.value
            setText(rawText)
            const num = Number(rawText)
            if (Number.isFinite(num) && num >= 0) {
              onChange(Math.round(num * currentUnit.factor))
            }
          }}
          className="flex-1 min-w-0 px-3 py-2 border rounded-xl text-xs font-mono"
        />
        <select
          value={unitKey}
          onChange={(e) => {
            const nextKey = e.target.value
            setUnitKey(nextKey)
            const nextUnit = units.find((u) => u.key === nextKey) || units[0]
            const num = Number(text)
            if (Number.isFinite(num) && num >= 0) {
              onChange(Math.round(num * nextUnit.factor))
            }
          }}
          className="w-24 shrink-0 px-2.5 py-2 border rounded-xl text-xs bg-white dark:bg-slate-900"
        >
          {units.map((u) => (
            <option key={u.key} value={u.key}>
              {u.label}
            </option>
          ))}
        </select>
      </div>
      <div className="text-[11px] text-slate-400 font-mono mt-1">{preview}</div>
    </div>
  )
}

export const SettingsPage: React.FC = () => {
  const { t } = useI18n()
  const toast = useToast()

  const ttlUnits = useMemo<UnitOption[]>(
    () => [
      { key: 's', label: t('unit.second'), factor: 1 },
      { key: 'm', label: t('unit.minute'), factor: 60 },
      { key: 'h', label: t('unit.hour'), factor: 3600 },
      { key: 'd', label: t('unit.day'), factor: 86400 },
    ],
    [t],
  )

  const sizeUnits = useMemo<UnitOption[]>(
    () => [
      { key: 'B', label: 'B', factor: 1 },
      { key: 'KB', label: 'KB', factor: 1024 },
      { key: 'MB', label: 'MB', factor: 1024 * 1024 },
      { key: 'GB', label: 'GB', factor: 1024 * 1024 * 1024 },
    ],
    [],
  )

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [syncVersion, setSyncVersion] = useState(0)

  // Settings form
  const [form, setForm] = useState<SystemSettings>({
    default_ttl: 600,
    max_ttl: 86400,
    text_json_limit: 1048576,
    files_push_limit: 20971520,
    max_files_per_push: 20,
    ratelimit_enable: true,
    ratelimit_rpm: 20,
    client_auth_enable: false,
    has_client_jwt_secret: false,
    audit_client_ops: true,
    audit_retention_days: 90,
  })
  const [jwtSecretInput, setJwtSecretInput] = useState('')
  const [clearJwtSecret, setClearJwtSecret] = useState(false)

  // Password form
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [changingPwd, setChangingPwd] = useState(false)

  // Admins list & modal
  const [admins, setAdmins] = useState<Admin[]>([])
  const [addOpen, setAddOpen] = useState(false)
  const [newAdminUser, setNewAdminUser] = useState('')
  const [newAdminPass, setNewAdminPass] = useState('')
  const [editingAdmin, setEditingAdmin] = useState<Admin | null>(null)
  const [editActive, setEditActive] = useState(true)
  const [editPass, setEditPass] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [s, adm] = await Promise.all([api.getSettings(), api.listAdmins()])
      setForm(s)
      setSyncVersion((v) => v + 1)
      setJwtSecretInput('')
      setClearJwtSecret(false)
      setAdmins(adm.items || [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleGenerateSecret = async () => {
    try {
      const res = await api.generateJWTSecret()
      setJwtSecretInput(res.client_jwt_secret)
      setClearJwtSecret(false)
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const handleSaveSettings = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        default_ttl: Number(form.default_ttl),
        max_ttl: Number(form.max_ttl),
        text_json_limit: Number(form.text_json_limit),
        files_push_limit: Number(form.files_push_limit),
        max_files_per_push: Number(form.max_files_per_push),
        ratelimit_enable: form.ratelimit_enable,
        ratelimit_rpm: Number(form.ratelimit_rpm),
        client_auth_enable: form.client_auth_enable,
        audit_client_ops: form.audit_client_ops,
        audit_retention_days: Number(form.audit_retention_days),
      }
      if (clearJwtSecret) {
        payload.clear_client_jwt_secret = true
      } else if (jwtSecretInput !== '') {
        payload.client_jwt_secret = jwtSecretInput
      }

      const updated = await api.updateSettings(payload)
      setForm(updated)
      setSyncVersion((v) => v + 1)
      setJwtSecretInput('')
      setClearJwtSecret(false)
      toast.success(t('settings.saved'))
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }
    setChangingPwd(true)
    try {
      await api.changePassword(oldPassword, newPassword)
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success(t('settings.passwordChanged'))
    } catch (e: any) {
      toast.error(e.message)
    } finally {
      setChangingPwd(false)
    }
  }

  const handleCreateAdmin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.createAdmin({
        username: newAdminUser,
        password: newAdminPass,
        is_active: true,
      })
      toast.success(t('settings.adminCreated'))
      setAddOpen(false)
      setNewAdminUser('')
      setNewAdminPass('')
      load()
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  const handleUpdateAdmin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingAdmin) return
    try {
      await api.updateAdmin(editingAdmin.id, {
        is_active: editActive,
        password: editPass || undefined,
      })
      toast.success(t('settings.adminUpdated'))
      setEditingAdmin(null)
      load()
    } catch (e: any) {
      toast.error(e.message)
    }
  }

  if (loading) return <LoadingState label={t('common.loading')} />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('settings.title')}</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('settings.subtitle')}</p>
      </div>

      {/* System Settings Form */}
      <form onSubmit={handleSaveSettings} className="space-y-6 text-xs">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Business & Rate Limit */}
          <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
              <Shield className="w-4 h-4 text-indigo-500" />
              <h2>{t('settings.businessTitle')}</h2>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.defaultTtl')}
                </label>
                <UnitNumberInput
                  value={form.default_ttl}
                  syncVersion={syncVersion}
                  units={ttlUnits}
                  onChange={(v) => setForm((f) => ({ ...f, default_ttl: v }))}
                  preview={`= ${form.default_ttl} ${t('unit.second')}`}
                />
              </div>
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.maxTtl')}
                </label>
                <UnitNumberInput
                  value={form.max_ttl}
                  syncVersion={syncVersion}
                  units={ttlUnits}
                  onChange={(v) => setForm((f) => ({ ...f, max_ttl: v }))}
                  preview={`= ${form.max_ttl} ${t('unit.second')}`}
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.textLimit')}
                </label>
                <UnitNumberInput
                  value={form.text_json_limit}
                  syncVersion={syncVersion}
                  units={sizeUnits}
                  onChange={(v) => setForm((f) => ({ ...f, text_json_limit: v }))}
                  preview={`= ${form.text_json_limit.toLocaleString()} B (${formatBytes(form.text_json_limit)})`}
                />
              </div>
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.filesLimit')}
                </label>
                <UnitNumberInput
                  value={form.files_push_limit}
                  syncVersion={syncVersion}
                  units={sizeUnits}
                  onChange={(v) => setForm((f) => ({ ...f, files_push_limit: v }))}
                  preview={`= ${form.files_push_limit.toLocaleString()} B (${formatBytes(form.files_push_limit)})`}
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 items-end">
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.maxFiles')}
                </label>
                <input
                  type="number"
                  min={1}
                  value={form.max_files_per_push}
                  onChange={(e) => setForm((f) => ({ ...f, max_files_per_push: Number(e.target.value) }))}
                  className="w-full px-3 py-2 border rounded-xl text-xs"
                />
              </div>
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.rateRpm')}
                </label>
                <input
                  type="number"
                  min={1}
                  disabled={!form.ratelimit_enable}
                  value={form.ratelimit_rpm}
                  onChange={(e) => setForm((f) => ({ ...f, ratelimit_rpm: Number(e.target.value) }))}
                  className="w-full px-3 py-2 border rounded-xl text-xs"
                />
              </div>
            </div>

            <label className="flex items-center gap-2 pt-1 cursor-pointer">
              <input
                type="checkbox"
                checked={form.ratelimit_enable}
                onChange={(e) => setForm((f) => ({ ...f, ratelimit_enable: e.target.checked }))}
              />
              <span className="font-medium">{t('settings.rateEnable')}</span>
            </label>
          </div>

          {/* Client Auth & Audit Policy */}
          <div className="space-y-6">
            <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                <KeyRound className="w-4 h-4 text-indigo-500" />
                <h2>{t('settings.clientAuthTitle')}</h2>
              </div>
              <p className="text-slate-500 dark:text-slate-400 leading-5">{t('settings.clientAuthDesc')}</p>

              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={form.client_auth_enable}
                  onChange={(e) => setForm((f) => ({ ...f, client_auth_enable: e.target.checked }))}
                />
                <span className="font-medium">{t('settings.clientAuthEnable')}</span>
              </label>

              <div>
                <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
                  <label className="font-medium text-slate-700 dark:text-slate-300">
                    {t('settings.clientJwtSecret')}
                  </label>
                  <div className="flex items-center gap-3">
                    <button
                      type="button"
                      onClick={handleGenerateSecret}
                      className="inline-flex items-center gap-1 text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
                    >
                      <RefreshCw className="w-3 h-3" />
                      <span>{t('settings.generateSecret')}</span>
                    </button>
                    {(form.has_client_jwt_secret || jwtSecretInput) && (
                      <button
                        type="button"
                        onClick={() => {
                          setJwtSecretInput('')
                          setClearJwtSecret(true)
                        }}
                        className="text-rose-600 hover:underline"
                      >
                        {t('settings.clearSecret')}
                      </button>
                    )}
                  </div>
                </div>
                <SecretField
                  value={jwtSecretInput}
                  hasSaved={form.has_client_jwt_secret && !clearJwtSecret}
                  onChange={(v) => {
                    setJwtSecretInput(v)
                    setClearJwtSecret(false)
                  }}
                  onReveal={async () => {
                    const res = await api.revealJWTSecret()
                    return res.client_jwt_secret
                  }}
                />
              </div>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4">
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{t('settings.auditTitle')}</h2>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={form.audit_client_ops}
                  onChange={(e) => setForm((f) => ({ ...f, audit_client_ops: e.target.checked }))}
                />
                <span className="font-medium">{t('settings.auditClientOps')}</span>
              </label>
              <div>
                <label className="block font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('settings.auditRetention')}
                </label>
                <input
                  type="number"
                  min={1}
                  max={36500}
                  value={form.audit_retention_days}
                  onChange={(e) => setForm((f) => ({ ...f, audit_retention_days: Number(e.target.value) }))}
                  className="w-40 px-3 py-2 border rounded-xl text-xs"
                />
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <button type="submit" disabled={saving} className="primary-button px-6 py-2.5">
            <Save className="w-4 h-4" />
            <span>{saving ? t('common.saving') : t('settings.saveAll')}</span>
          </button>
        </div>
      </form>

      {/* Bottom: Change Password & Admin Accounts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 text-xs">
        <form
          onSubmit={handleChangePassword}
          className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4"
        >
          <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
            <Lock className="w-4 h-4 text-indigo-500" />
            <h2>{t('settings.passwordTitle')}</h2>
          </div>
          <div>
            <label className="block font-medium mb-1">{t('settings.oldPassword')}</label>
            <input
              type="password"
              required
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              className="w-full px-3 py-2 border rounded-xl text-xs"
            />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block font-medium mb-1">{t('settings.newPassword')}</label>
              <input
                type="password"
                required
                minLength={8}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-3 py-2 border rounded-xl text-xs"
              />
            </div>
            <div>
              <label className="block font-medium mb-1">{t('settings.confirmNewPassword')}</label>
              <input
                type="password"
                required
                minLength={8}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full px-3 py-2 border rounded-xl text-xs"
              />
            </div>
          </div>
          <button type="submit" disabled={changingPwd} className="secondary-button">
            {t('settings.changePasswordBtn')}
          </button>
        </form>

        <div className="bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
              <Users className="w-4 h-4 text-indigo-500" />
              <h2>{t('settings.adminsTitle')}</h2>
            </div>
            <button type="button" onClick={() => setAddOpen(true)} className="secondary-button py-1.5 px-3 text-xs">
              <Plus className="w-3.5 h-3.5" />
              <span>{t('settings.addAdmin')}</span>
            </button>
          </div>
          <div className="divide-y divide-slate-100 dark:divide-slate-800">
            {admins.map((a) => (
              <div key={a.id} className="py-2.5 flex items-center justify-between gap-3">
                <div>
                  <div className="font-medium text-slate-800 dark:text-slate-200">{a.username}</div>
                  <div className="text-[11px] text-slate-400">
                    {a.last_login_at ? formatTime(a.last_login_at) : formatTime(a.created_at)}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge tone={a.is_active ? 'emerald' : 'slate'}>
                    {a.is_active ? t('common.enabled') : t('common.disabled')}
                  </Badge>
                  <button
                    type="button"
                    onClick={() => {
                      setEditingAdmin(a)
                      setEditActive(a.is_active)
                      setEditPass('')
                    }}
                    className="text-indigo-600 dark:text-indigo-400 hover:underline"
                  >
                    {t('common.edit')}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Add Admin Modal */}
      <Modal open={addOpen} onClose={() => setAddOpen(false)} title={t('settings.addAdmin')}>
        <form onSubmit={handleCreateAdmin} className="space-y-4 text-xs">
          <div>
            <label className="block font-medium mb-1">{t('auth.username')}</label>
            <input
              type="text"
              required
              minLength={3}
              value={newAdminUser}
              onChange={(e) => setNewAdminUser(e.target.value)}
              className="w-full px-3 py-2 border rounded-xl text-xs"
            />
          </div>
          <div>
            <label className="block font-medium mb-1">{t('auth.password')}</label>
            <input
              type="password"
              required
              minLength={8}
              value={newAdminPass}
              onChange={(e) => setNewAdminPass(e.target.value)}
              className="w-full px-3 py-2 border rounded-xl text-xs"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setAddOpen(false)} className="secondary-button">
              {t('common.cancel')}
            </button>
            <button type="submit" className="primary-button">
              {t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      {/* Edit Admin Modal */}
      <Modal
        open={Boolean(editingAdmin)}
        onClose={() => setEditingAdmin(null)}
        title={editingAdmin ? `${t('common.edit')} — ${editingAdmin.username}` : t('common.edit')}
      >
        <form onSubmit={handleUpdateAdmin} className="space-y-4 text-xs">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={editActive} onChange={(e) => setEditActive(e.target.checked)} />
            <span>{t('common.enable')}</span>
          </label>
          <div>
            <label className="block font-medium mb-1">{t('settings.resetPasswordOptional')}</label>
            <input
              type="password"
              minLength={8}
              value={editPass}
              onChange={(e) => setEditPass(e.target.value)}
              placeholder={t('auth.passwordPlaceholder')}
              className="w-full px-3 py-2 border rounded-xl text-xs"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setEditingAdmin(null)} className="secondary-button">
              {t('common.cancel')}
            </button>
            <button type="submit" className="primary-button">
              {t('common.save')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
