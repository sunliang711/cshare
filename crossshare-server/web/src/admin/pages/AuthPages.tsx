import React, { useState } from 'react'
import { ArrowRight, Eye, EyeOff, Loader2, LockKeyhole, LogOut, Share2, ShieldCheck } from 'lucide-react'
import { api } from '../api'
import { PreferenceControls } from '../components/PreferenceControls'
import { ResetAdminBanner } from '../components/ResetAdminBanner'
import { useI18n } from '../contexts/I18nContext'
import type { Admin } from '../types'

interface Props {
  mode: 'setup' | 'login' | 'force-password'
  version?: string
  resetAdminEnvActive?: boolean
  forcedAdmin?: Admin | null
  onSuccess: (admin: Admin, resetAdminEnvActive?: boolean) => void
  onLogout?: () => void
}

export const AuthPage: React.FC<Props> = ({
  mode,
  version = 'dev',
  resetAdminEnvActive = false,
  forcedAdmin = null,
  onSuccess,
  onLogout,
}) => {
  const { t } = useI18n()
  const versionLabel = version && version !== 'dev' && !version.startsWith('v') ? `v${version}` : version || 'dev'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if ((mode === 'setup' || mode === 'force-password') && password !== confirm) {
      setError(t('auth.passwordMismatch'))
      return
    }
    setLoading(true)
    try {
      if (mode === 'force-password') {
        const res = await api.changePassword('', password)
        const nextAdmin: Admin = res.admin || {
          ...(forcedAdmin as Admin),
          must_change_password: false,
        }
        setPassword('')
        setConfirm('')
        onSuccess(nextAdmin, res.reset_admin_env_active)
        return
      }
      const res =
        mode === 'setup' ? await api.setupInit(username, password) : await api.login(username, password)
      setPassword('')
      setConfirm('')
      onSuccess(res.admin, res.reset_admin_env_active)
    } catch (err: any) {
      setError(
        err.message === 'invalid username or password'
          ? t('auth.invalidCredentials')
          : err.message || t('auth.actionFailed'),
      )
    } finally {
      setLoading(false)
    }
  }

  const title =
    mode === 'setup'
      ? t('auth.setupTitle')
      : mode === 'force-password'
        ? t('auth.forcePasswordTitle')
        : t('auth.loginTitle')

  const subtitle =
    mode === 'setup'
      ? t('auth.setupSubtitle')
      : mode === 'force-password'
        ? t('auth.forcePasswordSubtitle', { username: forcedAdmin?.username || 'admin' })
        : t('auth.loginSubtitle')

  const submitLabel =
    mode === 'setup'
      ? t('auth.setupSubmit')
      : mode === 'force-password'
        ? t('auth.forcePasswordSubmit')
        : t('auth.loginSubmit')

  return (
    <div className="min-h-screen grid lg:grid-cols-2 bg-[#f7f9fc] dark:bg-[#0b1120] transition-colors">
      <div className="hidden lg:flex flex-col justify-between relative overflow-hidden bg-[#111b34] dark:bg-[#090e1a] border-r border-transparent dark:border-slate-800 text-white p-12 xl:p-16">
        <div className="absolute -right-32 top-1/4 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl" />
        <div className="flex items-center gap-3 font-semibold text-2xl">
          <img src="/favicon.svg?v=2" alt="" className="w-8 h-8 rounded-lg" />
          CrossShare<span className="text-indigo-400">.</span>
        </div>
        <div className="relative py-16 max-w-lg">
          <div className="w-16 h-16 bg-indigo-400/10 border border-indigo-400/20 rounded-2xl grid place-items-center mb-8">
            <Share2 className="w-8 h-8 text-indigo-300" />
          </div>
          <div className="text-[40px] xl:text-5xl font-semibold leading-tight tracking-tight">
            {t('auth.heroLine1')}
            <br />
            <span className="text-indigo-300">{t('auth.heroLine2')}</span>
          </div>
          <p className="text-slate-400 text-sm leading-7 mt-6">{t('auth.heroDesc')}</p>
          <div className="flex gap-5 text-xs text-slate-300 mt-8">
            <span>{t('auth.feature1')}</span>
            <span className="text-slate-600">/</span>
            <span>{t('auth.feature2')}</span>
            <span className="text-slate-600">/</span>
            <span>{t('auth.feature3')}</span>
          </div>
        </div>
        <div className="text-xs text-slate-500">
          {t('auth.footerBrand')} · <span className="font-mono">{versionLabel}</span>
        </div>
      </div>

      <div className="relative flex flex-col items-center justify-center p-6 sm:p-12">
        <div className="absolute top-5 right-5 sm:top-6 sm:right-6 flex items-center gap-2">
          <PreferenceControls />
          <a
            href="/"
            title={t('nav.toClient')}
            className="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/80 text-xs font-medium text-slate-600 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 transition"
          >
            <Share2 className="w-3.5 h-3.5 text-indigo-500 dark:text-indigo-400" />
            <span>{t('nav.toClient')}</span>
          </a>
        </div>

        <ResetAdminBanner active={resetAdminEnvActive} className="w-full max-w-md mb-4" />

        <div className="w-full max-w-md bg-white dark:bg-slate-900 rounded-2xl border border-slate-200/80 dark:border-slate-800 shadow-xl shadow-slate-200/30 dark:shadow-black/40 p-6 sm:p-9">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-xl bg-indigo-600 text-white flex items-center justify-center shrink-0">
              <ShieldCheck className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900 dark:text-slate-100">{title}</h1>
              <p className="text-xs text-slate-500 dark:text-slate-400">{subtitle}</p>
            </div>
          </div>

          {error && (
            <div
              role="alert"
              className="mb-4 p-3 rounded-lg bg-rose-50 dark:bg-rose-500/15 border border-rose-200 dark:border-rose-500/30 text-rose-700 dark:text-rose-300 text-sm"
            >
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {mode !== 'force-password' && (
              <div>
                <label htmlFor="auth-username" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('auth.username')}
                </label>
                <input
                  id="auth-username"
                  autoComplete="username"
                  autoFocus
                  type="text"
                  required
                  minLength={3}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder={t('auth.usernamePlaceholder')}
                  className="w-full px-3.5 py-2.5 border rounded-xl text-sm"
                />
              </div>
            )}

            <div>
              <label htmlFor="auth-password" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {mode === 'force-password' ? t('auth.newPassword') : t('auth.password')}
              </label>
              <div className="relative">
                <input
                  id="auth-password"
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  autoFocus={mode === 'force-password'}
                  type={showPassword ? 'text' : 'password'}
                  required
                  minLength={8}
                  maxLength={72}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('auth.passwordPlaceholder')}
                  className="w-full px-3.5 py-2.5 pr-10 border rounded-xl text-sm"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {(mode === 'setup' || mode === 'force-password') && (
              <div>
                <label htmlFor="auth-confirm" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  {t('auth.confirmPassword')}
                </label>
                <input
                  id="auth-confirm"
                  autoComplete="new-password"
                  type="password"
                  required
                  minLength={8}
                  maxLength={72}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  placeholder={t('auth.confirmPasswordPlaceholder')}
                  className="w-full px-3.5 py-2.5 border rounded-xl text-sm"
                />
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full primary-button py-2.5 mt-2"
            >
              {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <LockKeyhole className="w-4 h-4" />}
              <span>{submitLabel}</span>
              {!loading && <ArrowRight className="w-4 h-4" />}
            </button>
          </form>

          {mode === 'login' && (
            <details className="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800 text-xs text-slate-500 dark:text-slate-400">
              <summary className="cursor-pointer select-none font-medium text-slate-600 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 transition">
                {t('auth.forgotTitle')}
              </summary>
              <p className="mt-2 leading-5">{t('auth.forgotDesc')}</p>
            </details>
          )}

          {mode === 'force-password' && onLogout && (
            <button
              type="button"
              onClick={onLogout}
              className="w-full mt-3 inline-flex items-center justify-center gap-1.5 py-2 text-xs font-medium text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 transition"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span>{t('auth.backToLogin')}</span>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
