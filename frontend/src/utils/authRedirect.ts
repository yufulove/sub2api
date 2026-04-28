import { isImageSiteMode } from './siteMode'

type RedirectStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>
type SiteMode = 'image' | 'main'

interface PasswordResetRedirectRecord {
  path: string
  siteMode: SiteMode
  savedAt: number
}

const PASSWORD_RESET_REDIRECT_STORAGE_KEY = 'auth_password_reset_redirect'
const PASSWORD_RESET_REDIRECT_MAX_AGE_MS = 6 * 60 * 60 * 1000

export function getDefaultAuthenticatedRoute(isAdmin = false): string {
  if (isImageSiteMode()) {
    return '/studio/generate'
  }
  return isAdmin ? '/admin/dashboard' : '/dashboard'
}

export function buildRedirectQuery(
  requestedPath: unknown
): { redirect: string } | undefined {
  const redirect = sanitizeRedirectPath(requestedPath)
  return redirect ? { redirect } : undefined
}

export function sanitizeRedirectPath(path: unknown): string | null {
  if (typeof path !== 'string') {
    return null
  }

  const normalized = path.trim()
  if (!normalized) {
    return null
  }
  if (!normalized.startsWith('/')) {
    return null
  }
  if (normalized.startsWith('//')) {
    return null
  }
  if (normalized.includes('://')) {
    return null
  }
  if (normalized.includes('\n') || normalized.includes('\r')) {
    return null
  }
  return normalized
}

export function resolvePostAuthRedirect(
  requestedPath: unknown,
  isAdmin = false
): string {
  return sanitizeRedirectPath(requestedPath) || getDefaultAuthenticatedRoute(isAdmin)
}

export function resolveAuthRouteLocation(
  path: string,
  requestedPath: unknown
): { path: string; query?: { redirect: string } } {
  const query = buildRedirectQuery(requestedPath)
  return query ? { path, query } : { path }
}

export function rememberPasswordResetRedirect(
  requestedPath: unknown,
  options: {
    storage?: RedirectStorage | null
    savedAt?: number
  } = {}
): void {
  const storages = getRedirectStorages(options.storage)
  if (storages.length === 0) {
    return
  }

  const redirect = sanitizeRedirectPath(requestedPath)
  if (!redirect) {
    for (const storage of storages) {
      storage.removeItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY)
    }
    return
  }

  const record: PasswordResetRedirectRecord = {
    path: redirect,
    siteMode: getCurrentSiteMode(),
    savedAt: options.savedAt ?? Date.now()
  }

  const serializedRecord = JSON.stringify(record)
  for (const storage of storages) {
    storage.setItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY, serializedRecord)
  }
}

export function readPasswordResetRedirect(
  options: {
    storage?: RedirectStorage | null
    now?: number
    maxAgeMs?: number
  } = {}
): string | null {
  const storages = getRedirectStorages(options.storage)
  if (storages.length === 0) {
    return null
  }

  for (const storage of storages) {
    const raw = storage.getItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY)
    if (!raw) {
      continue
    }

    try {
      const parsed = JSON.parse(raw) as Partial<PasswordResetRedirectRecord>
      const redirect = sanitizeRedirectPath(parsed.path)
      const savedAt = Number(parsed.savedAt)
      const expectedSiteMode = getCurrentSiteMode()
      const maxAgeMs = options.maxAgeMs ?? PASSWORD_RESET_REDIRECT_MAX_AGE_MS
      const now = options.now ?? Date.now()

      if (
        !redirect ||
        !Number.isFinite(savedAt) ||
        parsed.siteMode !== expectedSiteMode ||
        now - savedAt > maxAgeMs
      ) {
        storage.removeItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY)
        continue
      }

      return redirect
    } catch {
      storage.removeItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY)
      continue
    }
  }

  return null
}

export function clearPasswordResetRedirect(storage?: RedirectStorage | null): void {
  const storages = getRedirectStorages(storage)
  if (storages.length === 0) {
    return
  }
  for (const target of storages) {
    target.removeItem(PASSWORD_RESET_REDIRECT_STORAGE_KEY)
  }
}

function getCurrentSiteMode(): SiteMode {
  return isImageSiteMode() ? 'image' : 'main'
}

function getRedirectStorages(storage?: RedirectStorage | null): RedirectStorage[] {
  if (storage !== undefined) {
    return storage ? [storage] : []
  }
  if (typeof window === 'undefined') {
    return []
  }

  const storages: RedirectStorage[] = []
  const pushStorage = (candidate: RedirectStorage | null): void => {
    if (candidate && !storages.includes(candidate)) {
      storages.push(candidate)
    }
  }

  pushStorage(readBrowserStorage(() => window.localStorage))
  pushStorage(readBrowserStorage(() => window.sessionStorage))

  return storages
}

function readBrowserStorage(getter: () => Storage | null | undefined): RedirectStorage | null {
  try {
    return getter() ?? null
  } catch {
    return null
  }
}
