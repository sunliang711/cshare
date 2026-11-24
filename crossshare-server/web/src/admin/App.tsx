import { useCallback, useEffect, useRef, useState } from 'react'
import { api, getLastAuthedAt, setUnauthorizedHandler } from './api'
import { LoadingState } from './components/Feedback'
import { Layout, type NavTab } from './components/Layout'
import { ReauthModal } from './components/ReauthModal'
import { ToastProvider, useToast } from './components/Toast'
import { useI18n } from './contexts/I18nContext'
import { AuditPage } from './pages/AuditPage'
import { AuthPage } from './pages/AuthPages'
import { CronPage } from './pages/CronPage'
import { DashboardPage } from './pages/DashboardPage'
import { MigrationPage } from './pages/MigrationPage'
import { SettingsPage } from './pages/SettingsPage'
import type { Admin } from './types'

const SESSION_CHECK_INTERVAL_MS = 60_000

function initialTab(): NavTab {
  const p = window.location.pathname
  if (p.startsWith('/admin/cron')) return 'cron'
  if (p.startsWith('/admin/audit')) return 'audit'
  if (p.startsWith('/admin/migration')) return 'migration'
  if (p.startsWith('/admin/settings')) return 'settings'
  return 'dashboard'
}

interface AuthenticatedShellProps {
  admin: Admin
  version: string
  resetAdminEnvActive: boolean
  tab: NavTab
  onNavigate: (next: NavTab) => void
  onAdminUpdate: (admin: Admin, resetAdminEnvActive?: boolean) => void
  onSignedOut: () => void
}

function AuthenticatedShell({
  admin,
  version,
  resetAdminEnvActive,
  tab,
  onNavigate,
  onAdminUpdate,
  onSignedOut,
}: AuthenticatedShellProps) {
  const { t } = useI18n()
  const toast = useToast()
  const [reauthOpen, setReauthOpen] = useState(false)
  const reauthOpenRef = useRef(false)
  const reauthResolverRef = useRef<((ok: boolean) => void) | null>(null)

  const settleReauth = useCallback((ok: boolean) => {
    reauthOpenRef.current = false
    setReauthOpen(false)
    const resolve = reauthResolverRef.current
    reauthResolverRef.current = null
    resolve?.(ok)
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(
      () =>
        new Promise<boolean>((resolve) => {
          reauthResolverRef.current = resolve
          reauthOpenRef.current = true
          setReauthOpen(true)
        }),
    )

    const onVisibilityChange = () => {
      if (document.visibilityState !== 'visible') return
      if (reauthOpenRef.current) return
      if (Date.now() - getLastAuthedAt() < SESSION_CHECK_INTERVAL_MS) return
      api
        .me()
        .then((me) => onAdminUpdate(me.admin, me.reset_admin_env_active))
        .catch(() => {})
    }
    document.addEventListener('visibilitychange', onVisibilityChange)

    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange)
      setUnauthorizedHandler(null)
      settleReauth(false)
    }
  }, [onAdminUpdate, settleReauth])

  const handleLogout = useCallback(async () => {
    setUnauthorizedHandler(null)
    settleReauth(false)
    try {
      await api.logout()
    } catch {
      // ignore network error on logout
    }
    onSignedOut()
  }, [onSignedOut, settleReauth])

  return (
    <>
      <Layout
        active={tab}
        onNavigate={onNavigate}
        admin={admin}
        version={version}
        resetAdminEnvActive={resetAdminEnvActive}
        onLogout={handleLogout}
      >
        {tab === 'dashboard' && <DashboardPage onNavigate={onNavigate} />}
        {tab === 'cron' && <CronPage />}
        {tab === 'audit' && <AuditPage />}
        {tab === 'migration' && <MigrationPage />}
        {tab === 'settings' && <SettingsPage />}
      </Layout>

      <ReauthModal
        open={reauthOpen}
        defaultUsername={admin.username}
        onSuccess={(nextAdmin, nextResetActive) => {
          onAdminUpdate(nextAdmin, nextResetActive)
          settleReauth(true)
          toast.success(t('auth.reloginSuccess'))
        }}
        onCancel={() => settleReauth(false)}
        onSignOut={handleLogout}
      />
    </>
  )
}

export function App() {
  const { t } = useI18n()
  const [booting, setBooting] = useState(true)
  const [bootFailed, setBootFailed] = useState(false)
  const [initialized, setInitialized] = useState(true)
  const [admin, setAdmin] = useState<Admin | null>(null)
  const [version, setVersion] = useState('dev')
  const [resetAdminEnvActive, setResetAdminEnvActive] = useState(false)
  const [tab, setTab] = useState<NavTab>(initialTab)

  useEffect(() => {
    const onPopState = () => setTab(initialTab())
    window.addEventListener('popstate', onPopState)
    ;(async () => {
      try {
        const st = await api.setupStatus()
        setInitialized(st.initialized)
        if (st.version) setVersion(st.version)
        if (typeof st.reset_admin_env_active === 'boolean') {
          setResetAdminEnvActive(st.reset_admin_env_active)
        }
        if (st.initialized) {
          try {
            const me = await api.me()
            setAdmin(me.admin)
            if (me.version) setVersion(me.version)
            if (typeof me.reset_admin_env_active === 'boolean') {
              setResetAdminEnvActive(me.reset_admin_env_active)
            }
          } catch {
            setAdmin(null)
          }
        }
      } catch {
        setBootFailed(true)
      } finally {
        setBooting(false)
      }
    })()
    return () => window.removeEventListener('popstate', onPopState)
  }, [])

  const navigate = (next: NavTab) => {
    if (next === tab) return
    setTab(next)
    const path = next === 'dashboard' ? '/admin' : `/admin/${next}`
    window.history.pushState({}, '', path)
    window.scrollTo({ top: 0 })
  }

  const handleAdminUpdate = useCallback((nextAdmin: Admin, nextResetActive?: boolean) => {
    setAdmin(nextAdmin)
    if (typeof nextResetActive === 'boolean') {
      setResetAdminEnvActive(nextResetActive)
    }
  }, [])

  const handleSignedOut = useCallback(() => {
    setAdmin(null)
  }, [])

  if (booting) {
    return (
      <div className="min-h-screen grid place-items-center bg-[#f5f7fb] dark:bg-[#0b1120]">
        <LoadingState label={t('app.loading')} />
      </div>
    )
  }

  if (bootFailed) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4 p-6 text-center bg-[#f5f7fb] dark:bg-[#0b1120]">
        <p role="alert" className="text-sm text-slate-600 dark:text-slate-300">
          {t('app.connectError')}
        </p>
        <button className="primary-button" onClick={() => window.location.reload()}>
          {t('app.reconnect')}
        </button>
      </div>
    )
  }

  if (!initialized) {
    return (
      <AuthPage
        mode="setup"
        version={version}
        resetAdminEnvActive={resetAdminEnvActive}
        onSuccess={(a, nextResetActive) => {
          setInitialized(true)
          handleAdminUpdate(a, nextResetActive)
        }}
      />
    )
  }

  if (!admin) {
    return (
      <AuthPage
        mode="login"
        version={version}
        resetAdminEnvActive={resetAdminEnvActive}
        onSuccess={handleAdminUpdate}
      />
    )
  }

  if (admin.must_change_password) {
    return (
      <AuthPage
        mode="force-password"
        version={version}
        resetAdminEnvActive={resetAdminEnvActive}
        forcedAdmin={admin}
        onSuccess={handleAdminUpdate}
        onLogout={async () => {
          try {
            await api.logout()
          } catch {
            // ignore
          }
          handleSignedOut()
        }}
      />
    )
  }

  return (
    <ToastProvider>
      <AuthenticatedShell
        admin={admin}
        version={version}
        resetAdminEnvActive={resetAdminEnvActive}
        tab={tab}
        onNavigate={navigate}
        onAdminUpdate={handleAdminUpdate}
        onSignedOut={handleSignedOut}
      />
    </ToastProvider>
  )
}

export default App
