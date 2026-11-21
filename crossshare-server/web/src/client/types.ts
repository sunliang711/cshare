export interface PushInput {
  type: 'text' | 'binary' | 'files'
  text?: string
  file?: File | null
  files?: File[]
  ttl: number
  size: number
  filename?: string
  contentType: string
}

export interface ServerPushResult {
  key: string
  ttl: number
  size: number
  stored_size?: number
  type: string
  filename?: string
  file_count?: number
  expire_at?: number
}

export interface P2PStatusState {
  message: string
  active: boolean
  detail: string
  startedAt: number
  diagnostic: string
}

export interface P2PState {
  role: 'sender' | 'receiver'
  sessionID: string
  pc: RTCPeerConnection | null
  dc: RTCDataChannel | null
  input?: PushInput
  lastSeq: number
  stopped: boolean
  connected: boolean
  transferStarted: boolean
  transferDone: boolean
  fallbackStarted: boolean
  attempt: number
  attemptMode: string
  connectTimer: ReturnType<typeof setTimeout> | null
  upgradeTimer: ReturnType<typeof setTimeout> | null
  slowTimer: ReturnType<typeof setTimeout> | null
  statusTimer: ReturnType<typeof setInterval> | null
  pollController: AbortController | null
  p2pDiagnostic: string
  p2pDiagnosticSent: boolean
  startedAt: number
  pendingCandidates: RTCIceCandidateInit[]
  futureCandidates: Record<number, RTCIceCandidateInit[]>
  statusTarget?: 'push' | 'pull'
  statusMessage?: string
  statusActive?: boolean
  statusDetail?: string
  receiveMeta?: {
    kind: string
    type: string
    filename?: string
    content_type?: string
    size?: number
  } | null
  receiveChunks?: ArrayBuffer[]
  receivedSize?: number
}
