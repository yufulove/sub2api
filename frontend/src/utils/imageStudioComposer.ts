export const IMAGE_STUDIO_PROMPT_STORAGE_KEY = 'studio_image_prompt'
export const IMAGE_STUDIO_MODEL_STORAGE_KEY = 'studio_image_model'
export const IMAGE_STUDIO_SIZE_STORAGE_KEY = 'studio_image_size'

export interface ImageStudioComposerPreset {
  prompt?: string
  model?: string
  size?: string
}

function getStorage(): Storage | null {
  if (typeof window === 'undefined') {
    return null
  }

  return window.localStorage
}

export function readStoredPrompt(fallback: string): string {
  const storage = getStorage()
  if (!storage) {
    return fallback
  }

  const raw = storage.getItem(IMAGE_STUDIO_PROMPT_STORAGE_KEY)
  return raw == null ? fallback : raw
}

export function persistPromptDraft(value: string): void {
  const storage = getStorage()
  if (!storage) {
    return
  }

  storage.setItem(IMAGE_STUDIO_PROMPT_STORAGE_KEY, value)
}

export function readStoredChoice(storageKey: string, allowedValues: string[], fallback: string): string {
  const storage = getStorage()
  if (!storage) {
    return fallback
  }

  const raw = storage.getItem(storageKey)
  return raw && allowedValues.includes(raw) ? raw : fallback
}

export function persistStoredChoice(storageKey: string, value: string): void {
  const storage = getStorage()
  if (!storage) {
    return
  }

  storage.setItem(storageKey, value)
}

export function applyComposerPreset(preset: ImageStudioComposerPreset): void {
  const storage = getStorage()
  if (!storage) {
    return
  }

  if (typeof preset.prompt === 'string') {
    storage.setItem(IMAGE_STUDIO_PROMPT_STORAGE_KEY, preset.prompt)
  }
  if (typeof preset.model === 'string' && preset.model.trim() !== '') {
    storage.setItem(IMAGE_STUDIO_MODEL_STORAGE_KEY, preset.model)
  }
  if (typeof preset.size === 'string' && preset.size.trim() !== '') {
    storage.setItem(IMAGE_STUDIO_SIZE_STORAGE_KEY, preset.size)
  }
}

function readSingleQueryString(value: unknown): string | null {
  return typeof value === 'string' ? value : null
}

export function buildComposerPresetQuery(preset: ImageStudioComposerPreset): Record<string, string> {
  const query: Record<string, string> = {}

  if (typeof preset.prompt === 'string' && preset.prompt.trim() !== '') {
    query.prompt = preset.prompt.trim()
  }
  if (typeof preset.model === 'string' && preset.model.trim() !== '') {
    query.model = preset.model.trim()
  }
  if (typeof preset.size === 'string' && preset.size.trim() !== '') {
    query.size = preset.size.trim()
  }

  return query
}

export function readComposerPresetFromQuery(
  query: Record<string, unknown>,
  allowedModels: string[],
  allowedSizes: string[]
): ImageStudioComposerPreset | null {
  const preset: ImageStudioComposerPreset = {}

  const prompt = readSingleQueryString(query.prompt)
  if (prompt && prompt.trim() !== '') {
    preset.prompt = prompt.trim()
  }

  const model = readSingleQueryString(query.model)
  if (model && allowedModels.includes(model)) {
    preset.model = model
  }

  const size = readSingleQueryString(query.size)
  if (size && allowedSizes.includes(size)) {
    preset.size = size
  }

  return Object.keys(preset).length > 0 ? preset : null
}
