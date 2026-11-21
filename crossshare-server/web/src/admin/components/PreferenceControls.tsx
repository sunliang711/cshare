import React from 'react'
import { Languages, Moon, Sun } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'
import { useI18n } from '../contexts/I18nContext'

export const PreferenceControls: React.FC<{ className?: string }> = ({ className = '' }) => {
  const { theme, toggleTheme } = useTheme()
  const { t, toggleLang } = useI18n()

  const themeLabel = theme === 'dark' ? t('pref.switchToLight') : t('pref.switchToDark')

  return (
    <div className={`inline-flex items-center gap-1.5 ${className}`}>
      <button
        type="button"
        onClick={toggleTheme}
        title={themeLabel}
        aria-label={themeLabel}
        className="inline-flex items-center justify-center h-8 w-8 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/80 text-slate-600 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 transition"
      >
        {theme === 'dark' ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4" />}
      </button>
      <button
        type="button"
        onClick={toggleLang}
        title={t('pref.switchLang')}
        aria-label={t('pref.switchLang')}
        className="inline-flex items-center gap-1 h-8 px-2.5 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800/80 text-xs font-medium text-slate-600 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 transition"
      >
        <Languages className="w-3.5 h-3.5" />
        <span>{t('pref.langShort')}</span>
      </button>
    </div>
  )
}
