import { getModelsByPlatform } from './useModelWhitelist'

export interface PlatformModelOption {
  id: string
  type: string
  display_name: string
  created_at: string
}

const curatedModelDisplayNames: Record<string, string> = {
  'gpt-image-2': 'GPT Image 2',
  'gpt-5.4': 'GPT-5.4',
  'gpt-5.3-codex': 'GPT-5.3 Codex',
  'gpt-5.3-codex-spark': 'GPT-5.3 Codex Spark',
  'gpt-5.2': 'GPT-5.2',
  'gpt-5.2-codex': 'GPT-5.2 Codex',
  'gpt-5.1-codex-max': 'GPT-5.1 Codex Max',
  'gpt-5.1-codex': 'GPT-5.1 Codex',
  'gpt-5.1-codex-mini': 'GPT-5.1 Codex Mini',
  'gpt-5.1': 'GPT-5.1',
  'gpt-5': 'GPT-5',
  'claude-opus-4-5-20251101': 'Claude Opus 4.5',
  'claude-opus-4-6': 'Claude Opus 4.6',
  'claude-opus-4-7': 'Claude Opus 4.7',
  'claude-sonnet-4-6': 'Claude Sonnet 4.6',
  'claude-sonnet-4-5-20250929': 'Claude Sonnet 4.5',
  'claude-haiku-4-5-20251001': 'Claude Haiku 4.5',
  'gemini-3.1-flash-image': 'Gemini 3.1 Flash Image',
  'gemini-2.5-flash-image': 'Gemini 2.5 Flash Image',
  'gemini-2.0-flash': 'Gemini 2.0 Flash',
  'gemini-2.5-flash': 'Gemini 2.5 Flash',
  'gemini-2.5-pro': 'Gemini 2.5 Pro',
  'gemini-3-flash-preview': 'Gemini 3 Flash Preview',
  'gemini-3-pro-preview': 'Gemini 3 Pro Preview'
}

export function getModelDisplayName(id: string): string {
  return curatedModelDisplayNames[id] || inferModelDisplayName(id)
}

function inferModelDisplayName(id: string): string {
  const trimmed = id.trim()
  if (!trimmed) return id

  if (trimmed.startsWith('gpt-')) {
    return `GPT-${humanizeModelSuffix(trimmed.slice(4))}`
  }

  if (trimmed.startsWith('chatgpt-')) {
    return `ChatGPT ${humanizeModelSuffix(trimmed.slice(8))}`
  }

  return id
}

function humanizeModelSuffix(suffix: string): string {
  if (!suffix) return suffix

  return suffix
    .split('-')
    .filter(Boolean)
    .map((segment, index) => {
      if (index === 0) return segment
      if (/^\d+(\.\d+)?$/.test(segment)) return segment
      return segment.charAt(0).toUpperCase() + segment.slice(1)
    })
    .join(' ')
}

export function getPlatformModelOptions(platform: string): PlatformModelOption[] {
  return getModelsByPlatform(platform).map(id => ({
    id,
    type: 'model',
    display_name: getModelDisplayName(id),
    created_at: ''
  }))
}

export function normalizePlatformModelOptions(
  platform: string,
  models: Array<{ id: string; display_name?: string; type?: string; created_at?: string }>
): PlatformModelOption[] {
  const orderedIDs = getModelsByPlatform(platform)
  const orderedIDSet = new Set(orderedIDs)
  const byID = new Map<string, { id: string; display_name?: string; type?: string; created_at?: string }>()

  for (const model of models) {
    if (model?.id && !byID.has(model.id)) {
      byID.set(model.id, model)
    }
  }

  const normalized = orderedIDs
    .filter(id => byID.has(id))
    .map(id => {
      const model = byID.get(id)!
      return {
        id,
        type: model.type || 'model',
        display_name: model.display_name || getModelDisplayName(id),
        created_at: model.created_at || ''
      }
    })

  const passthroughModels = models
    .filter(model => model?.id && !orderedIDSet.has(model.id))
    .map(model => ({
      id: model.id,
      type: model.type || 'model',
      display_name: model.display_name || getModelDisplayName(model.id),
      created_at: model.created_at || ''
    }))

  const merged = [...normalized, ...passthroughModels]
  return merged.length > 0 ? merged : getPlatformModelOptions(platform)
}

export function pickDefaultTestModel(platform: string, models: Array<{ id: string }>): string {
  if (models.length === 0) return ''

  if (platform === 'openai') {
    for (const id of getModelsByPlatform('openai')) {
      if (models.some(model => model.id === id)) return id
    }
    return models[0].id
  }

  if (platform === 'gemini' || platform === 'antigravity') {
    return (
      models.find(model => model.id === 'gemini-3.1-flash-image')?.id ||
      models.find(model => model.id === 'gemini-2.5-flash-image')?.id ||
      models.find(model => model.id === 'gemini-2.0-flash')?.id ||
      models.find(model => model.id === 'gemini-2.5-flash')?.id ||
      models.find(model => model.id === 'gemini-2.5-pro')?.id ||
      models.find(model => model.id === 'gemini-3-flash-preview')?.id ||
      models.find(model => model.id === 'gemini-3-pro-preview')?.id ||
      models[0].id
    )
  }

  return models.find(model => model.id.includes('sonnet'))?.id || models[0].id
}
