import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { en } from '../i18n/en'
import { zh, type TranslationKey } from '../i18n/zh'

export type Lang = 'zh' | 'en'

const STORAGE_KEY = 'cs_admin_lang'
const dictionaries: Record<Lang, Record<TranslationKey, string>> = { zh, en }

function resolveInitialLang(): Lang {
  const saved = localStorage.getItem(STORAGE_KEY) || localStorage.getItem('cs_lang')
  if (saved === 'en' || saved === 'zh') return saved
  return navigator.language?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

export function getCurrentLang(): Lang {
  return resolveInitialLang()
}

interface I18nContextValue {
  lang: Lang
  setLang: (lang: Lang) => void
  toggleLang: () => void
  t: (key: TranslationKey, vars?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nContextValue | null>(null)

export const I18nProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [lang, setLang] = useState<Lang>(resolveInitialLang)

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, lang)
    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en'
  }, [lang])

  const toggleLang = useCallback(() => {
    setLang((prev) => (prev === 'zh' ? 'en' : 'zh'))
  }, [])

  const t = useCallback(
    (key: TranslationKey, vars?: Record<string, string | number>): string => {
      let text = dictionaries[lang][key] ?? zh[key] ?? key
      if (vars) {
        for (const [k, v] of Object.entries(vars)) {
          text = text.replaceAll(`{${k}}`, String(v))
        }
      }
      return text
    },
    [lang],
  )

  return (
    <I18nContext.Provider value={{ lang, setLang, toggleLang, t }}>
      {children}
    </I18nContext.Provider>
  )
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}
