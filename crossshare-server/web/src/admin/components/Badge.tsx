import React from 'react'
import { useI18n } from '../contexts/I18nContext'

type Tone = 'emerald' | 'rose' | 'amber' | 'indigo' | 'slate'

const toneClasses: Record<Tone, string> = {
  emerald: 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:ring-emerald-500/30',
  rose: 'bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-500/15 dark:text-rose-300 dark:ring-rose-500/30',
  amber: 'bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-500/30',
  indigo: 'bg-indigo-50 text-indigo-700 ring-indigo-200 dark:bg-indigo-500/15 dark:text-indigo-300 dark:ring-indigo-500/30',
  slate: 'bg-slate-100 text-slate-600 ring-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:ring-slate-700',
}

export const Badge: React.FC<{ tone?: Tone; children: React.ReactNode }> = ({
  tone = 'slate',
  children,
}) => (
  <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium ring-1 ring-inset ${toneClasses[tone]}`}>
    {children}
  </span>
)

export const StatusBadge: React.FC<{ status?: string }> = ({ status }) => {
  const { t } = useI18n()
  switch (status) {
    case 'succeeded':
    case 'success':
      return <Badge tone="emerald">{t('audit.statusSuccess')}</Badge>
    case 'failed':
    case 'failure':
      return <Badge tone="rose">{t('audit.statusFailure')}</Badge>
    case 'running':
      return <Badge tone="indigo">running</Badge>
    case 'enabled':
      return <Badge tone="emerald">{t('common.enabled')}</Badge>
    case 'disabled':
      return <Badge tone="slate">{t('common.disabled')}</Badge>
    default:
      return <Badge tone="slate">{status || '—'}</Badge>
  }
}
