import type { UsageLog } from '@/types'
import type { ImageStudioComposerPreset } from './imageStudioComposer'

const billedSizeFallbacks: Record<string, string> = {
  '1K': '1024x1024',
  '2K': '2048x2048',
  '4K': '4096x4096'
}

function readTrimmed(value: string | null | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

export function resolveUsageImagePrompt(
  record: Pick<UsageLog, 'image_prompt' | 'image_revised_prompt'>
): string | null {
  return readTrimmed(record.image_prompt) ?? readTrimmed(record.image_revised_prompt)
}

export function resolveUsageImageRequestedSize(
  record: Pick<UsageLog, 'image_requested_size' | 'image_size'>
): string | null {
  const requestedSize = readTrimmed(record.image_requested_size)
  if (requestedSize) {
    return requestedSize
  }

  const billedSize = readTrimmed(record.image_size)?.toUpperCase()
  return billedSize ? billedSizeFallbacks[billedSize] ?? null : null
}

export function buildUsageComposerPreset(
  record: Pick<
    UsageLog,
    'model' | 'image_prompt' | 'image_revised_prompt' | 'image_requested_size' | 'image_size'
  >
): ImageStudioComposerPreset | null {
  const preset: ImageStudioComposerPreset = {}
  const prompt = resolveUsageImagePrompt(record)
  const model = readTrimmed(record.model)
  const size = resolveUsageImageRequestedSize(record)

  if (prompt) {
    preset.prompt = prompt
  }
  if (model) {
    preset.model = model
  }
  if (size) {
    preset.size = size
  }

  return Object.keys(preset).length > 0 ? preset : null
}
