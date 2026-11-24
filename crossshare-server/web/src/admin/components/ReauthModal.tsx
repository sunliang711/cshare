import React, { useEffect, useRef, useState } from 'react'
import { ArrowRight, Eye, EyeOff, Loader2, LockKeyhole, LogOut, ShieldAlert, X } from 'lucide-react'
import { api } from '../api'
import { useI18n } from '../contexts/I18nContext'
import type { Admin } from '../types'

interface ReauthModalProps {
  open: boolean
  defaultUsername: string
  onSuccess: (admin: Admin) => void
  onCancel: () => void
  onSignOut: () => void
}

export const ReauthModal: React.FC<ReauthModalProps> = ({
  open,
  defaultUsername,
  onSuccess,
  onCancel,
  onSignOut,
}) => {
  const { t } = useI18n()
  const [username, setUsername] = useState(defaultUsername)
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const usernameRef = useRef<HTMLInputElement | null>(null)
  const passwordRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (!open) return
    setUsername(defaultUsername)
    setPassword('')
    setShowPassword(false)
    setError('')
    setLoading(false)
    const timer = window.setTimeout(() => {
      if (defaultUsername) {
        passwordRef.current?.focus()
      } else {
        usernameRef.current?.focus()
      }
    }, 0)
    return () => window.clearTimeout(timer)
  }, [open, defaultUsername])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        // Stop propagation in capture phase so underlying Modal/Drawer stays open.
        e.preventDefault()
        e.stopPropagation()
        e.stopImmediatePropagation()
        onCancel()
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [open, onCancel])

  if (!open) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await api.login(username, password)
      onSuccess(res.admin)
    } catch (err: any) {
      setError(
        err?.message === 'invalid username or password'
          ? t('auth.invalidCredentials')
          : err?.message || t('auth.actionFailed'),
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-slate-900/55 backdrop-blur-[2px]" />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="reauth-modal-title"
        className="relative z-10 w-full max-w-md bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl overflow-hidden"
      >
        <div className="flex items-start justify-between gap-3 px-6 py-4 border-b border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-amber-50 dark:bg-amber-500/15 text-amber-600 dark:text-amber-400 flex items-center justify-center shrink-0">
              <ShieldAlert className="w-5 h-5" />
            </div>
            <div>
              <h3 id="reauth-modal-title" className="text-base font-semibold text-slate-900 dark:text-slate-100">
                {t('auth.sessionExpiredTitle')}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                {t('auth.sessionExpiredDesc')}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onCancel}
            aria-label={t('common.close')}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 shrink-0"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div
              role="alert"
              className="p-3 rounded-lg bg-rose-50 dark:bg-rose-500/15 border border-rose-200 dark:border-rose-500/30 text-rose-700 dark:text-rose-300 text-xs"
            >
              {error}
            </div>
          )}

          <div>
            <label htmlFor="reauth-username" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('auth.username')}
            </label>
            <input
              ref={usernameRef}
              id="reauth-username"
              autoComplete="username"
              type="text"
              required
              minLength={3}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('auth.usernamePlaceholder')}
              className="w-full px-3.5 py-2 border rounded-xl text-sm"
            />
          </div>

          <div>
            <label htmlFor="reauth-password" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('auth.password')}
            </label>
            <div className="relative">
              <input
                ref={passwordRef}
                id="reauth-password"
                autoComplete="current-password"
                type={showPassword ? 'text' : 'password'}
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('auth.passwordPlaceholder')}
                className="w-full px-3.5 py-2 pr-10 border rounded-xl text-sm"
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? t('secret.hide') : t('secret.show')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2 pt-2">
            <button
              type="button"
              onClick={onSignOut}
              disabled={loading}
              className="secondary-button"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span>{t('auth.backToLogin')}</span>
            </button>
            <button type="submit" disabled={loading} className="primary-button">
              {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <LockKeyhole className="w-4 h-4" />}
              <span>{t('auth.reloginSubmit')}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
