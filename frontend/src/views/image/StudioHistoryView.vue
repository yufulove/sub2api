<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { keysAPI, usageAPI } from '@/api'
import StudioShell from '@/components/image/StudioShell.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore, useImageStudioStore } from '@/stores'
import type { StudioSessionGeneration } from '@/stores/imageStudio'
import type { ApiKey, UsageLog, UsageQueryParams } from '@/types'
import { buildComposerPresetQuery, type ImageStudioComposerPreset } from '@/utils/imageStudioComposer'
import {
  buildImageStudioHistoryFlows,
  type ImageStudioHistoryFlow
} from '@/utils/imageStudioHistoryFlow'
import {
  buildUsageComposerPreset,
  resolveUsageImagePrompt,
  resolveUsageImageRequestedSize
} from '@/utils/imageStudioUsage'
import { formatWalletMoneyFromInternal } from '@/utils/walletDisplay'

type HistoryRangePreset = '7d' | '30d' | '90d' | 'all'

const MAX_HISTORY_ITEMS = 60
const MAX_FETCH_PAGES = 12

const router = useRouter()
const appStore = useAppStore()
const imageStudioStore = useImageStudioStore()
const { copyToClipboard } = useClipboard()
const rangePresets: HistoryRangePreset[] = ['7d', '30d', '90d', 'all']

const apiKeys = ref<ApiKey[]>([])
const selectedApiKeyId = ref<number | null>(null)
const rangePreset = ref<HistoryRangePreset>('30d')
const usageRecords = ref<UsageLog[]>([])
const loading = ref(false)
const loadingKeys = ref(false)
const errorMessage = ref('')

let pendingController: AbortController | null = null

const selectedKey = computed(() => apiKeys.value.find((key) => key.id === selectedApiKeyId.value) ?? null)
const sessionCards = computed(() => imageStudioStore.sessionGenerations)
const filteredSessionCards = computed(() => {
  const keyName = selectedKey.value?.name?.trim()
  if (!keyName) {
    return sessionCards.value
  }
  return sessionCards.value.filter((card) => card.keyName.trim() === keyName)
})
const isSessionHydrating = computed(() => imageStudioStore.isHydrating)
const historyFlows = computed(() => buildImageStudioHistoryFlows(usageRecords.value, filteredSessionCards.value))

const keyOptions = computed<SelectOption[]>(() => [
  { value: null, label: 'All API keys' },
  ...apiKeys.value.map((key) => ({
    value: key.id,
    label: key.group?.name ? `${key.name} / ${key.group.name}` : `${key.name} / No group`
  }))
])

const usageRequestCount = computed(() => usageRecords.value.length)
const usageImageCount = computed(() =>
  usageRecords.value.reduce((total, item) => total + Math.max(Number(item.image_count) || 0, 0), 0)
)
const usageTotalCost = computed(() =>
  usageRecords.value.reduce((total, item) => total + (Number(item.actual_cost) || 0), 0)
)
const flowGroupCount = computed(() => historyFlows.value.length)

watch(rangePreset, () => {
  loadUsageHistory()
})

watch(selectedApiKeyId, () => {
  loadUsageHistory()
})

function setRangePreset(value: HistoryRangePreset) {
  rangePreset.value = value
}

async function loadKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 200)
    apiKeys.value = response.items
  } catch {
    appStore.showError('Failed to load API keys for history filters.')
  } finally {
    loadingKeys.value = false
  }
}

async function loadUsageHistory() {
  pendingController?.abort()
  const controller = new AbortController()
  pendingController = controller
  loading.value = true
  errorMessage.value = ''

  try {
    const range = resolveDateRange(rangePreset.value)
    const baseParams: UsageQueryParams = {
      page_size: 100
    }

    if (selectedApiKeyId.value != null) {
      baseParams.api_key_id = selectedApiKeyId.value
    }
    if (range.startDate) {
      baseParams.start_date = range.startDate
    }
    if (range.endDate) {
      baseParams.end_date = range.endDate
    }

    const collected: UsageLog[] = []
    let currentPage = 1
    let totalPages = 1

    while (currentPage <= totalPages && currentPage <= MAX_FETCH_PAGES && collected.length < MAX_HISTORY_ITEMS) {
      const response = await usageAPI.query(
        {
          ...baseParams,
          page: currentPage
        },
        {
          signal: controller.signal
        }
      )

      if (controller.signal.aborted) {
        return
      }

      totalPages = response.pages || 1
      const imageRows = (response.items || []).filter((item) => Number(item.image_count) > 0)
      collected.push(...imageRows)

      if ((response.items || []).length === 0) {
        break
      }
      currentPage += 1
    }

    usageRecords.value = collected
      .sort((left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime())
      .slice(0, MAX_HISTORY_ITEMS)
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    errorMessage.value = 'Failed to load image usage history.'
    appStore.showError(errorMessage.value)
  } finally {
    if (pendingController === controller) {
      pendingController = null
    }
    loading.value = false
  }
}

function resolveDateRange(preset: HistoryRangePreset): { startDate?: string; endDate?: string } {
  if (preset === 'all') {
    return {}
  }

  const now = new Date()
  const start = new Date(now)

  if (preset === '7d') {
    start.setDate(start.getDate() - 6)
  } else if (preset === '30d') {
    start.setDate(start.getDate() - 29)
  } else {
    start.setDate(start.getDate() - 89)
  }

  return {
    startDate: formatDateOnly(start),
    endDate: formatDateOnly(now)
  }
}

function formatDateOnly(value: Date): string {
  return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`
}

function formatTimestamp(value: string): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

function formatFlowTimestamp(value: number): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

function formatSessionTimestamp(value: number): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value * 1000))
}

function formatCost(value: number | null | undefined, maximumFractionDigits = 4): string {
  return formatWalletMoneyFromInternal(value, appStore.cachedPublicSettings, {
    minimumFractionDigits: 2,
    maximumFractionDigits
  })
}

function downloadCard(card: StudioSessionGeneration, index: number) {
  const link = document.createElement('a')
  link.href = card.imageSrc
  link.download = `${sanitizeFilename(card.model)}-${card.size}-${card.created}-${index + 1}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function reuseCardPrompt(card: StudioSessionGeneration) {
  openComposerPreset({
    prompt: card.prompt,
    model: card.model,
    size: card.size
  })
}

function copyCardPrompt(card: StudioSessionGeneration) {
  void copyToClipboard(card.prompt, 'Prompt copied to clipboard')
}

function copyCardStudioLink(card: StudioSessionGeneration) {
  copyComposerPresetLink(
    {
      prompt: card.prompt,
      model: card.model,
      size: card.size
    },
    'Studio link copied to clipboard'
  )
}

function openUsageRecordInStudio(item: UsageLog) {
  const preset = buildUsageComposerPreset(item)
  if (!preset) {
    appStore.showError('Prompt metadata is not available for this history record')
    return
  }

  openComposerPreset(preset)
}

function copyUsageRecordPrompt(item: UsageLog) {
  const prompt = resolveUsageImagePrompt(item)
  if (!prompt) {
    appStore.showError('Prompt metadata is not available for this history record')
    return
  }

  void copyToClipboard(prompt, 'Prompt copied to clipboard')
}

function copyUsageRecordStudioLink(item: UsageLog) {
  const preset = buildUsageComposerPreset(item)
  if (!preset) {
    appStore.showError('Prompt metadata is not available for this history record')
    return
  }

  copyComposerPresetLink(preset, 'Studio link copied to clipboard')
}

function openFlowInStudio(flow: ImageStudioHistoryFlow) {
  openComposerPreset(flow.composerPreset)
}

function copyFlowPrompt(flow: ImageStudioHistoryFlow) {
  void copyToClipboard(flow.prompt, 'Prompt copied to clipboard')
}

function copyFlowStudioLink(flow: ImageStudioHistoryFlow) {
  copyComposerPresetLink(flow.composerPreset, 'Studio link copied to clipboard')
}

function resolveUsageRecordPrompt(item: UsageLog): string | null {
  return resolveUsageImagePrompt(item)
}

function resolveUsageRecordRequestedSize(item: UsageLog): string | null {
  return resolveUsageImageRequestedSize(item)
}

function hasUsageRecordComposerPreset(item: UsageLog): boolean {
  return buildUsageComposerPreset(item) != null
}

function openComposerPreset(preset: ImageStudioComposerPreset) {
  void router.push({
    path: '/studio/generate',
    query: buildComposerPresetQuery(preset)
  })
}

function copyComposerPresetLink(preset: ImageStudioComposerPreset, successMessage: string) {
  const target = router.resolve({
    path: '/studio/generate',
    query: buildComposerPresetQuery(preset)
  })

  void copyToClipboard(new URL(target.href, window.location.origin).toString(), successMessage)
}

function sanitizeFilename(value: string): string {
  return value.replace(/[^a-zA-Z0-9-_]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '')
}

function isAbortError(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }
  const payload = error as { name?: string; code?: string }
  return payload.name === 'AbortError' || payload.code === 'ERR_CANCELED'
}

onMounted(async () => {
  await imageStudioStore.ensureHydrated()
  await loadKeys()
  await loadUsageHistory()
})

onUnmounted(() => {
  pendingController?.abort()
})
</script>

<template>
  <main class="history-shell">
    <StudioShell max-width="1120px" />

    <section class="history-hero">
      <div>
        <p class="eyebrow">Image History</p>
        <h1>Review recent renders, grouped flows, and billed image activity</h1>
        <p class="copy">
          Use session cards for the images you just generated, then move up one level into grouped prompt
          flows before dropping into the raw usage ledger.
        </p>
      </div>

      <div class="hero-actions">
        <RouterLink class="ghost-link" to="/studio/generate">Back to generate</RouterLink>
        <button class="ghost-button" type="button" :disabled="loading" @click="loadUsageHistory">
          Refresh history
        </button>
      </div>
    </section>

    <section class="stats-grid">
      <article class="stat-card">
        <span>Session Renders</span>
        <strong>{{ filteredSessionCards.length }}</strong>
      </article>
      <article class="stat-card">
        <span>Prompt Flows</span>
        <strong>{{ flowGroupCount }}</strong>
      </article>
      <article class="stat-card">
        <span>Usage Requests</span>
        <strong>{{ usageRequestCount }}</strong>
      </article>
      <article class="stat-card">
        <span>Total Images</span>
        <strong>{{ usageImageCount }}</strong>
      </article>
      <article class="stat-card">
        <span>Billed Cost</span>
        <strong>{{ formatCost(usageTotalCost, 4) }}</strong>
      </article>
    </section>

    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Session</p>
          <h2>Current browser session renders</h2>
          <p class="copy">
            These cards stay tied to the current signed-in account in this browser, so you can move between
            generate and history without losing what you just made.
          </p>
        </div>
        <button
          v-if="filteredSessionCards.length > 0"
          class="ghost-button"
          type="button"
          @click="imageStudioStore.clearSessionGenerations()"
        >
          Clear session renders
        </button>
      </div>

      <div v-if="isSessionHydrating && filteredSessionCards.length === 0" class="empty-state">
        <strong>Restoring browser session renders</strong>
        <p>Saved image cards for this account are loading from local browser storage.</p>
      </div>

      <div v-else-if="filteredSessionCards.length === 0" class="empty-state">
        <strong>No session renders yet</strong>
        <p>Generate an image in this account session and it will appear here immediately.</p>
      </div>

      <div v-else class="session-grid">
        <article
          v-for="(card, index) in filteredSessionCards"
          :key="card.id"
          class="session-card"
        >
          <div class="session-image">
            <img :src="card.imageSrc" :alt="card.prompt" loading="lazy" />
          </div>
          <div class="session-body">
            <div class="meta-row">
              <span>{{ formatSessionTimestamp(card.created) }}</span>
              <span>{{ card.size }}</span>
            </div>
            <p class="prompt-text">{{ card.prompt }}</p>
            <p v-if="card.revisedPrompt && card.revisedPrompt !== card.prompt" class="secondary-text">
              Revised prompt: {{ card.revisedPrompt }}
            </p>
            <div class="footer-row">
              <div class="tag-row">
                <span>{{ card.model }}</span>
                <span>{{ card.keyName }}</span>
              </div>
              <div class="card-actions">
                <button type="button" class="inline-button" @click="reuseCardPrompt(card)">
                  Open in studio
                </button>
                <button type="button" class="inline-button" @click="copyCardPrompt(card)">
                  Copy prompt
                </button>
                <button type="button" class="inline-button" @click="copyCardStudioLink(card)">
                  Copy studio link
                </button>
                <button type="button" class="inline-button" @click="downloadCard(card, index)">
                  Download
                </button>
              </div>
            </div>
          </div>
        </article>
      </div>
    </section>

    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Flows</p>
          <h2>Prompt-grouped image series</h2>
          <p class="copy">
            This gallery view clusters the same prompt, model, and size into one reusable creative thread,
            so you can continue a series without scanning one ledger row at a time.
          </p>
        </div>
      </div>

      <div v-if="historyFlows.length === 0" class="empty-state">
        <strong>No prompt flows yet</strong>
        <p>Generate images or accumulate billed prompt metadata and grouped series will appear here.</p>
      </div>

      <div v-else class="flow-grid">
        <article
          v-for="flow in historyFlows"
          :key="flow.id"
          class="flow-card"
        >
          <div :class="['flow-preview', flow.sessionCards.length === 0 && 'flow-preview-empty']">
            <div v-if="flow.sessionCards.length > 0" class="flow-preview-grid">
              <img
                v-for="card in flow.sessionCards.slice(0, 4)"
                :key="card.id"
                :src="card.imageSrc"
                :alt="card.prompt"
                loading="lazy"
              />
            </div>
            <div v-else class="flow-preview-placeholder">
              <strong>No local preview</strong>
              <p>This browser does not have a recent render card for this series yet.</p>
            </div>
          </div>

          <div class="flow-body">
            <div class="meta-row">
              <span>{{ formatFlowTimestamp(flow.latestCreatedAt) }}</span>
              <span>{{ flow.model || 'Unknown model' }}</span>
            </div>

            <p class="prompt-text">{{ flow.prompt }}</p>
            <p v-if="flow.revisedPrompt && flow.revisedPrompt !== flow.prompt" class="secondary-text">
              Latest revised prompt: {{ flow.revisedPrompt }}
            </p>

            <div class="tag-row flow-tags">
              <span>{{ flow.size || 'Unknown size' }}</span>
              <span>{{ flow.usageRequestCount }} billed request{{ flow.usageRequestCount === 1 ? '' : 's' }}</span>
              <span>{{ flow.usageImageCount }} billed image{{ flow.usageImageCount === 1 ? '' : 's' }}</span>
              <span>{{ flow.sessionRenderCount }} local preview{{ flow.sessionRenderCount === 1 ? '' : 's' }}</span>
            </div>

            <div class="footer-row flow-footer">
              <div class="flow-summary">
                <strong>{{ formatCost(flow.totalCost, 4) }}</strong>
                <span>{{ flow.keyNames.join(' / ') || 'No key metadata' }}</span>
              </div>
              <div class="card-actions">
                <button type="button" class="inline-button" @click="openFlowInStudio(flow)">
                  Open in studio
                </button>
                <button type="button" class="inline-button" @click="copyFlowPrompt(flow)">
                  Copy prompt
                </button>
                <button type="button" class="inline-button" @click="copyFlowStudioLink(flow)">
                  Copy studio link
                </button>
              </div>
            </div>
          </div>
        </article>
      </div>
    </section>

    <section class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Ledger</p>
          <h2>Billed image requests</h2>
          <p class="copy">
            These records come from the durable usage log. They show request metadata, billed cost, and image count
            even when the original render card is no longer in the local session.
          </p>
        </div>
      </div>

      <div class="filter-bar">
        <Select
          v-model="selectedApiKeyId"
          :options="keyOptions"
          :disabled="loadingKeys"
        />
        <div class="preset-row">
          <button
            v-for="preset in rangePresets"
            :key="preset"
            type="button"
            :class="['preset-button', rangePreset === preset && 'preset-button-active']"
            @click="setRangePreset(preset)"
          >
            {{ preset }}
          </button>
        </div>
      </div>

      <div v-if="errorMessage" class="alert-box">
        {{ errorMessage }}
      </div>

      <div v-if="loading" class="ledger-list">
        <div class="ledger-card ledger-skeleton" />
        <div class="ledger-card ledger-skeleton" />
        <div class="ledger-card ledger-skeleton" />
      </div>

      <div v-else-if="usageRecords.length === 0" class="empty-state">
        <strong>No image usage records</strong>
        <p>There are no `image_count > 0` records for the current filter yet.</p>
      </div>

      <div v-else class="ledger-list">
        <article
          v-for="item in usageRecords"
          :key="item.id"
          class="ledger-card"
        >
          <div class="ledger-main">
            <div class="meta-row">
              <span>{{ formatTimestamp(item.created_at) }}</span>
              <span>{{ item.api_key?.name || 'Unknown key' }}</span>
            </div>
            <h3>{{ item.model }}</h3>
            <p class="secondary-text">
              {{ item.image_count }} image{{ item.image_count > 1 ? 's' : '' }} /
              {{ resolveUsageRecordRequestedSize(item) || item.image_size || '2K' }}
              <template v-if="item.image_size && resolveUsageRecordRequestedSize(item) !== item.image_size">
                / billed as {{ item.image_size }}
              </template>
            </p>
            <p v-if="resolveUsageRecordPrompt(item)" class="prompt-text">
              {{ resolveUsageRecordPrompt(item) }}
            </p>
            <p
              v-if="
                item.image_revised_prompt &&
                resolveUsageRecordPrompt(item) &&
                item.image_revised_prompt !== resolveUsageRecordPrompt(item)
              "
              class="secondary-text"
            >
              Revised prompt: {{ item.image_revised_prompt }}
            </p>
            <p v-else-if="!resolveUsageRecordPrompt(item)" class="secondary-text">
              Prompt metadata is not available for this older history record yet.
            </p>
            <div v-if="hasUsageRecordComposerPreset(item)" class="card-actions ledger-actions">
              <button type="button" class="inline-button" @click="openUsageRecordInStudio(item)">
                Open in studio
              </button>
              <button type="button" class="inline-button" @click="copyUsageRecordPrompt(item)">
                Copy prompt
              </button>
              <button type="button" class="inline-button" @click="copyUsageRecordStudioLink(item)">
                Copy studio link
              </button>
            </div>
          </div>
          <div class="ledger-side">
            <strong>{{ formatCost(item.actual_cost, 4) }}</strong>
            <span>{{ item.group?.name || 'No group' }}</span>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>

<style scoped>
.history-shell {
  --paper: #f4eee5;
  --panel: rgba(255, 255, 255, 0.78);
  --ink: #142720;
  --muted: rgba(20, 39, 32, 0.7);
  --line: rgba(20, 39, 32, 0.12);
  --accent: #b14b2f;
  min-height: 100vh;
  padding: 40px 24px 72px;
  background:
    radial-gradient(circle at top right, rgba(230, 138, 84, 0.16), transparent 28rem),
    linear-gradient(180deg, #faf6ef 0%, var(--paper) 100%);
  color: var(--ink);
  font-family: "Space Grotesk", "Noto Sans SC", sans-serif;
}

.history-hero,
.stats-grid,
.panel {
  width: min(1120px, 100%);
  margin: 0 auto;
}

.history-hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
}

.eyebrow {
  margin: 0 0 12px;
  font-size: 0.78rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: var(--accent);
}

.history-hero h1,
.panel-header h2 {
  margin: 0;
  line-height: 0.96;
  letter-spacing: -0.05em;
}

.history-hero h1 {
  font-size: clamp(2.4rem, 6vw, 4.1rem);
}

.copy,
.secondary-text,
.empty-state p {
  color: var(--muted);
}

.copy {
  max-width: 44rem;
  margin: 16px 0 0;
  line-height: 1.72;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.ghost-link,
.ghost-button,
.inline-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 42px;
  padding: 0 16px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.68);
  color: inherit;
  text-decoration: none;
  font: inherit;
  font-weight: 700;
  transition: transform 160ms ease, border-color 160ms ease;
}

.ghost-link:hover,
.ghost-button:hover,
.inline-button:hover,
.preset-button:hover {
  transform: translateY(-1px);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 14px;
  margin-top: 24px;
}

.stat-card,
.panel {
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel);
  backdrop-filter: blur(14px);
  box-shadow: 0 20px 60px rgba(20, 39, 32, 0.08);
}

.stat-card {
  padding: 18px 20px;
}

.stat-card span {
  display: block;
  font-size: 0.78rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(20, 39, 32, 0.56);
}

.stat-card strong {
  display: block;
  margin-top: 8px;
  font-size: 1.15rem;
}

.panel {
  padding: 28px;
  margin-top: 20px;
}

.panel-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
}

.panel-header h2 {
  font-size: clamp(1.8rem, 4vw, 2.6rem);
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(240px, 320px) 1fr;
  gap: 14px;
  align-items: center;
  margin-top: 22px;
}

.preset-row {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.preset-button {
  min-height: 42px;
  padding: 0 16px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.7);
  color: inherit;
  font: inherit;
  font-weight: 700;
  transition: transform 160ms ease, border-color 160ms ease;
}

.preset-button-active {
  border-color: rgba(177, 75, 47, 0.4);
  background: rgba(177, 75, 47, 0.1);
}

:deep(.select-trigger) {
  min-height: 48px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.86);
  border-color: rgba(20, 39, 32, 0.12);
}

.session-grid,
.ledger-list,
.flow-grid {
  display: grid;
  gap: 14px;
  margin-top: 22px;
}

.session-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.session-card,
.flow-card,
.ledger-card,
.empty-state {
  border: 1px solid var(--line);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.74);
}

.session-card {
  overflow: hidden;
}

.session-image,
.flow-preview {
  aspect-ratio: 1 / 1;
  background:
    radial-gradient(circle at top left, rgba(177, 75, 47, 0.14), transparent 55%),
    rgba(20, 39, 32, 0.04);
}

.session-image img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.flow-card {
  display: grid;
  grid-template-columns: minmax(180px, 240px) 1fr;
  overflow: hidden;
}

.flow-preview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  height: 100%;
}

.flow-preview-grid img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.flow-preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.flow-preview-placeholder {
  max-width: 15rem;
  text-align: center;
}

.flow-preview-placeholder strong {
  display: block;
}

.flow-preview-placeholder p {
  margin: 8px 0 0;
  color: var(--muted);
  line-height: 1.6;
}

.session-body,
.flow-body {
  padding: 18px 18px 20px;
}

.meta-row,
.footer-row,
.ledger-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.meta-row {
  font-size: 0.8rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(20, 39, 32, 0.56);
}

.prompt-text {
  margin: 14px 0 0;
  line-height: 1.7;
  font-size: 0.98rem;
}

.footer-row {
  margin-top: 16px;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-row span {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(20, 39, 32, 0.07);
  font-size: 0.78rem;
}

.card-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.inline-button {
  min-height: 38px;
  padding: 0 14px;
}

.ledger-card {
  padding: 18px 20px;
}

.flow-tags {
  margin-top: 16px;
}

.flow-footer {
  align-items: flex-end;
}

.flow-summary {
  display: grid;
  gap: 6px;
}

.flow-summary strong {
  font-size: 1rem;
}

.flow-summary span {
  color: var(--muted);
  font-size: 0.9rem;
}

.ledger-main h3 {
  margin: 12px 0 0;
  font-size: 1.04rem;
}

.ledger-main .secondary-text {
  margin: 8px 0 0;
}

.ledger-actions {
  margin-top: 14px;
  justify-content: flex-start;
}

.ledger-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  text-align: right;
}

.ledger-side strong {
  font-size: 1rem;
}

.ledger-side span {
  color: var(--muted);
  font-size: 0.9rem;
}

.ledger-skeleton {
  min-height: 102px;
  background:
    linear-gradient(110deg, rgba(255, 255, 255, 0.25) 8%, rgba(177, 75, 47, 0.16) 18%, rgba(255, 255, 255, 0.24) 33%),
    rgba(255, 255, 255, 0.72);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

.empty-state {
  margin-top: 22px;
  padding: 24px;
}

.empty-state strong {
  display: block;
  font-size: 1.08rem;
}

.alert-box {
  margin-top: 18px;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(163, 42, 42, 0.1);
  color: #7b251e;
}

@keyframes shimmer {
  to {
    background-position-x: -200%;
  }
}

@media (max-width: 1024px) {
  .history-hero,
  .panel-header,
  .filter-bar {
    display: grid;
  }

  .session-grid,
  .flow-card {
    grid-template-columns: 1fr;
  }

  .preset-row {
    justify-content: flex-start;
  }
}

@media (max-width: 720px) {
  .history-shell {
    padding-left: 16px;
    padding-right: 16px;
  }

  .panel {
    padding: 22px;
  }

  .hero-actions,
  .ledger-card,
  .footer-row {
    display: grid;
  }

  .card-actions {
    justify-content: flex-start;
  }

  .ledger-side {
    align-items: flex-start;
    text-align: left;
  }
}
</style>
