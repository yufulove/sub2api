import { useAuthStore } from '@/stores'
import { isImageSiteMode, resolveImageSiteURL, resolveMainSiteURL } from './siteMode'

const AUTH_BRIDGE_ROUTE = '/auth/bridge'
const AUTH_BRIDGE_SOURCE = 'sub2api-auth-bridge'
const AUTH_BRIDGE_TIMEOUT_MS = 2500

interface AttemptPeerAuthSyncOptions {
  timeoutMs?: number
}

interface CrossSiteAuthTransferPayload {
  accessToken: string
  refreshToken?: string
  expiresAt?: number
}

interface CrossSiteAuthTransferMessage {
  source: typeof AUTH_BRIDGE_SOURCE
  type: 'auth-transfer'
  requestId: string
  payload: CrossSiteAuthTransferPayload | null
}

let inflightPeerAuthSync: Promise<boolean> | null = null

export function getAuthBridgeRoute(): string {
  return AUTH_BRIDGE_ROUTE
}

function buildPeerBridgeURL(requestId: string): URL | null {
  if (typeof window === 'undefined') {
    return null
  }

  const currentOrigin = window.location.origin
  const query = new URLSearchParams({
    origin: currentOrigin,
    request_id: requestId
  }).toString()

  const peerURL = isImageSiteMode()
    ? resolveMainSiteURL(`${AUTH_BRIDGE_ROUTE}?${query}`)
    : resolveImageSiteURL(`${AUTH_BRIDGE_ROUTE}?${query}`)

  try {
    const normalized = new URL(peerURL, window.location.href)
    if (normalized.origin === currentOrigin) {
      return null
    }
    return normalized
  } catch {
    return null
  }
}

function isTransferMessage(value: unknown): value is CrossSiteAuthTransferMessage {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    candidate.source === AUTH_BRIDGE_SOURCE &&
    candidate.type === 'auth-transfer' &&
    typeof candidate.requestId === 'string'
  )
}

function persistTransferredTokens(payload: CrossSiteAuthTransferPayload): void {
  if (typeof window === 'undefined') {
    return
  }

  localStorage.setItem('auth_token', payload.accessToken)

  if (typeof payload.refreshToken === 'string' && payload.refreshToken.trim() !== '') {
    localStorage.setItem('refresh_token', payload.refreshToken)
  }

  if (typeof payload.expiresAt === 'number' && Number.isFinite(payload.expiresAt) && payload.expiresAt > 0) {
    localStorage.setItem('token_expires_at', String(payload.expiresAt))
  }
}

export async function attemptPeerAuthSync(
  options: AttemptPeerAuthSyncOptions = {}
): Promise<boolean> {
  if (typeof window === 'undefined') {
    return false
  }

  const authStore = useAuthStore()
  if (authStore.isAuthenticated) {
    return true
  }

  if (inflightPeerAuthSync) {
    return inflightPeerAuthSync
  }

  const requestId = `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  const bridgeURL = buildPeerBridgeURL(requestId)
  if (!bridgeURL) {
    return false
  }

  const timeoutMs =
    typeof options.timeoutMs === 'number' && Number.isFinite(options.timeoutMs) && options.timeoutMs > 0
      ? options.timeoutMs
      : AUTH_BRIDGE_TIMEOUT_MS

  inflightPeerAuthSync = new Promise<boolean>((resolve) => {
    const iframe = document.createElement('iframe')
    iframe.src = bridgeURL.toString()
    iframe.setAttribute('aria-hidden', 'true')
    iframe.tabIndex = -1
    iframe.style.display = 'none'

    let settled = false
    let timeoutId = 0

    const cleanup = () => {
      window.removeEventListener('message', handleMessage)
      if (timeoutId) {
        window.clearTimeout(timeoutId)
      }
      iframe.remove()
      inflightPeerAuthSync = null
    }

    const finish = (value: boolean) => {
      if (settled) {
        return
      }
      settled = true
      cleanup()
      resolve(value)
    }

    const handleMessage = async (event: MessageEvent) => {
      if (event.origin !== bridgeURL.origin || !isTransferMessage(event.data)) {
        return
      }

      const message = event.data
      if (timeoutId) {
        window.clearTimeout(timeoutId)
        timeoutId = 0
      }

      if (message.requestId !== requestId || !message.payload?.accessToken) {
        finish(false)
        return
      }

      try {
        persistTransferredTokens(message.payload)
        await authStore.setToken(message.payload.accessToken)
        finish(true)
      } catch (error) {
        console.error('Failed to apply peer auth sync:', error)
        finish(false)
      }
    }

    window.addEventListener('message', handleMessage)
    timeoutId = window.setTimeout(() => finish(false), timeoutMs)
    document.body.appendChild(iframe)
  })

  return inflightPeerAuthSync
}

export interface AuthBridgeSnapshot {
  accessToken: string
  refreshToken?: string
  expiresAt?: number
}

export function readAuthBridgeSnapshot(): AuthBridgeSnapshot | null {
  if (typeof window === 'undefined') {
    return null
  }

  const accessToken = String(localStorage.getItem('auth_token') || '').trim()
  if (!accessToken) {
    return null
  }

  const refreshToken = String(localStorage.getItem('refresh_token') || '').trim()
  const expiresAtRaw = Number(localStorage.getItem('token_expires_at') || '')

  return {
    accessToken,
    refreshToken: refreshToken || undefined,
    expiresAt: Number.isFinite(expiresAtRaw) && expiresAtRaw > 0 ? expiresAtRaw : undefined
  }
}

export function getAuthBridgeSource(): string {
  return AUTH_BRIDGE_SOURCE
}
