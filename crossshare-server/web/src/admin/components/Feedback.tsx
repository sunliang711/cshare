import React from 'react'
import { Loader2 } from 'lucide-react'

export const LoadingState: React.FC<{ label?: string }> = ({ label }) => (
  <div className="flex items-center justify-center gap-2 py-12 text-sm text-slate-500 dark:text-slate-400">
    <Loader2 className="w-4 h-4 animate-spin text-indigo-500" />
    {label && <span>{label}</span>}
  </div>
)

export const EmptyState: React.FC<{
  icon?: React.ReactNode
  title: string
  description?: string
}> = ({ icon, title, description }) => (
  <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
    {icon && (
      <div className="w-12 h-12 rounded-2xl bg-indigo-50 dark:bg-indigo-500/15 text-indigo-500 dark:text-indigo-400 flex items-center justify-center mb-3">
        {icon}
      </div>
    )}
    <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{title}</p>
    {description && <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm">{description}</p>}
  </div>
)
