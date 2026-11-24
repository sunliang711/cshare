import React, { createContext, useCallback, useContext, useMemo, useState } from 'react'
import { CheckCircle2, Info, X, XCircle } from 'lucide-react'

type ToastKind = 'success' | 'error' | 'info'

interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

interface ToastContextValue {
  success: (msg: string) => void
  error: (msg: string) => void
  info: (msg: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [items, setItems] = useState<ToastItem[]>([])

  const push = useCallback((kind: ToastKind, message: string) => {
    const id = Date.now() + Math.floor(Math.random() * 1000)
    setItems((prev) => [...prev, { id, kind, message }])
    setTimeout(() => {
      setItems((prev) => prev.filter((item) => item.id !== id))
    }, 3500)
  }, [])

  const value = useMemo<ToastContextValue>(
    () => ({
      success: (m) => push('success', m),
      error: (m) => push('error', m),
      info: (m) => push('info', m),
    }),
    [push],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed bottom-5 right-5 z-[80] flex flex-col gap-2 max-w-sm">
        {items.map((item) => (
          <div
            key={item.id}
            role="status"
            className={`flex items-start gap-2.5 px-4 py-3 rounded-xl border shadow-lg text-xs font-medium bg-white dark:bg-slate-900 ${
              item.kind === 'error'
                ? 'border-rose-200 dark:border-rose-500/40 text-rose-700 dark:text-rose-300'
                : item.kind === 'success'
                ? 'border-emerald-200 dark:border-emerald-500/40 text-emerald-700 dark:text-emerald-300'
                : 'border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-200'
            }`}
          >
            {item.kind === 'success' && <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5 text-emerald-500" />}
            {item.kind === 'error' && <XCircle className="w-4 h-4 shrink-0 mt-0.5 text-rose-500" />}
            {item.kind === 'info' && <Info className="w-4 h-4 shrink-0 mt-0.5 text-indigo-500" />}
            <span className="flex-1 break-words">{item.message}</span>
            <button
              type="button"
              onClick={() => setItems((prev) => prev.filter((it) => it.id !== item.id))}
              className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}
