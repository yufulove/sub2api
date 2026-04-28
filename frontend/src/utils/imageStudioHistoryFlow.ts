import type { StudioSessionGeneration } from '@/stores/imageStudio'
import type { UsageLog } from '@/types'
import type { ImageStudioComposerPreset } from './imageStudioComposer'
import { resolveUsageImagePrompt, resolveUsageImageRequestedSize } from './imageStudioUsage'

export interface ImageStudioHistoryFlow {
  id: string
  prompt: string
  revisedPrompt: string | null
  model: string | null
  size: string | null
  latestCreatedAt: number
  firstCreatedAt: number
  usageRequestCount: number
  usageImageCount: number
  totalCost: number
  sessionRenderCount: number
  sessionCards: StudioSessionGeneration[]
  usageRecords: UsageLog[]
  keyNames: string[]
  composerPreset: ImageStudioComposerPreset
}

interface FlowAccumulator {
  id: string
  prompt: string
  revisedPrompt: string | null
  model: string | null
  size: string | null
  latestCreatedAt: number
  firstCreatedAt: number
  usageRequestCount: number
  usageImageCount: number
  totalCost: number
  sessionRenderCount: number
  sessionCards: StudioSessionGeneration[]
  usageRecords: UsageLog[]
  keyNames: Set<string>
}

function readTrimmed(value: string | null | undefined): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function normalizePromptKey(value: string): string {
  return value.trim().replace(/\s+/g, ' ').toLowerCase()
}

function normalizeKeyPart(value: string | null): string {
  return (value ?? '').trim().toLowerCase()
}

function buildFlowKey(prompt: string, model: string | null, size: string | null): string {
  return [normalizePromptKey(prompt), normalizeKeyPart(model), normalizeKeyPart(size)].join('::')
}

function ensureAccumulator(
  groups: Map<string, FlowAccumulator>,
  prompt: string,
  model: string | null,
  size: string | null
): FlowAccumulator {
  const key = buildFlowKey(prompt, model, size)
  let group = groups.get(key)
  if (!group) {
    group = {
      id: key,
      prompt,
      revisedPrompt: null,
      model,
      size,
      latestCreatedAt: 0,
      firstCreatedAt: Number.POSITIVE_INFINITY,
      usageRequestCount: 0,
      usageImageCount: 0,
      totalCost: 0,
      sessionRenderCount: 0,
      sessionCards: [],
      usageRecords: [],
      keyNames: new Set<string>()
    }
    groups.set(key, group)
  }
  return group
}

function maybePromoteLatest(
  group: FlowAccumulator,
  createdAt: number,
  prompt: string,
  model: string | null,
  size: string | null,
  revisedPrompt: string | null
): void {
  if (createdAt < group.latestCreatedAt) {
    return
  }

  group.prompt = prompt
  group.model = model
  group.size = size
  if (revisedPrompt) {
    group.revisedPrompt = revisedPrompt
  }
}

export function buildImageStudioHistoryFlows(
  usageRecords: UsageLog[],
  sessionCards: StudioSessionGeneration[]
): ImageStudioHistoryFlow[] {
  const groups = new Map<string, FlowAccumulator>()

  for (const record of usageRecords) {
    const prompt = resolveUsageImagePrompt(record)
    if (!prompt) {
      continue
    }

    const model = readTrimmed(record.model)
    const size = resolveUsageImageRequestedSize(record) ?? readTrimmed(record.image_size)
    const createdAt = new Date(record.created_at).getTime()
    const revisedPrompt = readTrimmed(record.image_revised_prompt)
    const group = ensureAccumulator(groups, prompt, model, size)

    group.usageRecords.push(record)
    group.usageRequestCount += 1
    group.usageImageCount += Math.max(Number(record.image_count) || 0, 0)
    group.totalCost += Number(record.actual_cost) || 0
    group.latestCreatedAt = Math.max(group.latestCreatedAt, createdAt)
    group.firstCreatedAt = Math.min(group.firstCreatedAt, createdAt)
    maybePromoteLatest(group, createdAt, prompt, model, size, revisedPrompt)

    if (revisedPrompt && revisedPrompt !== prompt) {
      group.revisedPrompt = revisedPrompt
    }
    if (record.api_key?.name) {
      group.keyNames.add(record.api_key.name)
    }
  }

  for (const card of sessionCards) {
    const prompt = readTrimmed(card.prompt)
    if (!prompt) {
      continue
    }

    const model = readTrimmed(card.model)
    const size = readTrimmed(card.size)
    const createdAt = card.created * 1000
    const revisedPrompt = readTrimmed(card.revisedPrompt)
    const group = ensureAccumulator(groups, prompt, model, size)

    group.sessionCards.push(card)
    group.sessionRenderCount += 1
    group.latestCreatedAt = Math.max(group.latestCreatedAt, createdAt)
    group.firstCreatedAt = Math.min(group.firstCreatedAt, createdAt)
    maybePromoteLatest(group, createdAt, prompt, model, size, revisedPrompt)

    if (revisedPrompt && revisedPrompt !== prompt) {
      group.revisedPrompt = revisedPrompt
    }
    if (card.keyName) {
      group.keyNames.add(card.keyName)
    }
  }

  return Array.from(groups.values())
    .map((group) => ({
      id: group.id,
      prompt: group.prompt,
      revisedPrompt: group.revisedPrompt,
      model: group.model,
      size: group.size,
      latestCreatedAt: group.latestCreatedAt,
      firstCreatedAt: Number.isFinite(group.firstCreatedAt) ? group.firstCreatedAt : group.latestCreatedAt,
      usageRequestCount: group.usageRequestCount,
      usageImageCount: group.usageImageCount,
      totalCost: group.totalCost,
      sessionRenderCount: group.sessionRenderCount,
      sessionCards: group.sessionCards.sort((left, right) => right.created - left.created),
      usageRecords: group.usageRecords.sort(
        (left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
      ),
      keyNames: Array.from(group.keyNames).sort(),
      composerPreset: {
        prompt: group.prompt,
        ...(group.model ? { model: group.model } : {}),
        ...(group.size ? { size: group.size } : {})
      }
    }))
    .sort((left, right) => right.latestCreatedAt - left.latestCreatedAt)
}
