export type SiteMode = 'main' | 'image'

interface SiteHomeOptions {
  isAuthenticated?: boolean
  isAdmin?: boolean
}

const IMAGE_SUBDOMAIN_PREFIXES = ['img', 'image']
const IMAGE_SITE_ROUTE_PREFIX = '/studio'

interface NormalizedRelativeURL {
  pathname: string
  search: string
  hash: string
}

function splitConfiguredHosts(raw: string | undefined): string[] {
  if (!raw) {
    return []
  }
  return raw
    .split(',')
    .map((host) => host.trim().toLowerCase())
    .filter((host) => host.length > 0)
}

function normalizePath(path: string): string {
  const trimmed = path.trim()
  if (!trimmed) {
    return '/'
  }
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

function normalizeRelativeURL(path: string, fallbackPath = '/'): NormalizedRelativeURL {
  const candidate = normalizePath(path || fallbackPath)
  const normalized = new URL(candidate, 'https://sub2api.local')
  return {
    pathname: normalized.pathname || '/',
    search: normalized.search,
    hash: normalized.hash
  }
}

function stringifyRelativeURL(path: string, fallbackPath = '/'): string {
  const normalized = normalizeRelativeURL(path, fallbackPath)
  return `${normalized.pathname}${normalized.search}${normalized.hash}`
}

export function getCurrentSiteMode(): SiteMode {
  const forcedMode = String(import.meta.env.VITE_SITE_MODE || '').trim().toLowerCase()
  if (forcedMode === 'image') {
    return 'image'
  }
  if (forcedMode === 'main') {
    return 'main'
  }

  if (typeof window === 'undefined') {
    return 'main'
  }

  const hostname = window.location.hostname.trim().toLowerCase()
  if (!hostname) {
    return 'main'
  }

  const configuredHosts = splitConfiguredHosts(import.meta.env.VITE_IMAGE_SITE_HOSTS)
  if (configuredHosts.includes(hostname)) {
    return 'image'
  }

  const segments = hostname.split('.').filter((segment) => segment.length > 0)
  if (segments.length > 1 && IMAGE_SUBDOMAIN_PREFIXES.includes(segments[0])) {
    return 'image'
  }

  return 'main'
}

export function isImageSiteMode(): boolean {
  return getCurrentSiteMode() === 'image'
}

export function getMainSiteHomePath(options: SiteHomeOptions = {}): string {
  if (options.isAuthenticated) {
    return options.isAdmin ? '/admin/dashboard' : '/dashboard'
  }
  return '/home'
}

export function getImageSiteHomePath(options: SiteHomeOptions = {}): string {
  return options.isAuthenticated ? '/studio/generate' : '/studio'
}

export function getCurrentSiteHomePath(options: SiteHomeOptions = {}): string {
  return isImageSiteMode() ? getImageSiteHomePath(options) : getMainSiteHomePath(options)
}

export function isImageSiteRoutePath(path: string): boolean {
  const { pathname } = normalizeRelativeURL(path, IMAGE_SITE_ROUTE_PREFIX)
  return pathname === IMAGE_SITE_ROUTE_PREFIX || pathname.startsWith(`${IMAGE_SITE_ROUTE_PREFIX}/`)
}

export function resolveMainSiteURL(path = '/'): string {
  const normalizedPath = normalizeRelativeURL(path, '/')
  const relativeURL = stringifyRelativeURL(path, '/')
  const configuredBase = String(import.meta.env.VITE_MAIN_SITE_URL || '').trim()
  if (configuredBase) {
    return new URL(relativeURL, configuredBase.endsWith('/') ? configuredBase : `${configuredBase}/`).toString()
  }

  if (typeof window === 'undefined') {
    return relativeURL
  }

  const current = new URL(window.location.href)
  const segments = current.hostname.split('.').filter((segment) => segment.length > 0)
  if (segments.length > 1 && IMAGE_SUBDOMAIN_PREFIXES.includes(segments[0])) {
    current.hostname = segments.slice(1).join('.')
  }
  current.pathname = normalizedPath.pathname
  current.search = normalizedPath.search
  current.hash = normalizedPath.hash
  return current.toString()
}

export function resolveImageSiteURL(path = '/studio'): string {
  const normalizedPath = normalizeRelativeURL(path, IMAGE_SITE_ROUTE_PREFIX)
  const relativeURL = stringifyRelativeURL(path, IMAGE_SITE_ROUTE_PREFIX)
  const configuredBase = String(import.meta.env.VITE_IMAGE_SITE_URL || '').trim()
  if (configuredBase) {
    return new URL(relativeURL, configuredBase.endsWith('/') ? configuredBase : `${configuredBase}/`).toString()
  }

  if (typeof window === 'undefined') {
    const configuredHost = splitConfiguredHosts(import.meta.env.VITE_IMAGE_SITE_HOSTS)[0]
    if (!configuredHost) {
      return relativeURL
    }
    return `https://${configuredHost}${relativeURL}`
  }

  const current = new URL(window.location.href)
  const configuredHost = splitConfiguredHosts(import.meta.env.VITE_IMAGE_SITE_HOSTS)[0]
  if (configuredHost) {
    current.hostname = configuredHost
  }
  current.pathname = normalizedPath.pathname
  current.search = normalizedPath.search
  current.hash = normalizedPath.hash
  return current.toString()
}
