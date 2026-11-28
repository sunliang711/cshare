import React, { useCallback, useEffect, useState } from 'react'
import { Lock, Monitor, Plus, RefreshCw, Users } from 'lucide-react'
import { api, APIError, formatTime } from '../api'
import { Badge } from '../components/Badge'
import { EmptyState, LoadingState } from '../components/Feedback'
import { Modal } from '../components/Modal'
import { useToast } from '../components/Toast'
import { useI18n } from '../contexts/I18nContext'
import type { TranslationKey } from '../i18n/zh'
import type { Admin, AdminSession } from '../types'

interface Props {
  admin: Admin
  onAdminUpdate: (admin: Admin) => void
  onSignedOut: () => void
}

const accountErrors: Record<string, TranslationKey> = {
  wrong_current_password: 'accounts.wrongPassword',
  invalid_password: 'accounts.wrongPassword',
  last_admin: 'accounts.lastAdmin',
  username_taken: 'accounts.usernameTaken',
  weak_password: 'accounts.weakPassword',
  invalid_role: 'accounts.invalidRole',
  invalid_username: 'accounts.invalidUsername',
  cannot_disable_self: 'accounts.cannotDisableSelf',
  forbidden: 'accounts.forbidden',
  not_found: 'accounts.notFound',
}

export const AccountsPage: React.FC<Props> = ({ admin, onAdminUpdate, onSignedOut }) => {
  const { t } = useI18n()
  const toast = useToast()
  const isAdmin = admin.role === 'admin'
  const [sessions, setSessions] = useState<AdminSession[]>([])
  const [admins, setAdmins] = useState<Admin[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Password form
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [revokeTarget, setRevokeTarget] = useState<AdminSession | null>(null)

  // Admins list & modal
  const [accountOpen, setAccountOpen] = useState(false)
  const [editingAdmin, setEditingAdmin] = useState<Admin | null>(null)
  const [accountUser, setAccountUser] = useState('')
  const [accountPassword, setAccountPassword] = useState('')
  const [accountRole, setAccountRole] = useState<Admin['role']>('viewer')
  const [accountActive, setAccountActive] = useState(true)
  const [currentPassword, setCurrentPassword] = useState('')

  const errorMessage = useCallback((err: unknown) => {
    if (err instanceof APIError && accountErrors[err.code]) return t(accountErrors[err.code])
    return err instanceof Error ? err.message : t('auth.actionFailed')
  }, [t])

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [sessionRes, accountRes] = await Promise.all([
        api.listSessions(),
        isAdmin ? api.listAdmins() : Promise.resolve({ items: [] as Admin[] }),
      ])
      setSessions(sessionRes.sessions)
      setAdmins(accountRes.items)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [isAdmin, errorMessage])

  useEffect(() => { void load() }, [load])

  const changePassword = async (event: React.FormEvent) => {
    event.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }
    setBusy(true)
    try {
      const result = await api.changePassword(oldPassword, newPassword)
      if (result.admin) onAdminUpdate(result.admin)
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success(t('accounts.passwordChanged'))
      await load()
    } catch (err) {
      toast.error(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const revoke = async (session: AdminSession) => {
    setBusy(true)
    try {
      await api.revokeSession(session.id)
      setRevokeTarget(null)
      toast.success(t('accounts.sessionRevoked'))
      if (session.current) {
        onSignedOut()
        return
      }
      await load()
    } catch (err) {
      toast.error(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const openAccount = (account: Admin | null) => {
    setEditingAdmin(account)
    setAccountUser(account?.username || '')
    setAccountPassword('')
    setAccountRole(account?.role || 'viewer')
    setAccountActive(account?.is_active ?? true)
    setCurrentPassword('')
    setAccountOpen(true)
  }

  const saveAccount = async (event: React.FormEvent) => {
    event.preventDefault()
    setBusy(true)
    try {
      if (editingAdmin) {
        const updated = await api.updateAdmin(editingAdmin.id, {
          is_active: accountActive, role: accountRole,
          password: accountPassword || undefined, current_password: currentPassword,
        })
        if (updated.id === admin.id) {
          if (updated.role !== admin.role) {
            setAccountOpen(false)
            onSignedOut()
            return
          }
          onAdminUpdate(updated)
        }
      } else {
        await api.createAdmin({ username: accountUser, password: accountPassword, role: accountRole, current_password: currentPassword })
      }
      toast.success(t(editingAdmin ? 'accounts.updated' : 'accounts.created'))
      setAccountOpen(false)
      setCurrentPassword('')
      setAccountPassword('')
      await load()
    } catch (err) {
      toast.error(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const lastAdmin = editingAdmin?.role === 'admin' && editingAdmin.is_active &&
    admins.filter((account) => account.role === 'admin' && account.is_active).length <= 1
  const panel = 'bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-5 sm:p-6 space-y-4'
  const input = 'w-full px-3 py-2 border rounded-xl text-xs'

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">{t('accounts.title')}</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t(isAdmin ? 'accounts.subtitle' : 'accounts.viewerSubtitle')}</p>
        </div>
        <button type="button" onClick={() => void load()} disabled={loading || busy} className="secondary-button">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />{t('common.refresh')}
        </button>
      </div>
      {error && <div role="alert" className="text-sm text-rose-600 dark:text-rose-400">{error}</div>}

      {/* Bottom: Change Password & Admin Accounts */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6 text-xs">
        <form onSubmit={changePassword} className={panel}>
          <h2 className="flex items-center gap-2 text-sm font-semibold"><Lock className="w-4 h-4 text-indigo-500" />{t('accounts.passwordTitle', { username: admin.username })}</h2>
          <label className="block space-y-1.5">
            <span>{t('settings.oldPassword')}</span>
            <input type="password" autoComplete="current-password" required value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} className={input} />
          </label>
          <label className="block space-y-1.5">
            <span>{t('settings.newPassword')}</span>
            <input type="password" autoComplete="new-password" required minLength={8} maxLength={72} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} className={input} />
          </label>
          <label className="block space-y-1.5">
            <span>{t('settings.confirmNewPassword')}</span>
            <input type="password" autoComplete="new-password" required minLength={8} maxLength={72} value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} className={input} />
          </label>
          <button type="submit" disabled={busy} className="primary-button">{t('settings.changePasswordBtn')}</button>
        </form>

        <section className={panel}>
          <h2 className="flex items-center gap-2 text-sm font-semibold"><Monitor className="w-4 h-4 text-indigo-500" />{t('accounts.sessions')}</h2>
          {loading && sessions.length === 0 ? <LoadingState label={t('common.loading')} /> : sessions.length === 0 ? (
            <EmptyState title={t('accounts.noSessions')} />
          ) : (
            <div className="space-y-3 max-h-[420px] overflow-y-auto">
              {sessions.map((session) => (
                <div key={session.id} className="flex items-start justify-between gap-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/60 dark:bg-slate-800/40 p-3">
                  <div className="min-w-0 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono font-medium">{session.client_ip || '—'}</span>
                      {session.current && <Badge tone="emerald">{t('accounts.current')}</Badge>}
                    </div>
                    <div className="text-slate-500 dark:text-slate-400 break-words text-[11px]" title={session.user_agent}>{session.user_agent || '—'}</div>
                    <div className="text-slate-500 dark:text-slate-400">{t('accounts.lastSeen')} {formatTime(session.last_seen_at)}</div>
                    <div className="text-slate-500 dark:text-slate-400">{t('accounts.expires')} {formatTime(session.expires_at)}</div>
                  </div>
                  <button type="button" disabled={busy || loading} onClick={() => session.current ? setRevokeTarget(session) : void revoke(session)} className="secondary-button shrink-0 py-1.5 px-2.5">{t('accounts.revoke')}</button>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      {isAdmin && (
        <section className={panel}>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 className="flex items-center gap-2 text-sm font-semibold"><Users className="w-4 h-4 text-indigo-500" />{t('accounts.management')}</h2>
            <button type="button" onClick={() => openAccount(null)} disabled={busy} className="primary-button text-xs"><Plus className="w-4 h-4" />{t('accounts.add')}</button>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs whitespace-nowrap">
              <thead className="border-b border-slate-200 dark:border-slate-800 text-slate-500">
                <tr>
                  <th className="py-3 pr-4">{t('auth.username')}</th><th className="py-3 pr-4">{t('accounts.role')}</th>
                  <th className="py-3 pr-4">{t('accounts.status')}</th><th className="py-3 pr-4">{t('accounts.lastLogin')}</th>
                  <th className="py-3 text-right">{t('accounts.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {admins.map((account) => (
                  <tr key={account.id}>
                    <td className="py-3 pr-4 font-medium">{account.username}{account.id === admin.id && <span className="ml-2 text-slate-400">({t('accounts.you')})</span>}</td>
                    <td className="py-3 pr-4"><Badge tone={account.role === 'admin' ? 'indigo' : 'slate'}>{t(account.role === 'admin' ? 'app.roleAdmin' : 'app.roleViewer')}</Badge></td>
                    <td className="py-3 pr-4"><Badge tone={account.is_active ? 'emerald' : 'slate'}>{t(account.is_active ? 'common.enabled' : 'common.disabled')}</Badge></td>
                    <td className="py-3 pr-4 text-slate-500">{formatTime(account.last_login_at)}</td>
                    <td className="py-3 text-right"><button type="button" disabled={busy} onClick={() => openAccount(account)} className="secondary-button py-1.5 px-2.5">{t('common.edit')}</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Add Admin Modal */}
      {/* Edit Admin Modal */}
      <Modal open={accountOpen} onClose={() => !busy && setAccountOpen(false)} title={editingAdmin ? `${t('common.edit')} — ${editingAdmin.username}` : t('accounts.add')}>
        <form onSubmit={saveAccount} className="space-y-4 text-xs">
          {!editingAdmin && <label className="block space-y-1.5"><span>{t('auth.username')}</span><input required minLength={3} maxLength={64} autoComplete="off" value={accountUser} onChange={(e) => setAccountUser(e.target.value)} className={input} /></label>}
          <label className="block space-y-1.5">
            <span>{t('accounts.role')}</span>
            <select value={accountRole} disabled={lastAdmin} onChange={(e) => setAccountRole(e.target.value as Admin['role'])} className={input}>
              <option value="viewer">{t('app.roleViewer')}</option><option value="admin">{t('app.roleAdmin')}</option>
            </select>
          </label>
          {editingAdmin && <label className="flex items-center gap-2"><input type="checkbox" checked={accountActive} disabled={lastAdmin || editingAdmin.id === admin.id} onChange={(e) => setAccountActive(e.target.checked)} />{t('common.enable')}</label>}
          {lastAdmin && <p className="text-slate-500">{t('accounts.lastAdmin')}</p>}
          <label className="block space-y-1.5">
            <span>{t(editingAdmin ? 'settings.resetPasswordOptional' : 'auth.password')}</span>
            <input type="password" autoComplete="new-password" required={!editingAdmin} minLength={8} maxLength={72} value={accountPassword} onChange={(e) => setAccountPassword(e.target.value)} className={input} />
          </label>
          {editingAdmin && <p className="text-slate-500">{t('accounts.changeNotice')}</p>}
          <label className="block space-y-1.5">
            <span>{t('accounts.verifyPassword')}</span>
            <input type="password" required autoComplete="current-password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} className={input} />
          </label>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setAccountOpen(false)} disabled={busy} className="secondary-button">{t('common.cancel')}</button>
            <button type="submit" disabled={busy} className="primary-button">{t(busy ? 'common.saving' : 'common.save')}</button>
          </div>
        </form>
      </Modal>

      <Modal open={Boolean(revokeTarget)} onClose={() => !busy && setRevokeTarget(null)} title={t('accounts.revokeCurrent')}>
        <p className="text-sm text-slate-600 dark:text-slate-300">{t('accounts.revokeConfirm')}</p>
        <div className="flex justify-end gap-2 mt-5">
          <button type="button" disabled={busy} onClick={() => setRevokeTarget(null)} className="secondary-button">{t('common.cancel')}</button>
          <button type="button" disabled={busy} onClick={() => revokeTarget && void revoke(revokeTarget)} className="primary-button">{t('accounts.revoke')}</button>
        </div>
      </Modal>
    </div>
  )
}
