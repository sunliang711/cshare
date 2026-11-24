import React from 'react'
import { AlertTriangle } from 'lucide-react'
import { useI18n } from '../contexts/I18nContext'

interface Props {
  active?: boolean
  className?: string
}

export const ResetAdminBanner: React.FC<Props> = ({ active, className = '' }) => {
  const { t } = useI18n()
  if (!active) return null

  return (
    <div
      role="status"
      className={`flex items-start gap-2.5 px-4 py-3 rounded-xl border border-amber-300/80 dark:border-amber-500/40 bg-amber-50 dark:bg-amber-500/15 text-amber-900 dark:text-amber-200 text-xs leading-5 ${className}`}
    >
      <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
      <span>{t('auth.resetAdminBanner')}</span>
    </div>
  )
}
