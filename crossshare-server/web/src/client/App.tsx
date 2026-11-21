import React, { useCallback, useEffect, useRef, useState } from 'react'
import { copyText } from '../shared/clipboard'
import { clientI18n, type ClientI18nKey, type ClientLang } from './i18n'
import { renderQRCode } from './qrcode-lite'
import type { P2PState, P2PStatusState, PushInput, ServerPushResult } from './types'

declare global {
  interface Window {
    __CROSSSHARE_SERVER__?: string
  }
}

const P2P_ICE_SERVERS: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun.cloudflare.com:3478' },
]
const P2P_LAN_FALLBACK_DELAY = 2000
const P2P_SLOW_NOTICE_DELAY = 5000
const P2P_CONNECT_TIMEOUT = 10000
const P2P_CHUNK_SIZE = 64 * 1024
const P2P_BUFFER_LIMIT = 1 << 20
const P2P_POLL_WAIT_SECONDS = 25

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KiB', 'MiB', 'GiB']
  let i = -1
  let b = bytes
  do {
    b /= 1024
    i++
  } while (b >= 1024 && i < units.length - 1)
  return `${b.toFixed(1)} ${units[i]}`
}

function humanDuration(sec: number): string {
  if (sec < 60) return `${sec}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m${sec % 60}s`
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return m ? `${h}h${m}m` : `${h}h`
}

function fileSignature(file: File): string {
  return `${file.name}\n${file.size}\n${file.lastModified}`
}

function defaultServerUrl(): string {
  return window.__CROSSSHARE_SERVER__ || window.location.origin
}

function buildShareUrl(key: string): string {
  const url = new URL(window.location.href)
  url.search = ''
  url.hash = ''
  url.searchParams.set('key', key)
  return url.toString()
}

function buildP2PUrl(sessionID: string): string {
  const url = new URL(window.location.href)
  url.search = ''
  url.hash = ''
  url.searchParams.set('p2p', sessionID)
  return url.toString()
}

function supportsP2P(): boolean {
  return Boolean(
    window.RTCPeerConnection &&
      window.RTCSessionDescription &&
      window.RTCIceCandidate,
  )
}

export function App() {
  const [lang, setLang] = useState<ClientLang>(
    () => (localStorage.getItem('cs_lang') as ClientLang) || 'zh',
  )
  const [theme, setTheme] = useState<'light' | 'dark'>(
    () => (localStorage.getItem('cs_theme') === 'dark' ? 'dark' : 'light'),
  )
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [serverUrl, setServerUrl] = useState(
    () => localStorage.getItem('cs_server') || defaultServerUrl(),
  )
  const [authToken, setAuthToken] = useState(
    () => localStorage.getItem('cs_token') || '',
  )
  const [healthStatus, setHealthStatus] = useState<{ text: string; color: string }>({
    text: '',
    color: '',
  })

  const [activeTab, setActiveTab] = useState<'push' | 'pull'>('push')
  const [pushMode, setPushMode] = useState<'text' | 'file'>('text')
  const [pushText, setPushText] = useState('')
  const [pushTtl, setPushTtl] = useState('')
  const [pushMoreOpen, setPushMoreOpen] = useState(false)
  const [pullMoreOpen, setPullMoreOpen] = useState(false)

  const [transferMode, setTransferMode] = useState<'smart' | 'server'>(() =>
    localStorage.getItem('cs_transfer') === 'server' ? 'server' : 'smart',
  )
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [dragOver, setDragOver] = useState(false)
  const [pushBusy, setPushBusy] = useState(false)
  const [sendCardCollapsed, setSendCardCollapsed] = useState(false)
  const sendCardManualExpandedRef = useRef(false)

  // Push result
  const [pushResultVisible, setPushResultVisible] = useState(false)
  const [pushResultMode, setPushResultMode] = useState<'' | 'server' | 'p2p' | 'p2pDone'>('')
  const [resultKeyLabel, setResultKeyLabel] = useState('')
  const [currentResult, setCurrentResult] = useState<{ mode: string; key: string; url: string }>({
    mode: '',
    key: '',
    url: '',
  })
  const [serverResultMeta, setServerResultMeta] = useState('')
  const [countdownExpireAt, setCountdownExpireAt] = useState<number | null>(null)
  const [countdownTick, setCountdownTick] = useState(0)
  const [pushP2PStatus, setPushP2PStatus] = useState<P2PStatusState | null>(null)

  // Pull state
  const [pullKey, setPullKey] = useState('')
  const [deleteAfterPull, setDeleteAfterPull] = useState(false)
  const [pullBusy, setPullBusy] = useState(false)
  const [pullResultVisible, setPullResultVisible] = useState(false)
  const [pullTextContent, setPullTextContent] = useState<string | null>(null)
  const [pullFile, setPullFile] = useState<{ name: string; url: string } | null>(null)
  const [pullMetaText, setPullMetaText] = useState('')
  const [pullP2PStatus, setPullP2PStatus] = useState<P2PStatusState | null>(null)

  // Delete state
  const [deleteKey, setDeleteKey] = useState('')
  const [deleteBusy, setDeleteBusy] = useState(false)
  const [deleteMeta, setDeleteMeta] = useState('')

  // QR modal
  const [qrModalUrl, setQrModalUrl] = useState<string | null>(null)

  // Toast
  const [toastState, setToastState] = useState<{ msg: string; type: '' | 'success' | 'error'; visible: boolean }>({
    msg: '',
    type: '',
    visible: false,
  })
  const toastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const settingsRef = useRef<HTMLDivElement | null>(null)
  const settingsBtnRef = useRef<HTMLButtonElement | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const qrCanvasRef = useRef<HTMLCanvasElement | null>(null)
  const p2pStateRef = useRef<P2PState | null>(null)
  const pullFileUrlRef = useRef<string | null>(null)
  const langRef = useRef(lang)
  langRef.current = lang
  const currentResultRef = useRef(currentResult)
  currentResultRef.current = currentResult

  const t = useCallback(
    (key: ClientI18nKey): string => clientI18n[lang][key] ?? key,
    [lang],
  )
  const tLatest = useCallback(
    (key: ClientI18nKey): string => clientI18n[langRef.current][key] ?? key,
    [],
  )

  const showToast = useCallback((msg: string, type: '' | 'success' | 'error' = '') => {
    setToastState({ msg, type, visible: true })
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current)
    toastTimerRef.current = setTimeout(() => {
      setToastState((prev) => ({ ...prev, visible: false }))
    }, 2500)
  }, [])

  // Theme sync
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('cs_theme', theme)
    const metaColor = document.getElementById('metaThemeColor') as HTMLMetaElement | null
    if (metaColor) {
      metaColor.content = theme === 'dark' ? '#0b1020' : '#eef2fb'
    }
  }, [theme])

  // Lang sync
  useEffect(() => {
    localStorage.setItem('cs_lang', lang)
    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en'
  }, [lang])

  // Countdown tick
  useEffect(() => {
    if (!countdownExpireAt) return
    const timer = setInterval(() => {
      setCountdownTick((n) => n + 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [countdownExpireAt])

  // Cleanup object URL
  useEffect(() => {
    return () => {
      if (pullFileUrlRef.current) URL.revokeObjectURL(pullFileUrlRef.current)
    }
  }, [])

  // Close settings panel on outside pointerdown
  useEffect(() => {
    const onPointerDown = (e: PointerEvent) => {
      if (!settingsOpen) return
      const target = e.target as Node
      if (settingsRef.current?.contains(target) || settingsBtnRef.current?.contains(target)) {
        return
      }
      setSettingsOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [settingsOpen])

  // QR rendering when modal opens
  useEffect(() => {
    if (!qrModalUrl || !qrCanvasRef.current) return
    try {
      renderQRCode(qrCanvasRef.current, qrModalUrl, {
        size: 300,
        errorCorrectionLevel: 'M',
      })
    } catch (e: any) {
      showToast(`${t('qrFail')}: ${e?.message || ''}`, 'error')
      setQrModalUrl(null)
    }
  }, [qrModalUrl, showToast, t])

  const getApiUrl = useCallback((path: string) => {
    const base = (localStorage.getItem('cs_server') || defaultServerUrl()).replace(/\/+$/, '')
    return `${base}/api/v1${path}`
  }, [])

  const getAuthHeaders = useCallback((): Record<string, string> => {
    const tk = localStorage.getItem('cs_token') || ''
    return tk ? { Authorization: `Bearer ${tk}` } : {}
  }, [])

  const collapseSendCard = useCallback((collapsed: boolean, manual: boolean) => {
    if (manual) sendCardManualExpandedRef.current = !collapsed
    setSendCardCollapsed(collapsed)
  }, [])

  const autoCollapseSendCard = useCallback(() => {
    if (sendCardManualExpandedRef.current) return
    setSendCardCollapsed(true)
  }, [])

  const resetSendCardAutoState = useCallback((expand: boolean) => {
    sendCardManualExpandedRef.current = false
    if (expand) setSendCardCollapsed(false)
  }, [])

  // --- P2P Helpers ---

  const closeP2PConnection = (state: P2PState) => {
    if (state.dc) {
      state.dc.onopen = null
      state.dc.onerror = null
      state.dc.close()
      state.dc = null
    }
    if (state.pc) {
      state.pc.onicecandidate = null
      state.pc.onconnectionstatechange = null
      state.pc.ondatachannel = null
      state.pc.close()
      state.pc = null
    }
  }

  const clearP2PTimers = (state: P2PState) => {
    if (state.connectTimer) clearTimeout(state.connectTimer)
    if (state.upgradeTimer) clearTimeout(state.upgradeTimer)
    if (state.slowTimer) clearTimeout(state.slowTimer)
    state.connectTimer = null
    state.upgradeTimer = null
    state.slowTimer = null
  }

  const stopP2PStatusTimer = (state: P2PState) => {
    if (state.statusTimer) clearInterval(state.statusTimer)
    state.statusTimer = null
  }

  const stopP2PPoll = (state: P2PState) => {
    if (state.pollController) {
      state.pollController.abort()
      state.pollController = null
    }
  }

  const stopP2PTransport = (state: P2PState) => {
    clearP2PTimers(state)
    stopP2PStatusTimer(state)
    stopP2PPoll(state)
    closeP2PConnection(state)
  }

  const renderP2PStatus = useCallback(
    (
      target: 'push' | 'pull',
      message: string,
      active: boolean,
      detail: string,
      startedAt: number,
      diagnostic: string,
    ) => {
      const payload: P2PStatusState = {
        message,
        active,
        detail,
        startedAt,
        diagnostic,
      }
      if (target === 'pull') {
        setPullP2PStatus(payload)
      } else {
        setPushP2PStatus(payload)
      }
    },
    [],
  )

  const updateP2PStatus = useCallback(
    (target: 'push' | 'pull', message: string, active: boolean, detail: string) => {
      const state = p2pStateRef.current
      if (state) {
        state.statusTarget = target
        state.statusMessage = message
        state.statusActive = active
        state.statusDetail = detail
      }
      renderP2PStatus(
        target,
        message,
        active,
        detail,
        state?.startedAt || Date.now(),
        state?.p2pDiagnostic || '',
      )
      if (target === 'push' && active) {
        setPushResultVisible(true)
        autoCollapseSendCard()
      }
      if (!state) return

      if (active && !state.statusTimer) {
        state.statusTimer = setInterval(() => {
          if (p2pStateRef.current !== state || !state.statusActive) {
            stopP2PStatusTimer(state)
            return
          }
          renderP2PStatus(
            state.statusTarget || target,
            state.statusMessage || message,
            Boolean(state.statusActive),
            state.statusDetail || '',
            state.startedAt,
            state.p2pDiagnostic,
          )
        }, 500)
      }
      if (!active) {
        stopP2PStatusTimer(state)
      }
    },
    [autoCollapseSendCard, renderP2PStatus],
  )

  const cancelP2P = useCallback(
    (showToastMsg: boolean) => {
      const state = p2pStateRef.current
      if (!state) return
      state.stopped = true
      stopP2PTransport(state)
      if (state.sessionID) {
        fetch(getApiUrl(`/p2p/sessions/${state.sessionID}`), {
          method: 'DELETE',
          headers: getAuthHeaders(),
        }).catch(() => {})
      }
      p2pStateRef.current = null
      if (showToastMsg) {
        setPushP2PStatus(null)
        setServerResultMeta(tLatest('p2pCancelled'))
        showToast(tLatest('p2pCancelled'), 'success')
      }
    },
    [getApiUrl, getAuthHeaders, showToast, tLatest],
  )

  const postP2PMessage = useCallback(
    async (sessionID: string, from: string, to: string, type: string, payload: unknown) => {
      const resp = await fetch(getApiUrl(`/p2p/sessions/${sessionID}/messages`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders(),
        },
        body: JSON.stringify({ from, to, type, payload }),
      })
      const data = await resp.json()
      if (data.code !== 0) {
        throw new Error(data.msg || 'p2p signal failed')
      }
      return data.data
    },
    [getApiUrl, getAuthHeaders],
  )

  const uploadToServer = useCallback(
    async (input: PushInput): Promise<ServerPushResult> => {
      let resp: Response
      if (input.type === 'text') {
        const body: Record<string, unknown> = { text: input.text }
        if (input.ttl > 0) body.ttl = input.ttl
        resp = await fetch(getApiUrl('/push/text'), {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...getAuthHeaders(),
          },
          body: JSON.stringify(body),
        })
      } else {
        const body = new FormData()
        for (const file of input.files || (input.file ? [input.file] : [])) {
          body.append('files', file, file.name)
        }
        if (input.ttl > 0) body.append('ttl', String(input.ttl))
        if (input.type === 'files' && input.filename) {
          body.append('name', input.filename)
        }
        resp = await fetch(getApiUrl('/push/files'), {
          method: 'POST',
          headers: getAuthHeaders(),
          body,
        })
      }
      const data = await resp.json()
      if (data.code !== 0) {
        throw new Error(data.msg || tLatest('pushFail'))
      }
      return data.data
    },
    [getApiUrl, getAuthHeaders, tLatest],
  )

  const renderServerPushResult = useCallback(
    (r: ServerPushResult) => {
      const res = { mode: 'server', key: r.key, url: buildShareUrl(r.key) }
      setCurrentResult(res)
      currentResultRef.current = res
      setResultKeyLabel(r.key)
      setPushP2PStatus(null)
      let meta = `${tLatest('metaType')}: ${r.type} · ${tLatest('metaSize')}: ${humanSize(r.size)} · TTL: ${humanDuration(r.ttl)}`
      if (r.filename) meta += ` · ${tLatest('metaFile')}: ${r.filename}`
      if (r.file_count) meta += ` · ${tLatest('metaFiles')}: ${r.file_count}`
      if (r.stored_size && r.stored_size !== r.size) {
        meta += ` · ${tLatest('metaStored')}: ${humanSize(r.stored_size)}`
      }
      setServerResultMeta(meta)
      setPushResultMode('server')
      autoCollapseSendCard()
      setPushResultVisible(true)
      setCountdownExpireAt(r.expire_at || Math.floor(Date.now() / 1000) + r.ttl)
    },
    [autoCollapseSendCard, tLatest],
  )

  const getP2PDiagnostic = useCallback(async (pc: RTCPeerConnection): Promise<string> => {
    if (!pc.getStats) return ''
    const stats = await pc.getStats()
    let selectedPair: any = null
    stats.forEach((report: any) => {
      if (report.type === 'transport' && report.selectedCandidatePairId) {
        selectedPair = stats.get(report.selectedCandidatePairId)
      }
    })
    if (!selectedPair) {
      let succeededPair: any = null
      let succeededPairBytes = -1
      stats.forEach((report: any) => {
        if (
          report.type === 'candidate-pair' &&
          (report.selected || (report.nominated && report.state === 'succeeded'))
        ) {
          selectedPair = report
        }
        if (report.type === 'candidate-pair' && report.state === 'succeeded') {
          const bytes = (report.bytesSent || 0) + (report.bytesReceived || 0)
          if (bytes > succeededPairBytes) {
            succeededPair = report
            succeededPairBytes = bytes
          }
        }
      })
      if (!selectedPair) selectedPair = succeededPair
    }
    if (!selectedPair) return ''
    const local: any = stats.get(selectedPair.localCandidateId)
    const remote: any = stats.get(selectedPair.remoteCandidateId)
    if (!local || !remote) return ''

    const localType = local.candidateType || ''
    const remoteType = remote.candidateType || ''
    const route =
      localType === 'relay' || remoteType === 'relay'
        ? tLatest('p2pRouteRelay')
        : localType === 'host' && remoteType === 'host'
        ? tLatest('p2pRouteLan')
        : tLatest('p2pRouteNat')
    const candLabel = (tp: string) => {
      const labels =
        langRef.current === 'zh'
          ? { host: '主机', srflx: '公网映射', prflx: '对端反射', relay: '中继' }
          : { host: 'host', srflx: 'srflx', prflx: 'prflx', relay: 'relay' }
      return (labels as Record<string, string>)[tp] || tp || '-'
    }
    const protocol = local.protocol || remote.protocol || ''
    const parts = [
      `${tLatest('p2pRouteLabel')}: ${route}`,
      `${tLatest('p2pCandidateLocal')}: ${candLabel(localType)}`,
      `${tLatest('p2pCandidateRemote')}: ${candLabel(remoteType)}`,
    ]
    if (protocol) parts.push(protocol.toUpperCase())
    return parts.join(' · ')
  }, [tLatest])

  const updateP2PDiagnostic = useCallback(
    async (state: P2PState | null, target: 'push' | 'pull') => {
      if (!state || !state.pc || p2pStateRef.current !== state) return
      const diagnostic = await getP2PDiagnostic(state.pc)
      if (!diagnostic || p2pStateRef.current !== state) return
      state.p2pDiagnostic = diagnostic
      if (
        state.role === 'sender' &&
        !state.p2pDiagnosticSent &&
        state.dc &&
        state.dc.readyState === 'open'
      ) {
        state.p2pDiagnosticSent = true
        try {
          state.dc.send(JSON.stringify({ kind: 'diagnostic', diagnostic }))
        } catch {
          // ignore
        }
      }
      renderP2PStatus(
        state.statusTarget || target,
        state.statusMessage || tLatest('p2pConnected'),
        state.statusActive !== false,
        state.statusDetail || '',
        state.startedAt,
        state.p2pDiagnostic,
      )
    },
    [getP2PDiagnostic, renderP2PStatus, tLatest],
  )

  const scheduleP2PDiagnostic = useCallback(
    (state: P2PState, target: 'push' | 'pull') => {
      updateP2PDiagnostic(state, target).catch(() => {})
      setTimeout(() => updateP2PDiagnostic(state, target).catch(() => {}), 500)
      setTimeout(() => updateP2PDiagnostic(state, target).catch(() => {}), 1500)
    },
    [updateP2PDiagnostic],
  )

  const fallbackP2PToServer = useCallback(async () => {
    const state = p2pStateRef.current
    if (
      !state ||
      state.role !== 'sender' ||
      !state.input ||
      state.connected ||
      state.transferStarted ||
      state.transferDone ||
      state.fallbackStarted
    ) {
      return
    }
    state.fallbackStarted = true
    stopP2PTransport(state)
    state.stopped = true
    updateP2PStatus('push', tLatest('p2pFallback'), true, currentResultRef.current.url)

    try {
      const result = await uploadToServer(state.input)
      await postP2PMessage(state.sessionID, 'sender', 'receiver', 'fallback', {
        key: result.key,
      }).catch(() => {})
      renderServerPushResult(result)
      p2pStateRef.current = null
      showToast(tLatest('pushOk'), 'success')
    } catch (e: any) {
      state.fallbackStarted = false
      resetSendCardAutoState(true)
      updateP2PStatus('push', tLatest('p2pFailed'), false, currentResultRef.current.url)
      showToast(`${tLatest('pushFail')}: ${e?.message || ''}`, 'error')
    }
  }, [postP2PMessage, renderServerPushResult, resetSendCardAutoState, showToast, tLatest, updateP2PStatus, uploadToServer])

  const markP2PFailed = useCallback(() => {
    const state = p2pStateRef.current
    if (!state || state.transferDone) return
    resetSendCardAutoState(true)
    updateP2PStatus('push', tLatest('p2pFailed'), false, currentResultRef.current.url)
    if (state.transferStarted || state.connected) {
      showToast(tLatest('p2pFailed'), 'error')
      return
    }
    fallbackP2PToServer()
  }, [fallbackP2PToServer, resetSendCardAutoState, showToast, tLatest, updateP2PStatus])

  const sendP2PContent = useCallback(
    async (input: PushInput) => {
      const state = p2pStateRef.current
      if (!state || !state.dc || state.stopped) return
      const dc = state.dc
      state.connected = true
      state.transferStarted = true
      clearP2PTimers(state)
      updateP2PStatus('push', tLatest('p2pSending'), true, currentResultRef.current.url)
      dc.send(
        JSON.stringify({
          kind: 'meta',
          type: input.type,
          filename: input.filename || '',
          content_type: input.contentType,
          size: input.size,
        }),
      )

      const blob =
        input.type === 'text'
          ? new Blob([input.text || ''], { type: input.contentType })
          : input.file!
      for (let offset = 0; offset < blob.size; offset += P2P_CHUNK_SIZE) {
        if (!p2pStateRef.current || p2pStateRef.current.stopped) return
        if (dc.bufferedAmount >= P2P_BUFFER_LIMIT) {
          dc.bufferedAmountLowThreshold = P2P_BUFFER_LIMIT / 2
          await new Promise<void>((resolve) => {
            dc.onbufferedamountlow = () => {
              dc.onbufferedamountlow = null
              resolve()
            }
          })
        }
        dc.send(await blob.slice(offset, offset + P2P_CHUNK_SIZE).arrayBuffer())
      }

      await updateP2PDiagnostic(state, 'push')
      dc.send(JSON.stringify({ kind: 'done', diagnostic: state.p2pDiagnostic || '' }))
      state.transferDone = true
      state.stopped = true
      setPushResultMode('p2pDone')
      updateP2PStatus('push', tLatest('p2pNotStored'), false, currentResultRef.current.url)
      setPushResultVisible(true)
      autoCollapseSendCard()
      showToast(tLatest('p2pSent'), 'success')
    },
    [autoCollapseSendCard, showToast, tLatest, updateP2PDiagnostic, updateP2PStatus],
  )

  const startP2PSenderAttempt = useCallback(
    async (mode: 'lan' | 'stun') => {
      const state = p2pStateRef.current
      if (!state || state.role !== 'sender' || state.stopped || state.connected || state.transferDone) {
        return
      }
      if (state.attemptMode === mode && state.pc) return

      const attempt = mode === 'lan' ? 1 : 2
      state.attempt = attempt
      state.attemptMode = mode
      state.pendingCandidates = state.futureCandidates[attempt] || []
      delete state.futureCandidates[attempt]
      clearP2PTimers(state)
      closeP2PConnection(state)
      updateP2PStatus(
        'push',
        mode === 'lan' ? tLatest('p2pLanCheck') : tLatest('p2pInternetCheck'),
        true,
        currentResultRef.current.url,
      )

      const pc = new RTCPeerConnection({ iceServers: mode === 'lan' ? [] : P2P_ICE_SERVERS })
      const dc = pc.createDataChannel('crossshare')
      state.pc = pc
      state.dc = dc

      pc.onicecandidate = (event) => {
        if (event.candidate && p2pStateRef.current === state && state.attempt === attempt) {
          postP2PMessage(state.sessionID, 'sender', 'receiver', 'candidate', {
            attempt,
            mode,
            data: event.candidate.toJSON(),
          }).catch(() => {})
        }
      }
      pc.onconnectionstatechange = () => {
        if (p2pStateRef.current !== state || state.attempt !== attempt || state.transferDone) return
        if (pc.connectionState === 'connected') {
          state.connected = true
          clearP2PTimers(state)
          updateP2PStatus('push', tLatest('p2pConnected'), true, currentResultRef.current.url)
          scheduleP2PDiagnostic(state, 'push')
        }
        if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
          if (!state.connected && mode === 'lan') {
            startP2PSenderAttempt('stun').catch(() => markP2PFailed())
            return
          }
          markP2PFailed()
        }
      }
      dc.onopen = () => {
        if (p2pStateRef.current !== state || state.attempt !== attempt) return
        state.connected = true
        clearP2PTimers(state)
        updateP2PStatus('push', tLatest('p2pConnected'), true, currentResultRef.current.url)
        scheduleP2PDiagnostic(state, 'push')
        if (state.input) {
          sendP2PContent(state.input).catch((e: any) => {
            showToast(`${tLatest('reqFail')}: ${e?.message || ''}`, 'error')
            markP2PFailed()
          })
        }
      }
      dc.onerror = () => {
        if (p2pStateRef.current === state && state.attempt === attempt) markP2PFailed()
      }

      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      await postP2PMessage(state.sessionID, 'sender', 'receiver', 'offer', {
        attempt,
        mode,
        data: { type: pc.localDescription!.type, sdp: pc.localDescription!.sdp },
      })
    },
    [markP2PFailed, postP2PMessage, scheduleP2PDiagnostic, sendP2PContent, showToast, tLatest, updateP2PStatus],
  )

  const armP2PConnectTimers = useCallback(
    (attempt: number) => {
      const state = p2pStateRef.current
      if (!state || state.role !== 'sender' || state.connected || state.transferDone) return
      clearP2PTimers(state)

      if (attempt === 1) {
        state.upgradeTimer = setTimeout(() => {
          if (p2pStateRef.current === state && !state.connected) {
            startP2PSenderAttempt('stun').catch(() => markP2PFailed())
          }
        }, P2P_LAN_FALLBACK_DELAY)
      }
      state.slowTimer = setTimeout(() => {
        if (p2pStateRef.current === state && !state.connected) {
          updateP2PStatus('push', tLatest('p2pSlow'), true, currentResultRef.current.url)
        }
      }, P2P_SLOW_NOTICE_DELAY)
      state.connectTimer = setTimeout(() => {
        if (p2pStateRef.current === state && !state.connected) {
          markP2PFailed()
        }
      }, P2P_CONNECT_TIMEOUT)
    },
    [markP2PFailed, startP2PSenderAttempt, tLatest, updateP2PStatus],
  )

  const showP2PReceiveError = useCallback(
    (message: string) => {
      if (p2pStateRef.current) stopP2PStatusTimer(p2pStateRef.current)
      setPullTextContent(null)
      setPullFile(null)
      setPullP2PStatus(null)
      setPullMetaText(message)
      setPullResultVisible(true)
      showToast(message, 'error')
    },
    [showToast],
  )

  const finishP2PReceive = useCallback(() => {
    const state = p2pStateRef.current
    if (!state || !state.receiveMeta) return
    const meta = state.receiveMeta
    const blob = new Blob(state.receiveChunks || [], {
      type: meta.content_type || 'application/octet-stream',
    })
    state.stopped = true
    stopP2PStatusTimer(state)

    if (meta.type === 'text') {
      blob.text().then((text) => {
        setPullFile(null)
        setPullTextContent(text)
        updateP2PStatus('pull', `${tLatest('p2pDone')} · ${tLatest('metaSize')}: ${humanSize(blob.size)}`, false, '')
        scheduleP2PDiagnostic(state, 'pull')
        showToast(tLatest('p2pDone'), 'success')
      })
      return
    }

    if (pullFileUrlRef.current) URL.revokeObjectURL(pullFileUrlRef.current)
    const url = URL.createObjectURL(blob)
    pullFileUrlRef.current = url
    setPullTextContent(null)
    setPullFile({ name: meta.filename || 'download', url })
    updateP2PStatus('pull', `${tLatest('p2pDone')} · ${tLatest('metaSize')}: ${humanSize(blob.size)}`, false, '')
    scheduleP2PDiagnostic(state, 'pull')
    showToast(tLatest('p2pDone'), 'success')
  }, [scheduleP2PDiagnostic, showToast, tLatest, updateP2PStatus])

  const startP2PReceiverAttempt = useCallback(
    async (attempt: number, mode: string) => {
      const state = p2pStateRef.current
      if (!state || state.role !== 'receiver' || state.stopped) return
      if (attempt < state.attempt) return
      updateP2PStatus('pull', mode === 'lan' ? tLatest('p2pLanCheck') : tLatest('p2pInternetCheck'), true, '')
      if (attempt === state.attempt && state.attemptMode === mode && state.pc) return

      if (attempt !== state.attempt || state.attemptMode !== mode || !state.pc) {
        state.attempt = attempt
        state.attemptMode = mode
        state.pendingCandidates = state.futureCandidates[attempt] || []
        delete state.futureCandidates[attempt]
        closeP2PConnection(state)
      }

      const pc = new RTCPeerConnection({ iceServers: mode === 'lan' ? [] : P2P_ICE_SERVERS })
      state.pc = pc

      pc.onicecandidate = (event) => {
        if (event.candidate && p2pStateRef.current === state && state.attempt === attempt) {
          postP2PMessage(state.sessionID, 'receiver', 'sender', 'candidate', {
            attempt,
            mode,
            data: event.candidate.toJSON(),
          }).catch(() => {})
        }
      }
      pc.onconnectionstatechange = () => {
        if (p2pStateRef.current !== state || state.attempt !== attempt) return
        if (pc.connectionState === 'connected') {
          state.connected = true
          updateP2PStatus('pull', tLatest('p2pConnected'), true, '')
          scheduleP2PDiagnostic(state, 'pull')
        }
        if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
          updateP2PStatus('pull', tLatest('p2pSlow'), true, '')
        }
      }
      pc.ondatachannel = (event) => {
        if (p2pStateRef.current !== state || state.attempt !== attempt) return
        state.dc = event.channel
        state.dc.binaryType = 'arraybuffer'
        state.dc.onmessage = (ev: MessageEvent) => {
          if (p2pStateRef.current !== state || state.role !== 'receiver') return
          if (typeof ev.data === 'string') {
            const msg = JSON.parse(ev.data)
            if (msg.kind === 'diagnostic' && msg.diagnostic) {
              state.p2pDiagnostic = msg.diagnostic
              renderP2PStatus(
                state.statusTarget || 'pull',
                state.statusMessage || tLatest('p2pConnected'),
                state.statusActive !== false,
                state.statusDetail || '',
                state.startedAt,
                state.p2pDiagnostic,
              )
              return
            }
            if (msg.kind === 'meta') {
              state.receiveMeta = msg
              state.receiveChunks = []
              state.receivedSize = 0
              updateP2PStatus(
                'pull',
                `${tLatest('p2pReceiving')} · ${tLatest('metaSize')}: ${humanSize(msg.size || 0)}`,
                true,
                '',
              )
              return
            }
            if (msg.kind === 'done') {
              if (msg.diagnostic) state.p2pDiagnostic = msg.diagnostic
              finishP2PReceive()
            }
            return
          }
          state.receiveChunks = state.receiveChunks || []
          state.receiveChunks.push(ev.data)
          const nextReceived = (state.receivedSize || 0) + ev.data.byteLength
          state.receivedSize = nextReceived
          if (state.receiveMeta) {
            updateP2PStatus(
              'pull',
              `${tLatest('p2pReceiving')} · ${humanSize(nextReceived)} / ${humanSize(state.receiveMeta.size || 0)}`,
              true,
              '',
            )
          }
        }
        state.dc.onerror = () => showP2PReceiveError(tLatest('p2pSenderOffline'))
      }
    },
    [finishP2PReceive, postP2PMessage, renderP2PStatus, scheduleP2PDiagnostic, showP2PReceiveError, tLatest, updateP2PStatus],
  )

  const handlePull = useCallback(
    async (explicitKey?: string) => {
      const key = (explicitKey !== undefined ? explicitKey : pullKey).trim()
      if (!key) {
        showToast(tLatest('enterKeyWarn'), 'error')
        return
      }
      setPullMoreOpen(false)
      setPullBusy(true)

      try {
        const headers: Record<string, string> = { ...getAuthHeaders() }
        if (deleteAfterPull) {
          headers['Delete-After-Pull'] = 'true'
        }

        if (!headers['Delete-After-Pull']) {
          const jsonResp = await fetch(getApiUrl(`/pull/${key}`), {
            headers: { ...headers, Accept: 'application/json' },
          })
          const ct = jsonResp.headers.get('Content-Type') || ''
          if (ct.includes('application/json')) {
            const data = await jsonResp.json()
            if (data.code !== 0) {
              showToast(`${tLatest('pullFail')}: ${data.msg}`, 'error')
              return
            }
            const r = data.data
            if (r.text !== undefined) {
              setPullFile(null)
              setPullTextContent(r.text)
              setPullP2PStatus(null)
              let meta = `Key: ${r.key} · ${tLatest('metaSize')}: ${humanSize(r.size)} · ${tLatest('metaType')}: ${r.content_type}`
              if (r.deleted) meta += ` · ${tLatest('deleted')}`
              setPullMetaText(meta)
              setPullResultVisible(true)
              showToast(tLatest('pullOk'), 'success')
              return
            }
          }
        }

        const streamResp = await fetch(getApiUrl(`/pull/${key}`), { headers })
        if (!streamResp.ok) {
          const errData = await streamResp.json().catch(() => null)
          showToast(`${tLatest('pullFail')}: ${errData?.msg || `HTTP ${streamResp.status}`}`, 'error')
          return
        }

        const blob = await streamResp.blob()
        const shareType = streamResp.headers.get('Crossshare-Type')
        const filename = streamResp.headers.get('Crossshare-Filename') || 'download'
        const fileCount = streamResp.headers.get('Crossshare-File-Count')
        const deleted = streamResp.headers.get('Key-Deleted') === 'true'

        setPullP2PStatus(null)
        if (shareType === 'Text') {
          const text = await blob.text()
          setPullFile(null)
          setPullTextContent(text)
        } else {
          if (pullFileUrlRef.current) URL.revokeObjectURL(pullFileUrlRef.current)
          const url = URL.createObjectURL(blob)
          pullFileUrlRef.current = url
          setPullTextContent(null)
          setPullFile({ name: filename, url })
        }

        let meta = `Key: ${key} · ${tLatest('metaSize')}: ${humanSize(blob.size)}`
        if (shareType) meta += ` · ${tLatest('metaType')}: ${shareType}`
        if (fileCount) meta += ` · ${tLatest('metaFiles')}: ${fileCount}`
        if (deleted) meta += ` · ${tLatest('deleted')}`
        setPullMetaText(meta)
        setPullResultVisible(true)
        showToast(tLatest('pullOk'), 'success')
      } catch (e: any) {
        showToast(`${tLatest('reqFail')}: ${e?.message || ''}`, 'error')
      } finally {
        setPullBusy(false)
      }
    },
    [deleteAfterPull, getApiUrl, getAuthHeaders, pullKey, showToast, tLatest],
  )

  const handleP2PMessage = useCallback(
    async (msg: { seq: number; type: string; payload: any }) => {
      const state = p2pStateRef.current
      if (!state) return

      const unpack = (payload: any) => {
        if (payload && typeof payload.attempt === 'number' && payload.data) {
          return { attempt: payload.attempt as number, mode: (payload.mode as string) || 'stun', data: payload.data }
        }
        return { attempt: 1, mode: 'stun', data: payload }
      }

      const flushCandidates = async () => {
        if (!state.pc || !state.pendingCandidates.length) return
        const cands = state.pendingCandidates.splice(0)
        for (const cand of cands) {
          await state.pc.addIceCandidate(new RTCIceCandidate(cand))
        }
      }

      if (msg.type === 'fallback' && state.role === 'receiver' && msg.payload?.key) {
        stopP2PTransport(state)
        state.stopped = true
        p2pStateRef.current = null
        const k = String(msg.payload.key)
        setPullKey(k)
        setPullP2PStatus(null)
        setPullMetaText(tLatest('p2pFallback'))
        await handlePull(k)
        return
      }

      if (msg.type === 'offer' && state.role === 'receiver') {
        const signal = unpack(msg.payload)
        if (signal.attempt < state.attempt) return
        await startP2PReceiverAttempt(signal.attempt, signal.mode)
        const pc = state.pc
        if (!pc) return
        updateP2PStatus('pull', tLatest('p2pSignalExchange'), true, '')
        await pc.setRemoteDescription(new RTCSessionDescription(signal.data))
        await flushCandidates()
        const answer = await pc.createAnswer()
        await pc.setLocalDescription(answer)
        await postP2PMessage(state.sessionID, 'receiver', 'sender', 'answer', {
          attempt: state.attempt,
          mode: state.attemptMode,
          data: { type: pc.localDescription!.type, sdp: pc.localDescription!.sdp },
        })
        return
      }

      if (msg.type === 'answer' && state.role === 'sender') {
        const signal = unpack(msg.payload)
        if (signal.attempt !== state.attempt || !state.pc) return
        updateP2PStatus('push', tLatest('p2pSignalExchange'), true, currentResultRef.current.url)
        await state.pc.setRemoteDescription(new RTCSessionDescription(signal.data))
        await flushCandidates()
        armP2PConnectTimers(signal.attempt)
        return
      }

      if (msg.type === 'candidate' && msg.payload) {
        const signal = unpack(msg.payload)
        if (signal.attempt < state.attempt) return
        if (signal.attempt > state.attempt) {
          state.futureCandidates[signal.attempt] = state.futureCandidates[signal.attempt] || []
          state.futureCandidates[signal.attempt].push(signal.data)
          return
        }
        if (!state.pc || !state.pc.remoteDescription) {
          state.pendingCandidates.push(signal.data)
          return
        }
        await state.pc.addIceCandidate(new RTCIceCandidate(signal.data))
      }
    },
    [armP2PConnectTimers, handlePull, postP2PMessage, startP2PReceiverAttempt, tLatest, updateP2PStatus],
  )

  const pollP2PMessages = useCallback(
    async (role: 'sender' | 'receiver') => {
      while (p2pStateRef.current && !p2pStateRef.current.stopped && p2pStateRef.current.role === role) {
        const state = p2pStateRef.current
        const controller = new AbortController()
        state.pollController = controller
        try {
          const resp = await fetch(
            getApiUrl(`/p2p/sessions/${state.sessionID}/messages?to=${role}&after=${state.lastSeq}&wait=${P2P_POLL_WAIT_SECONDS}`),
            { headers: getAuthHeaders(), signal: controller.signal },
          )
          if (state.pollController === controller) state.pollController = null
          if (resp.status === 204) return
          const data = await resp.json()
          if (data.code !== 0) {
            if (p2pStateRef.current && p2pStateRef.current.role === 'receiver') {
              showP2PReceiveError(tLatest('p2pSenderOffline'))
            }
            return
          }
          for (const msg of data.data.messages || []) {
            if (!p2pStateRef.current || p2pStateRef.current.stopped) return
            p2pStateRef.current.lastSeq = Math.max(p2pStateRef.current.lastSeq, msg.seq)
            await handleP2PMessage(msg)
          }
        } catch (e: any) {
          if (state.pollController === controller) state.pollController = null
          if (e?.name === 'AbortError') return
          if (p2pStateRef.current && !p2pStateRef.current.stopped) {
            if (role === 'sender') markP2PFailed()
            if (role === 'receiver') showP2PReceiveError(tLatest('p2pSenderOffline'))
          }
          return
        }
      }
    },
    [getApiUrl, getAuthHeaders, handleP2PMessage, markP2PFailed, showP2PReceiveError, tLatest],
  )

  const startP2PPush = useCallback(
    async (input: PushInput) => {
      cancelP2P(false)
      const resp = await fetch(getApiUrl('/p2p/sessions'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: '{}',
      })
      const data = await resp.json()
      if (data.code !== 0) {
        resetSendCardAutoState(true)
        showToast(`${tLatest('pushFail')}: ${data.msg}`, 'error')
        return
      }

      const sessionID = data.data.session_id as string
      const shareUrl = buildP2PUrl(sessionID)
      const res = { mode: 'p2p', key: sessionID, url: shareUrl }
      setCurrentResult(res)
      currentResultRef.current = res
      setResultKeyLabel(tLatest('p2pLinkLabel'))
      setServerResultMeta('')
      setCountdownExpireAt(null)
      setPushResultMode('p2p')
      setPushResultVisible(true)

      p2pStateRef.current = {
        role: 'sender',
        sessionID,
        pc: null,
        dc: null,
        input,
        lastSeq: 0,
        stopped: false,
        connected: false,
        transferStarted: false,
        transferDone: false,
        fallbackStarted: false,
        attempt: 0,
        attemptMode: '',
        connectTimer: null,
        upgradeTimer: null,
        slowTimer: null,
        statusTimer: null,
        pollController: null,
        p2pDiagnostic: '',
        p2pDiagnosticSent: false,
        startedAt: Date.now(),
        pendingCandidates: [],
        futureCandidates: {},
      }

      pollP2PMessages('sender')
      updateP2PStatus('push', tLatest('p2pWaiting'), true, shareUrl)
      autoCollapseSendCard()
      await startP2PSenderAttempt('lan')
      showToast(tLatest('p2pWaiting'), 'success')
    },
    [autoCollapseSendCard, cancelP2P, getApiUrl, getAuthHeaders, pollP2PMessages, resetSendCardAutoState, showToast, startP2PSenderAttempt, tLatest, updateP2PStatus],
  )

  const startP2PReceive = useCallback(
    (sessionID: string) => {
      if (!supportsP2P()) {
        showP2PReceiveError(tLatest('p2pUnsupported'))
        return
      }
      cancelP2P(false)
      setPullTextContent(null)
      setPullFile(null)
      setPullMetaText('')
      setPullResultVisible(true)

      p2pStateRef.current = {
        role: 'receiver',
        sessionID,
        pc: null,
        dc: null,
        lastSeq: 0,
        stopped: false,
        connected: false,
        attempt: 0,
        attemptMode: '',
        transferStarted: false,
        transferDone: false,
        fallbackStarted: false,
        connectTimer: null,
        upgradeTimer: null,
        slowTimer: null,
        statusTimer: null,
        pollController: null,
        p2pDiagnostic: '',
        p2pDiagnosticSent: false,
        startedAt: Date.now(),
        receiveMeta: null,
        receiveChunks: [],
        receivedSize: 0,
        pendingCandidates: [],
        futureCandidates: {},
      }

      updateP2PStatus('pull', tLatest('p2pSignalExchange'), true, '')
      pollP2PMessages('receiver')
    },
    [cancelP2P, pollP2PMessages, showP2PReceiveError, tLatest, updateP2PStatus],
  )

  // Auto pull from URL ?key= or ?p2p=
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const p2pSession = params.get('p2p')
    const key = params.get('key')
    if (!key && !p2pSession) return

    const cleanUrl = window.location.pathname + window.location.hash
    window.history.replaceState(null, '', cleanUrl)
    setActiveTab('pull')

    if (p2pSession) {
      setTimeout(() => startP2PReceive(p2pSession), 0)
      return
    }
    if (key) {
      setPullKey(key)
      setTimeout(() => {
        handlePull(key)
      }, 0)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const clearPushResult = (resetCard = true) => {
    setCountdownExpireAt(null)
    setCurrentResult({ mode: '', key: '', url: '' })
    setResultKeyLabel('')
    setServerResultMeta('')
    setPushP2PStatus(null)
    setPushResultVisible(false)
    setPushResultMode('')
    if (resetCard) resetSendCardAutoState(true)
  }

  const handlePush = useCallback(async () => {
    resetSendCardAutoState(false)
    const ttl = parseInt(pushTtl, 10) || 0
    let input: PushInput | null = null

    if (pushMode === 'text') {
      if (!pushText.trim()) {
        resetSendCardAutoState(true)
        showToast(t('enterText'), 'error')
        return
      }
      input = {
        type: 'text',
        text: pushText,
        ttl,
        size: new Blob([pushText]).size,
        contentType: 'text/plain; charset=utf-8',
      }
    } else {
      if (!selectedFiles.length) {
        resetSendCardAutoState(true)
        showToast(t('selectFile'), 'error')
        return
      }
      const totalSize = selectedFiles.reduce((sum, file) => sum + file.size, 0)
      const singleFile = selectedFiles.length === 1 ? selectedFiles[0] : null
      input = {
        type: singleFile ? 'binary' : 'files',
        file: singleFile,
        files: [...selectedFiles],
        ttl,
        size: totalSize,
        filename: singleFile ? singleFile.name : 'crossshare-files.zip',
        contentType: singleFile ? singleFile.type || 'application/octet-stream' : 'application/zip',
      }
    }

    setPushMoreOpen(false)
    setPushBusy(true)
    clearPushResult(false)

    try {
      if (transferMode === 'smart' && input.type !== 'files') {
        if (!supportsP2P()) {
          showToast(t('p2pUnsupported'), 'error')
        } else {
          await startP2PPush(input)
          return
        }
      }
      cancelP2P(false)
      const res = await uploadToServer(input)
      renderServerPushResult(res)
      showToast(t('pushOk'), 'success')
    } catch (e: any) {
      resetSendCardAutoState(true)
      showToast(`${t('pushFail')}: ${e?.message || ''}`, 'error')
    } finally {
      setPushBusy(false)
    }
  }, [cancelP2P, pushMode, pushText, pushTtl, renderServerPushResult, resetSendCardAutoState, selectedFiles, showToast, startP2PPush, t, transferMode, uploadToServer])

  // Keyboard shortcuts
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setQrModalUrl(null)
        setSettingsOpen(false)
        return
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && activeTab === 'push') {
        if (!pushBusy) {
          handlePush()
        }
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [activeTab, handlePush, pushBusy])

  const addSelectedFiles = (files: FileList | File[]) => {
    resetSendCardAutoState(true)
    setSelectedFiles((prev) => {
      const seen = new Set(prev.map(fileSignature))
      const next = [...prev]
      for (const file of Array.from(files)) {
        const sig = fileSignature(file)
        if (seen.has(sig)) continue
        seen.add(sig)
        next.push(file)
      }
      return next
    })
  }

  const removeSelectedFile = (index: number) => {
    resetSendCardAutoState(true)
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index))
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const clearSelectedFiles = () => {
    resetSendCardAutoState(true)
    setSelectedFiles([])
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleSaveSettings = () => {
    const cleanUrl = serverUrl.replace(/\/+$/, '')
    setServerUrl(cleanUrl)
    localStorage.setItem('cs_server', cleanUrl)
    localStorage.setItem('cs_token', authToken)
    showToast(t('settingsSaved'), 'success')
  }

  const handleHealthCheck = async () => {
    setHealthStatus({ text: t('checking'), color: 'var(--text2)' })
    try {
      const base = serverUrl.replace(/\/+$/, '')
      const headers = authToken ? { Authorization: `Bearer ${authToken}` } : undefined
      const resp = await fetch(`${base}/api/v1/health`, { headers })
      const data = await resp.json()
      if (data.code === 0) {
        setHealthStatus({ text: `✓ ${data.data.status}`, color: 'var(--success)' })
      } else {
        setHealthStatus({ text: `✗ ${data.msg}`, color: 'var(--danger)' })
      }
    } catch {
      setHealthStatus({ text: t('noConnect'), color: 'var(--danger)' })
    }
  }

  const handleDelete = async () => {
    const key = deleteKey.trim()
    if (!key) {
      showToast(t('enterKeyWarn'), 'error')
      return
    }
    setDeleteBusy(true)
    try {
      const resp = await fetch(getApiUrl(`/pull/${key}`), {
        method: 'DELETE',
        headers: getAuthHeaders(),
      })
      const data = await resp.json()
      if (data.code !== 0) {
        showToast(`${t('deleteFail')}: ${data.msg}`, 'error')
        return
      }
      setDeleteMeta(`Key: ${data.data.key} · ${t('deleted')}`)
      showToast(t('deleteOk'), 'success')
    } catch (e: any) {
      showToast(`${t('reqFail')}: ${e?.message || ''}`, 'error')
    } finally {
      setDeleteBusy(false)
    }
  }

  const handleClearPull = () => {
    if (p2pStateRef.current && p2pStateRef.current.role === 'receiver') {
      const state = p2pStateRef.current
      state.stopped = true
      stopP2PTransport(state)
      p2pStateRef.current = null
    }
    if (pullFileUrlRef.current) {
      URL.revokeObjectURL(pullFileUrlRef.current)
      pullFileUrlRef.current = null
    }
    setPullKey('')
    setPullTextContent(null)
    setPullFile(null)
    setPullMetaText('')
    setPullP2PStatus(null)
    setPullResultVisible(false)
  }

  const charCount = Array.from(pushText).length
  const pushReady = pushMode === 'text' ? Boolean(pushText.trim()) : selectedFiles.length > 0
  const fileCountLabel = (cnt: number) => (lang === 'zh' ? `${cnt} 个文件` : `${cnt} files`)
  const totalSelectedSize = selectedFiles.reduce((sum, f) => sum + f.size, 0)

  // Countdown computation
  void countdownTick
  const countdownRemaining = countdownExpireAt ? countdownExpireAt - Math.floor(Date.now() / 1000) : null

  const renderP2PBlock = (st: P2PStatusState) => {
    const elapsed = st.startedAt ? `${((Date.now() - st.startedAt) / 1000).toFixed(1)}s` : ''
    const fullDetail = [st.detail, st.diagnostic].filter(Boolean).join(' · ')
    return (
      <>
        <div className={`p2p-status ${st.active ? 'is-active' : ''}`}>
          <span className={st.active ? 'p2p-spinner' : 'p2p-dot'} />
          <span className="p2p-status-main">{st.message}</span>
          {elapsed && (
            <span className="p2p-elapsed">
              {t('p2pElapsed')}: {elapsed}
            </span>
          )}
        </div>
        {fullDetail && <div className="p2p-status-detail">{fullDetail}</div>}
      </>
    )
  }

  return (
    <>
      <div className="aurora" aria-hidden="true">
        <span className="blob blob-a" />
        <span className="blob blob-b" />
        <span className="blob blob-c" />
      </div>

      <div className="app-shell">
        <header className="app-header">
          <a className="brand" href="/" aria-label="CrossShare">
            <img src="/favicon.svg?v=2" alt="" />
            <div>
              <h1>CrossShare</h1>
              <p className="subtitle">{t('subtitle')}</p>
            </div>
          </a>
          <div className="header-actions">
            <button
              className="icon-btn"
              id="themeToggle"
              title={t('toggleTheme')}
              onClick={() => setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'))}
            >
              {theme === 'dark' ? '☀' : '☾'}
            </button>
            <button
              className="icon-btn"
              id="langToggle"
              title={lang === 'zh' ? 'Switch to English' : '切换到中文'}
              onClick={() => setLang((prev) => (prev === 'zh' ? 'en' : 'zh'))}
            >
              {lang === 'zh' ? 'EN' : '中'}
            </button>
            <button
              ref={settingsBtnRef}
              className="icon-btn"
              id="settingsToggle"
              title={t('settings')}
              onClick={() => setSettingsOpen((prev) => !prev)}
            >
              ⚙
            </button>
          </div>
        </header>

        <div ref={settingsRef} id="settingsPanel" className={`settings ${settingsOpen ? '' : 'hidden'}`}>
          <label>
            <span>{t('serverUrlLabel')}</span>
            <input
              type="text"
              id="serverUrl"
              value={serverUrl}
              onChange={(e) => setServerUrl(e.target.value)}
              placeholder="http://localhost:10431"
            />
          </label>
          <label>
            <span>{t('tokenLabel')}</span>
            <input
              type="password"
              id="authToken"
              value={authToken}
              onChange={(e) => setAuthToken(e.target.value)}
              placeholder="JWT token"
            />
          </label>
          <div className="settings-actions">
            <button className="btn btn-sm" id="saveSettings" onClick={handleSaveSettings}>
              {t('save')}
            </button>
            <button className="btn btn-sm btn-ghost" id="healthCheck" onClick={handleHealthCheck}>
              {t('checkConn')}
            </button>
            <span id="healthStatus" style={{ color: healthStatus.color || undefined }}>
              {healthStatus.text}
            </span>
          </div>
        </div>

        <main className="stage">
          <nav className="tabs">
            <button
              className={`tab ${activeTab === 'push' ? 'active' : ''}`}
              onClick={() => setActiveTab('push')}
            >
              {t('push')}
            </button>
            <button
              className={`tab ${activeTab === 'pull' ? 'active' : ''}`}
              onClick={() => setActiveTab('pull')}
            >
              {t('pull')}
            </button>
          </nav>

          <section id="pushTab" className={`tab-content ${activeTab === 'push' ? 'active' : ''}`}>
            <div className={`glass-card send-card ${sendCardCollapsed ? 'collapsed' : ''}`}>
              <button
                type="button"
                id="sendCardToggle"
                className="send-card-strip"
                title={sendCardCollapsed ? t('expandSendPanel') : t('collapseSendPanel')}
                aria-label={sendCardCollapsed ? t('expandSendPanel') : t('collapseSendPanel')}
                aria-expanded={!sendCardCollapsed}
                onClick={() => {
                  if (sendCardCollapsed) {
                    collapseSendCard(false, true)
                  } else {
                    collapseSendCard(true, false)
                  }
                }}
              >
                <span className="send-card-strip-icon" aria-hidden="true" />
                <span className="send-card-strip-text">
                  <span>{t('sendPanelCollapsedTitle')}</span>
                </span>
              </button>

              <div className="mode-switch">
                <button
                  className={`mode ${pushMode === 'text' ? 'active' : ''}`}
                  data-mode="text"
                  onClick={() => {
                    resetSendCardAutoState(true)
                    setPushMode('text')
                  }}
                >
                  {t('text')}
                </button>
                <button
                  className={`mode ${pushMode === 'file' ? 'active' : ''}`}
                  data-mode="file"
                  onClick={() => {
                    resetSendCardAutoState(true)
                    setPushMode('file')
                  }}
                >
                  {t('file')}
                </button>
              </div>

              <div id="textMode" className={pushMode === 'text' ? '' : 'hidden'}>
                <label className="sr-only" htmlFor="pushText">
                  {t('textContent')}
                </label>
                <div className="text-editor">
                  <textarea
                    id="pushText"
                    value={pushText}
                    onChange={(e) => {
                      resetSendCardAutoState(false)
                      setPushText(e.target.value)
                    }}
                    placeholder={t('pushPlaceholder')}
                    rows={7}
                  />
                  <span className="char-count" aria-hidden="true">
                    {charCount ? `${charCount} ${t('charsUnit')}` : ''}
                  </span>
                </div>
              </div>

              <div id="fileMode" className={pushMode === 'file' ? '' : 'hidden'}>
                <div
                  className={`drop-zone ${dragOver ? 'dragover' : ''} ${selectedFiles.length > 0 ? 'has-file' : ''} ${selectedFiles.length > 1 ? 'has-multiple-files' : ''}`}
                  id="dropZone"
                  onDragOver={(e) => {
                    e.preventDefault()
                    setDragOver(true)
                  }}
                  onDragLeave={() => setDragOver(false)}
                  onDrop={(e) => {
                    e.preventDefault()
                    setDragOver(false)
                    if (e.dataTransfer.files.length) {
                      addSelectedFiles(e.dataTransfer.files)
                    }
                  }}
                >
                  <button
                    type="button"
                    id="clearFile"
                    className={`file-clear ${selectedFiles.length === 0 ? 'hidden' : ''}`}
                    title={t('clear')}
                    aria-label={t('clear')}
                    onClick={(e) => {
                      e.preventDefault()
                      e.stopPropagation()
                      clearSelectedFiles()
                    }}
                  >
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M6 6l12 12" />
                      <path d="M18 6L6 18" />
                    </svg>
                  </button>
                  <span className="drop-icon" aria-hidden="true">
                    <svg className="drop-icon-upload" viewBox="0 0 24 24" fill="none">
                      <path d="M12 16V5" />
                      <path d="M8 9l4-4 4 4" />
                      <path d="M5 16v2a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-2" />
                    </svg>
                    <svg className="drop-icon-ready" viewBox="0 0 24 24" fill="none">
                      <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" />
                      <path d="M14 3v5h5" />
                      <path d="M9 14l2 2 4-4" />
                    </svg>
                  </span>
                  <p id="dropTitle">
                    {selectedFiles.length > 0
                      ? lang === 'zh'
                        ? `已选择 ${fileCountLabel(selectedFiles.length)}`
                        : `${fileCountLabel(selectedFiles.length)} selected`
                      : t('dropHint')}
                  </p>
                  <span id="dropSubtitle" className="drop-subtitle">
                    {selectedFiles.length > 0
                      ? lang === 'zh'
                        ? `总计 ${humanSize(totalSelectedSize)}，可继续发送`
                        : `${humanSize(totalSelectedSize)} total, ready to send`
                      : t('dropSubHint')}
                  </span>
                  <ul
                    id="selectedFileList"
                    className={`selected-file-list ${selectedFiles.length === 0 ? 'hidden' : ''}`}
                    aria-label={t('selectedFiles')}
                    aria-live="polite"
                  >
                    {selectedFiles.map((file, idx) => (
                      <li key={fileSignature(file)} className="selected-file-item">
                        <span className="selected-file-name" title={file.name}>
                          {file.name}
                        </span>
                        <span className="selected-file-size">{humanSize(file.size)}</span>
                        <button
                          type="button"
                          className="selected-file-remove"
                          title={t('removeFile')}
                          aria-label={`${t('removeFile')}: ${file.name}`}
                          onClick={(e) => {
                            e.preventDefault()
                            e.stopPropagation()
                            removeSelectedFile(idx)
                          }}
                        >
                          <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                            <path d="M6 6l12 12" />
                            <path d="M18 6L6 18" />
                          </svg>
                        </button>
                      </li>
                    ))}
                  </ul>
                  <input
                    ref={fileInputRef}
                    type="file"
                    id="fileInput"
                    multiple
                    onChange={(e) => {
                      if (e.target.files?.length) {
                        addSelectedFiles(e.target.files)
                        e.target.value = ''
                      }
                    }}
                  />
                </div>
              </div>

              <div className="options-bar">
                <label
                  className="direct-toggle"
                  title={transferMode === 'smart' ? t('directTransferHint') : t('serverTransfer')}
                >
                  <input
                    type="checkbox"
                    id="directTransferToggle"
                    checked={transferMode === 'smart'}
                    aria-label={transferMode === 'smart' ? t('smartTransfer') : t('serverTransfer')}
                    onChange={(e) => {
                      resetSendCardAutoState(true)
                      const next = e.target.checked ? 'smart' : 'server'
                      setTransferMode(next)
                      localStorage.setItem('cs_transfer', next)
                    }}
                  />
                  <span className="toggle-track" aria-hidden="true">
                    <span className="toggle-thumb" />
                  </span>
                  <span className="direct-toggle-text">
                    <strong className="direct-toggle-title">
                      {transferMode === 'smart' ? t('smartTransfer') : t('serverTransfer')}
                    </strong>
                    <small className="direct-toggle-desc">{t('smartTransferDesc')}</small>
                  </span>
                </label>

                <details
                  className="push-more"
                  open={pushMoreOpen}
                  onToggle={(e) => setPushMoreOpen((e.currentTarget as HTMLDetailsElement).open)}
                >
                  <summary>{t('moreSettings')}</summary>
                  <div className="options-row">
                    <label className="inline">
                      <span>TTL</span>
                      <input
                        type="number"
                        id="pushTtl"
                        value={pushTtl}
                        onChange={(e) => setPushTtl(e.target.value)}
                        placeholder={t('defaultPlaceholder')}
                        min={1}
                      />
                      <span className="unit">{t('seconds')}</span>
                    </label>
                  </div>
                </details>
              </div>

              <button
                className="btn btn-hero"
                id="pushBtn"
                disabled={pushBusy || !pushReady}
                title={t('push')}
                onClick={handlePush}
              >
                {pushBusy ? t('pushing') : t('push')}
              </button>
            </div>

            <div id="pushResult" className={`glass-card result ${pushResultVisible ? '' : 'hidden'}`}>
              <div className="result-key">
                <span id="resultKey">{resultKeyLabel}</span>
                <div className="result-key-actions">
                  <button
                    className={`btn btn-sm ${pushResultMode === 'server' ? '' : 'hidden'}`}
                    onClick={() => {
                      if (!currentResult.key) return
                      copyText(currentResult.key).then(() => showToast(t('keyCopied'), 'success')).catch(() => {})
                    }}
                  >
                    {t('copy')}
                  </button>
                  <button
                    className={`btn btn-sm ${pushResultMode === 'server' || pushResultMode === 'p2p' ? '' : 'hidden'}`}
                    onClick={() => {
                      const u = currentResult.url || buildShareUrl(currentResult.key || resultKeyLabel)
                      copyText(u).then(() => showToast(t('linkCopied'), 'success')).catch(() => {})
                    }}
                  >
                    {t('copyLink')}
                  </button>
                  <button
                    className={`btn btn-sm ${pushResultMode === 'server' || pushResultMode === 'p2p' ? '' : 'hidden'}`}
                    onClick={() => {
                      const u = currentResult.url || buildShareUrl(currentResult.key || resultKeyLabel)
                      setQrModalUrl(u)
                    }}
                  >
                    {t('showQr')}
                  </button>
                  <button
                    className={`btn btn-sm btn-ghost ${pushResultMode === 'server' ? '' : 'hidden'}`}
                    onClick={() => clearPushResult(true)}
                  >
                    {t('cleanup')}
                  </button>
                  <button
                    className={`btn btn-sm ${pushResultMode === 'p2p' ? '' : 'hidden'}`}
                    onClick={() => fallbackP2PToServer()}
                  >
                    {t('saveToServer')}
                  </button>
                  <button
                    className={`btn btn-sm btn-ghost ${pushResultMode === 'p2p' ? '' : 'hidden'}`}
                    onClick={() => cancelP2P(true)}
                  >
                    {t('cancel')}
                  </button>
                </div>
              </div>
              {countdownRemaining !== null && (
                <div
                  id="countdownDisplay"
                  className={`countdown ${
                    countdownRemaining <= 0
                      ? 'countdown-expired'
                      : countdownRemaining <= 60
                      ? 'countdown-warning'
                      : ''
                  }`}
                >
                  {countdownRemaining <= 0
                    ? t('expired')
                    : `${t('expiresIn')}: ${humanDuration(countdownRemaining)}`}
                </div>
              )}
              <div className="result-meta" id="resultMeta">
                {pushP2PStatus ? renderP2PBlock(pushP2PStatus) : serverResultMeta}
              </div>
            </div>
          </section>

          <section id="pullTab" className={`tab-content ${activeTab === 'pull' ? 'active' : ''}`}>
            <div className="glass-card receive-card">
              <div className="pull-control">
                <label className="field-label" htmlFor="pullKey">
                  {t('enterKey')}
                </label>
                <div className="pull-input">
                  <input
                    type="text"
                    id="pullKey"
                    value={pullKey}
                    onChange={(e) => setPullKey(e.target.value)}
                    placeholder={t('enterKey')}
                  />
                  <button
                    className="btn btn-primary"
                    id="pullBtn"
                    disabled={pullBusy}
                    title={t('pull')}
                    onClick={() => handlePull()}
                  >
                    {pullBusy ? t('pulling') : t('pull')}
                  </button>
                </div>
                <label className="checkbox receive-option">
                  <input
                    type="checkbox"
                    id="deleteAfterPull"
                    checked={deleteAfterPull}
                    onChange={(e) => setDeleteAfterPull(e.target.checked)}
                  />
                  <span>
                    <strong>{t('deleteAfterPull')}</strong>
                    <small>{t('deleteAfterPullHint')}</small>
                  </span>
                </label>
              </div>

              <details
                className="pull-more"
                open={pullMoreOpen}
                onToggle={(e) => setPullMoreOpen((e.currentTarget as HTMLDetailsElement).open)}
              >
                <summary>{t('moreActions')}</summary>
                <div className="pull-actions">
                  <button className="btn btn-ghost" id="clearPull" onClick={handleClearPull}>
                    {t('clear')}
                  </button>
                </div>

                <div className="delete-section">
                  <div className="pull-input">
                    <input
                      type="text"
                      id="deleteKey"
                      value={deleteKey}
                      onChange={(e) => setDeleteKey(e.target.value)}
                      placeholder={t('enterKey')}
                    />
                    <button className="btn btn-danger" id="deleteBtn" disabled={deleteBusy} onClick={handleDelete}>
                      {t('delete')}
                    </button>
                  </div>
                  <div id="deleteResult" className={`result ${deleteMeta ? '' : 'hidden'}`}>
                    <div className="result-meta">{deleteMeta}</div>
                  </div>
                </div>
              </details>
            </div>

            <div id="pullResult" className={`glass-card result ${pullResultVisible ? '' : 'hidden'}`}>
              <div id="pullTextResult" className={pullTextContent !== null ? '' : 'hidden'}>
                <div className="pull-text-header">
                  <span className="pull-label">{t('textContent')}</span>
                  <button
                    className="btn btn-sm"
                    onClick={() => {
                      if (pullTextContent !== null) {
                        copyText(pullTextContent).then(() => showToast(t('contentCopied'), 'success')).catch(() => {})
                      }
                    }}
                  >
                    {t('copy')}
                  </button>
                </div>
                <pre id="pullTextContent">{pullTextContent || ''}</pre>
              </div>
              <div id="pullFileResult" className={pullFile ? '' : 'hidden'}>
                <div className="file-download">
                  <span id="pullFileName">{pullFile?.name || ''}</span>
                  <a id="pullFileLink" className="btn btn-sm" href={pullFile?.url} download={pullFile?.name}>
                    {t('download')}
                  </a>
                </div>
              </div>
              <div className="result-meta" id="pullMeta">
                {pullP2PStatus ? renderP2PBlock(pullP2PStatus) : pullMetaText}
              </div>
            </div>
          </section>
        </main>

        <div id="toast" className={`toast ${toastState.type} ${toastState.visible ? '' : 'hidden'}`}>
          {toastState.msg}
        </div>
      </div>

      <div id="qrModal" className={`modal ${qrModalUrl ? '' : 'hidden'}`} role="dialog" aria-modal="true">
        <div className="modal-backdrop" onClick={() => setQrModalUrl(null)} />
        <div className="modal-panel">
          <div className="modal-header">
            <h2>{t('shareQr')}</h2>
            <button className="icon-btn" title={t('close')} onClick={() => setQrModalUrl(null)}>
              &times;
            </button>
          </div>
          <div className="qr-box">
            <canvas ref={qrCanvasRef} id="qrCanvas" width={300} height={300} />
          </div>
          <div className="qr-link">{qrModalUrl || ''}</div>
        </div>
      </div>
    </>
  )
}

export default App
