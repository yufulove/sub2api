import type { PublicSettings } from '@/types'

type WalletSettings = Pick<PublicSettings, 'wallet_display_mode' | 'payment_balance_recharge_multiplier'>

const DEFAULT_RECHARGE_MULTIPLIER = 0.01

export function getWalletDisplayMode(settings?: Partial<WalletSettings> | null): 'usd' | 'cny' {
  return settings?.wallet_display_mode === 'usd' ? 'usd' : 'cny'
}

export function getWalletRechargeMultiplier(settings?: Partial<WalletSettings> | null): number {
  const value = Number(settings?.payment_balance_recharge_multiplier)
  return Number.isFinite(value) && value > 0 ? value : DEFAULT_RECHARGE_MULTIPLIER
}

export function toWalletDisplayAmount(
  internalUsd: number | null | undefined,
  settings?: Partial<WalletSettings> | null
): number {
  const numeric = Number(internalUsd)
  if (!Number.isFinite(numeric)) {
    return 0
  }
  if (getWalletDisplayMode(settings) === 'usd') {
    return numeric
  }
  return numeric / getWalletRechargeMultiplier(settings)
}

export function fromWalletDisplayAmount(
  displayAmount: number | null | undefined,
  settings?: Partial<WalletSettings> | null
): number {
  const numeric = Number(displayAmount)
  if (!Number.isFinite(numeric)) {
    return 0
  }
  if (getWalletDisplayMode(settings) === 'usd') {
    return numeric
  }
  return numeric * getWalletRechargeMultiplier(settings)
}

export function formatWalletMoney(
  displayAmount: number | null | undefined,
  settings?: Partial<WalletSettings> | null,
  options: Intl.NumberFormatOptions = {}
): string {
  const mode = getWalletDisplayMode(settings)
  const numeric = Number(displayAmount)
  const amount = Number.isFinite(numeric) ? numeric : 0
  const currency = mode === 'usd' ? 'USD' : 'CNY'
  const locale = mode === 'usd' ? 'en-US' : 'zh-CN'
  const formatter = new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    ...options
  })
  return formatter.format(amount)
}

export function formatWalletMoneyFromInternal(
  internalUsd: number | null | undefined,
  settings?: Partial<WalletSettings> | null,
  options: Intl.NumberFormatOptions = {}
): string {
  return formatWalletMoney(toWalletDisplayAmount(internalUsd, settings), settings, options)
}

export function formatWalletPairFromInternal(
  usedInternalUsd: number | null | undefined,
  limitInternalUsd: number | null | undefined,
  settings?: Partial<WalletSettings> | null,
  options: Intl.NumberFormatOptions = {}
): string {
  return `${formatWalletMoneyFromInternal(usedInternalUsd, settings, options)} / ${formatWalletMoneyFromInternal(limitInternalUsd, settings, options)}`
}

export function getWalletCurrencyPrefix(settings?: Partial<WalletSettings> | null): string {
  return getWalletDisplayMode(settings) === 'usd' ? '$' : '\u00A5'
}
