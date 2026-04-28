<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { keysAPI } from '@/api'
import { generateImage } from '@/api/image'
import StudioShell from '@/components/image/StudioShell.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore, useAuthStore, useImageStudioStore } from '@/stores'
import type { StudioSessionGeneration } from '@/stores/imageStudio'
import type { ApiKey } from '@/types'
import {
  buildComposerPresetQuery,
  IMAGE_STUDIO_MODEL_STORAGE_KEY,
  IMAGE_STUDIO_SIZE_STORAGE_KEY,
  persistPromptDraft,
  persistStoredChoice,
  readComposerPresetFromQuery,
  readStoredChoice,
  readStoredPrompt
} from '@/utils/imageStudioComposer'
import { resolveMainSiteURL } from '@/utils/siteMode'
import { formatWalletMoneyFromInternal } from '@/utils/walletDisplay'

type SizeTier = '1K' | '2K' | '4K'

interface ImageModelOption {
  value: string
  label: string
  description: string
}

interface ImageSizeOption {
  value: string
  label: string
  description: string
  tier: SizeTier
  aspectRatio: string
}

const KEY_STORAGE_KEY = 'studio_image_key_id'

const imageModelOptions: ImageModelOption[] = [
  {
    value: 'gemini-2.5-flash-image',
    label: 'Gemini 2.5 Flash Image',
    description: 'Default stable image route'
  },
  {
    value: 'gemini-2.5-flash-image-preview',
    label: 'Gemini 2.5 Flash Image Preview',
    description: 'Preview route for quick validation'
  },
  {
    value: 'gemini-3.1-flash-image',
    label: 'Gemini 3.1 Flash Image',
    description: 'Newer fast image model'
  },
  {
    value: 'gemini-3.1-flash-image-preview',
    label: 'Gemini 3.1 Flash Image Preview',
    description: 'Preview route for 3.1'
  },
  {
    value: 'gemini-3-pro-image',
    label: 'Gemini 3 Pro Image',
    description: 'Quality-first image model'
  },
  {
    value: 'gemini-3-pro-image-preview',
    label: 'Gemini 3 Pro Image Preview',
    description: 'Preview route for Pro'
  }
]

const imageSizeOptions: ImageSizeOption[] = [
  {
    value: '1024x1024',
    label: '1024x1024',
    description: 'Square / 1K tier',
    tier: '1K',
    aspectRatio: '1 / 1'
  },
  {
    value: '1536x1024',
    label: '1536x1024',
    description: 'Landscape / 2K tier',
    tier: '2K',
    aspectRatio: '3 / 2'
  },
  {
    value: '1024x1536',
    label: '1024x1536',
    description: 'Portrait / 2K tier',
    tier: '2K',
    aspectRatio: '2 / 3'
  },
  {
    value: '2048x2048',
    label: '2048x2048',
    description: 'Large square / 2K tier',
    tier: '2K',
    aspectRatio: '1 / 1'
  },
  {
    value: '4096x4096',
    label: '4096x4096',
    description: 'High resolution / 4K tier',
    tier: '4K',
    aspectRatio: '1 / 1'
  }
]

const promptSuggestions = [
  'Minimal poster for a Nordic skincare brand, soft paper texture, warm beige and deep green palette, generous headline space',
  'Cyberpunk rainy night street scene, neon reflections, wet asphalt, cinematic framing, high contrast',
  'Coffee shop interior campaign visual, warm light, wood furniture, realistic lifestyle mood, shallow depth of field',
  'Traditional Chinese landscape scroll, mist, distant mountains, layered ink wash, elegant negative space'
]

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const imageStudioStore = useImageStudioStore()
const { copyToClipboard } = useClipboard()

const brandName = computed(() => appStore.siteName || 'FionaAI')
const walletBalance = computed(() =>
  formatWalletMoneyFromInternal(authStore.user?.balance ?? 0, appStore.cachedPublicSettings)
)
const mainSiteKeysURL = computed(() => resolveMainSiteURL('/keys'))
const mainSiteUsageURL = computed(() => resolveMainSiteURL('/usage'))

const apiKeys = ref<ApiKey[]>([])
const keysLoading = ref(false)
const keysErrorMessage = ref('')
const generateErrorMessage = ref('')
const selectedApiKeyId = ref<number | null>(null)
const selectedModel = ref(
  readStoredChoice(
    IMAGE_STUDIO_MODEL_STORAGE_KEY,
    imageModelOptions.map((item) => item.value),
    imageModelOptions[0].value
  )
)
const selectedSize = ref(
  readStoredChoice(
    IMAGE_STUDIO_SIZE_STORAGE_KEY,
    imageSizeOptions.map((item) => item.value),
    imageSizeOptions[0].value
  )
)
const prompt = ref(readStoredPrompt(promptSuggestions[0]))
const isGenerating = ref(false)
const promptInputRef = ref<HTMLTextAreaElement | null>(null)

let pendingController: AbortController | null = null
let storedKeyCandidate = readStoredKeyId()

const selectedKey = computed(() =>
  apiKeys.value.find((key) => key.id === selectedApiKeyId.value) ?? null
)

const generationCards = computed(() => imageStudioStore.sessionGenerations)
const isSessionHydrating = computed(() => imageStudioStore.isHydrating)

const selectedSizeOption = computed(() =>
  imageSizeOptions.find((item) => item.value === selectedSize.value) ?? imageSizeOptions[0]
)

const availableKeys = computed(() =>
  apiKeys.value.filter((key) => getKeyAvailability(key).usable)
)

const unavailableKeyReasons = computed(() =>
  apiKeys.value
    .map((key) => ({ key, availability: getKeyAvailability(key) }))
    .filter((item) => !item.availability.usable)
)

const keySelectOptions = computed<SelectOption[]>(() =>
  apiKeys.value.map((key) => {
    const availability = getKeyAvailability(key)
    const groupName = key.group?.name || 'No group'
    return {
      value: key.id,
      label: availability.usable
        ? `${key.name} / ${groupName}`
        : `${key.name} / ${groupName} / ${availability.reason}`,
      disabled: !availability.usable
    }
  })
)

const selectedGroupLabel = computed(() => {
  const group = selectedKey.value?.group
  if (!group) {
    return 'No group assigned'
  }
  return `${group.name} / ${platformLabel(group.platform)}`
})

const estimatedCost = computed<number | null>(() => {
  const group = selectedKey.value?.group
  if (!group) {
    return null
  }

  switch (selectedSizeOption.value.tier) {
    case '1K':
      return group.image_price_1k ?? null
    case '2K':
      return group.image_price_2k ?? null
    case '4K':
      return group.image_price_4k ?? null
    default:
      return null
  }
})

const estimatedCostLabel = computed(() => {
  if (estimatedCost.value == null) {
    return 'Price depends on the selected route'
  }
  return formatWalletMoneyFromInternal(estimatedCost.value, appStore.cachedPublicSettings, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4
  })
})

const canGenerate = computed(() =>
  !!selectedKey.value && prompt.value.trim().length > 0 && !isGenerating.value
)

watch(
  availableKeys,
  (keys) => {
    if (keys.length === 0) {
      selectedApiKeyId.value = null
      return
    }

    if (selectedApiKeyId.value && keys.some((key) => key.id === selectedApiKeyId.value)) {
      return
    }

    if (storedKeyCandidate && keys.some((key) => key.id === storedKeyCandidate)) {
      selectedApiKeyId.value = storedKeyCandidate
      storedKeyCandidate = null
      return
    }

    selectedApiKeyId.value = keys[0].id
  },
  { immediate: true }
)

watch(selectedApiKeyId, (value) => {
  persistKeyId(value)
})

watch(selectedModel, (value) => {
  persistStoredChoice(IMAGE_STUDIO_MODEL_STORAGE_KEY, value)
})

watch(selectedSize, (value) => {
  persistStoredChoice(IMAGE_STUDIO_SIZE_STORAGE_KEY, value)
})

watch(prompt, (value) => {
  persistPromptDraft(value)
})

watch(
  () => [route.query.prompt, route.query.model, route.query.size],
  () => {
    applyRoutePreset()
  },
  { immediate: true }
)

async function loadKeys() {
  keysLoading.value = true
  keysErrorMessage.value = ''

  try {
    const response = await keysAPI.list(1, 200)
    apiKeys.value = response.items
  } catch {
    keysErrorMessage.value = 'Failed to load API keys. Refresh and try again.'
    appStore.showError(keysErrorMessage.value)
  } finally {
    keysLoading.value = false
  }
}

async function handleGenerate() {
  const apiKey = selectedKey.value
  const cleanPrompt = prompt.value.trim()

  if (!apiKey || !cleanPrompt) {
    return
  }

  generateErrorMessage.value = ''
  pendingController?.abort()
  const controller = new AbortController()
  pendingController = controller
  isGenerating.value = true

  try {
    const response = await generateImage({
      apiKey: apiKey.key,
      model: selectedModel.value,
      prompt: cleanPrompt,
      size: selectedSize.value,
      signal: controller.signal
    })

    const items = Array.isArray(response.data) ? response.data : []
    const newCards: StudioSessionGeneration[] = items
      .filter((item) => typeof item.b64_json === 'string' && item.b64_json.trim() !== '')
      .map((item, index) => ({
        id: `${response.created}-${index}-${Math.random().toString(36).slice(2, 8)}`,
        created: response.created,
        prompt: cleanPrompt,
        revisedPrompt: item.revised_prompt?.trim() || '',
        model: selectedModel.value,
        size: selectedSize.value,
        keyName: apiKey.name,
        imageSrc: `data:image/png;base64,${item.b64_json}`
      }))

    if (newCards.length === 0) {
      throw new Error('The server returned no renderable image data.')
    }

    imageStudioStore.prependGenerations(newCards)
    appStore.showSuccess(`Received ${newCards.length} image result${newCards.length > 1 ? 's' : ''}`)
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    const message = error instanceof Error ? error.message : 'Image generation failed.'
    generateErrorMessage.value = message
    appStore.showError(message)
  } finally {
    if (pendingController === controller) {
      pendingController = null
    }
    isGenerating.value = false
  }
}

function downloadCard(card: StudioSessionGeneration, index: number) {
  const link = document.createElement('a')
  link.href = card.imageSrc
  link.download = `${sanitizeFilename(card.model)}-${card.size}-${card.created}-${index + 1}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function choosePromptSuggestion(value: string) {
  prompt.value = value
}

async function reuseCardPrompt(card: StudioSessionGeneration) {
  prompt.value = card.prompt

  if (imageModelOptions.some((option) => option.value === card.model)) {
    selectedModel.value = card.model
  }
  if (imageSizeOptions.some((option) => option.value === card.size)) {
    selectedSize.value = card.size
  }

  await nextTick()
  promptInputRef.value?.focus()
  promptInputRef.value?.setSelectionRange(prompt.value.length, prompt.value.length)
  if (typeof promptInputRef.value?.scrollIntoView === 'function') {
    promptInputRef.value.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

function copyCardPrompt(card: StudioSessionGeneration) {
  void copyToClipboard(card.prompt, 'Prompt copied to clipboard')
}

function copyCardStudioLink(card: StudioSessionGeneration) {
  void copyToClipboard(buildStudioLink(card.prompt, card.model, card.size), 'Studio link copied to clipboard')
}

function copyCurrentStudioLink() {
  const cleanPrompt = prompt.value.trim()
  if (!cleanPrompt) {
    appStore.showError('Write a prompt before copying a studio link')
    return
  }

  void copyToClipboard(
    buildStudioLink(cleanPrompt, selectedModel.value, selectedSize.value),
    'Studio link copied to clipboard'
  )
}

function getKeyAvailability(key: ApiKey): { usable: boolean; reason?: string } {
  if (key.status !== 'active') {
    return { usable: false, reason: statusLabel(key.status) }
  }
  if (!key.group_id || !key.group) {
    return { usable: false, reason: 'No group assigned' }
  }
  if (key.group.platform === 'openai') {
    return { usable: false, reason: 'OpenAI upstream image route is not implemented yet' }
  }
  return { usable: true }
}

function statusLabel(status: ApiKey['status']): string {
  switch (status) {
    case 'inactive':
      return 'Inactive'
    case 'quota_exhausted':
      return 'Quota exhausted'
    case 'expired':
      return 'Expired'
    default:
      return 'Unavailable'
  }
}

function platformLabel(platform?: string): string {
  switch (platform) {
    case 'anthropic':
      return 'Anthropic group'
    case 'gemini':
      return 'Gemini group'
    case 'antigravity':
      return 'Antigravity group'
    case 'openai':
      return 'OpenAI group'
    default:
      return 'Unknown group'
  }
}

function resolveSizeLabel(size: string): string {
  return imageSizeOptions.find((item) => item.value === size)?.label || size
}

function resolveAspectRatio(size: string): string {
  return imageSizeOptions.find((item) => item.value === size)?.aspectRatio || '1 / 1'
}

function formatCreatedAt(created: number): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(created * 1000))
}

function isAbortError(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }
  const payload = error as { name?: string; code?: string }
  return payload.name === 'AbortError' || payload.code === 'ERR_CANCELED'
}

function sanitizeFilename(value: string): string {
  return value.replace(/[^a-zA-Z0-9-_]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '')
}

function applyRoutePreset() {
  const preset = readComposerPresetFromQuery(
    route.query as Record<string, unknown>,
    imageModelOptions.map((item) => item.value),
    imageSizeOptions.map((item) => item.value)
  )

  if (!preset) {
    return
  }

  if (typeof preset.prompt === 'string') {
    prompt.value = preset.prompt
  }
  if (typeof preset.model === 'string') {
    selectedModel.value = preset.model
  }
  if (typeof preset.size === 'string') {
    selectedSize.value = preset.size
  }
}

function buildStudioLink(promptValue: string, modelValue: string, sizeValue: string): string {
  const target = router.resolve({
    path: '/studio/generate',
    query: buildComposerPresetQuery({
      prompt: promptValue,
      model: modelValue,
      size: sizeValue
    })
  })

  return new URL(target.href, window.location.origin).toString()
}

function readStoredKeyId(): number | null {
  if (typeof window === 'undefined') {
    return null
  }
  const raw = window.localStorage.getItem(KEY_STORAGE_KEY)
  const numeric = Number(raw)
  return Number.isInteger(numeric) && numeric > 0 ? numeric : null
}

function persistKeyId(value: number | null) {
  if (typeof window === 'undefined') {
    return
  }
  if (value == null) {
    window.localStorage.removeItem(KEY_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(KEY_STORAGE_KEY, String(value))
}

onMounted(() => {
  void imageStudioStore.ensureHydrated()
  loadKeys()
})

onUnmounted(() => {
  pendingController?.abort()
})
</script>

<template>
  <main class="generate-shell">
    <StudioShell max-width="1180px" />

    <section class="generate-hero">
      <div>
        <p class="eyebrow">Generate</p>
        <h1>{{ brandName }} Studio</h1>
        <p class="lead">
          Generate images with your own key, model, and size settings, then keep the newest renders
          visible in the same workspace.
        </p>
      </div>

      <div class="hero-links">
        <RouterLink class="ghost-link" to="/studio/history">Open history</RouterLink>
        <RouterLink class="ghost-link" to="/studio/pricing">View pricing</RouterLink>
        <a class="ghost-link" :href="mainSiteKeysURL">Manage API keys on main site</a>
        <a class="ghost-link" :href="mainSiteUsageURL">View usage on main site</a>
      </div>
    </section>

    <section class="status-ribbon">
      <div class="stat-pill">
        <span>Wallet</span>
        <strong>{{ walletBalance }}</strong>
      </div>
      <div class="stat-pill">
        <span>Current Tier</span>
        <strong>{{ selectedSizeOption.tier }}</strong>
      </div>
      <div class="stat-pill">
        <span>Est. Cost</span>
        <strong>{{ estimatedCostLabel }}</strong>
      </div>
      <div class="stat-pill">
        <span>Usable Keys</span>
        <strong>{{ availableKeys.length }}</strong>
      </div>
    </section>

    <section class="workspace-grid">
      <form class="composer-card" @submit.prevent="handleGenerate">
        <div class="card-heading">
          <p class="eyebrow">Request</p>
          <h2>Compose a prompt and render</h2>
          <p class="copy">
            Choose a usable key, model, and size first. Estimated cost follows the bound group pricing
            for the selected image tier.
          </p>
        </div>

        <div class="composer-actions">
          <button type="button" class="secondary-button" @click="copyCurrentStudioLink">
            Copy studio link
          </button>
        </div>

        <div class="field-block">
          <label class="field-label">API key</label>
          <Select
            v-model="selectedApiKeyId"
            :options="keySelectOptions"
            :disabled="keysLoading || keySelectOptions.length === 0"
            :searchable="keySelectOptions.length > 8"
            placeholder="Choose a usable key"
            search-placeholder="Search keys or groups"
          />
          <p v-if="selectedKey" class="field-note">
            Current route: {{ selectedGroupLabel }} / key: {{ selectedKey.name }}
          </p>
          <p v-else-if="keysLoading" class="field-note">Loading your API keys...</p>
          <p v-else class="field-note">Choose a usable image key to start.</p>
        </div>

        <div class="field-block">
          <div class="field-label-row">
            <label class="field-label">Model</label>
            <span class="field-tag">{{ selectedModel }}</span>
          </div>
          <div class="choice-grid">
            <button
              v-for="option in imageModelOptions"
              :key="option.value"
              type="button"
              :class="['choice-button', selectedModel === option.value && 'choice-button-active']"
              @click="selectedModel = option.value"
            >
              <strong>{{ option.label }}</strong>
              <span>{{ option.description }}</span>
            </button>
          </div>
        </div>

        <div class="field-block">
          <div class="field-label-row">
            <label class="field-label">Size</label>
            <span class="field-tag">{{ selectedSizeOption.description }}</span>
          </div>
          <div class="size-grid">
            <button
              v-for="option in imageSizeOptions"
              :key="option.value"
              type="button"
              :class="['size-button', selectedSize === option.value && 'size-button-active']"
              @click="selectedSize = option.value"
            >
              <strong>{{ option.label }}</strong>
              <span>{{ option.description }}</span>
            </button>
          </div>
        </div>

        <div class="field-block">
          <div class="field-label-row">
            <label class="field-label" for="studio-prompt">Prompt</label>
            <span class="field-tag">{{ prompt.trim().length }} chars</span>
          </div>
          <textarea
            id="studio-prompt"
            v-model="prompt"
            ref="promptInputRef"
            class="prompt-box"
            rows="8"
            placeholder="Describe the subject, style, composition, lighting, mood, and anything the image should avoid."
          />
          <div class="suggestion-row">
            <button
              v-for="item in promptSuggestions"
              :key="item"
              type="button"
              class="suggestion-chip"
              @click="choosePromptSuggestion(item)"
            >
              {{ item }}
            </button>
          </div>
          <p class="field-note">Prompt draft saves locally in this browser while you work.</p>
        </div>

        <div v-if="keysErrorMessage" class="alert-box alert-danger">
          {{ keysErrorMessage }}
        </div>
        <div v-if="generateErrorMessage" class="alert-box alert-danger">
          {{ generateErrorMessage }}
        </div>

        <div v-if="!keysLoading && availableKeys.length === 0" class="alert-box alert-warning">
          <strong>No usable image key is available right now.</strong>
          <ul class="reason-list">
            <li v-for="item in unavailableKeyReasons.slice(0, 5)" :key="item.key.id">
              {{ item.key.name }}: {{ item.availability.reason }}
            </li>
          </ul>
          <a class="inline-link" :href="mainSiteKeysURL">Open main-site key management</a>
        </div>

        <div class="submit-row">
          <button class="primary-button" type="submit" :disabled="!canGenerate">
            {{ isGenerating ? 'Generating...' : 'Generate image' }}
          </button>
          <button class="secondary-button" type="button" :disabled="keysLoading" @click="loadKeys">
            Refresh keys
          </button>
        </div>
      </form>

      <section class="results-card">
        <div class="card-heading">
          <p class="eyebrow">Results</p>
          <h2>Recent renders</h2>
          <p class="copy">
            New images appear at the top immediately. Recent renders stay available in this browser for the same account.
          </p>
        </div>

        <div v-if="generationCards.length > 0" class="results-actions">
          <RouterLink class="inline-link" to="/studio/history">Open history</RouterLink>
          <button
            type="button"
            class="secondary-button"
            :disabled="isGenerating"
            @click="imageStudioStore.clearSessionGenerations()"
          >
            Clear recent renders
          </button>
        </div>

        <div v-if="isGenerating" class="loading-grid">
          <div class="loading-card" />
          <div class="loading-card" />
        </div>

        <div v-else-if="isSessionHydrating && generationCards.length === 0" class="empty-state">
          <strong>Restoring browser session</strong>
          <p>Recent image cards are loading from local browser storage for this account.</p>
        </div>

        <div v-else-if="generationCards.length === 0" class="empty-state">
          <strong>No image yet</strong>
          <p>Choose a usable key, write a prompt, and your rendered images will appear here.</p>
        </div>

        <div v-else class="gallery-grid">
          <article
            v-for="(card, index) in generationCards"
            :key="card.id"
            class="image-card"
          >
            <div class="image-frame" :style="{ aspectRatio: resolveAspectRatio(card.size) }">
              <img :src="card.imageSrc" :alt="card.prompt" loading="lazy" />
            </div>

            <div class="image-body">
              <div class="image-meta">
                <span>{{ formatCreatedAt(card.created) }}</span>
                <span>{{ resolveSizeLabel(card.size) }}</span>
              </div>
              <p class="image-prompt">{{ card.prompt }}</p>
              <p v-if="card.revisedPrompt && card.revisedPrompt !== card.prompt" class="image-revised">
                Revised prompt: {{ card.revisedPrompt }}
              </p>
              <div class="image-footer">
                <div class="image-tags">
                  <span>{{ card.model }}</span>
                  <span>{{ card.keyName }}</span>
                </div>
                <div class="image-actions">
                  <button type="button" class="card-action-button" @click="reuseCardPrompt(card)">
                    Reuse prompt
                  </button>
                  <button type="button" class="card-action-button" @click="copyCardPrompt(card)">
                    Copy prompt
                  </button>
                  <button type="button" class="card-action-button" @click="copyCardStudioLink(card)">
                    Copy studio link
                  </button>
                  <button type="button" class="download-button" @click="downloadCard(card, index)">
                    Download PNG
                  </button>
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.generate-shell {
  --paper: #f4efe6;
  --panel: rgba(255, 255, 255, 0.78);
  --ink: #12231e;
  --muted: rgba(18, 35, 30, 0.7);
  --line: rgba(18, 35, 30, 0.12);
  --accent: #c65c2c;
  --accent-strong: #8f3410;
  min-height: 100vh;
  padding: 40px 24px 72px;
  background:
    radial-gradient(circle at top left, rgba(198, 92, 44, 0.16), transparent 28rem),
    radial-gradient(circle at right center, rgba(31, 107, 89, 0.15), transparent 24rem),
    linear-gradient(180deg, #f7f1e8 0%, var(--paper) 100%);
  color: var(--ink);
  font-family: "Space Grotesk", "Noto Sans SC", sans-serif;
}

.generate-hero,
.status-ribbon,
.workspace-grid {
  width: min(1180px, 100%);
  margin: 0 auto;
}

.generate-hero {
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
  color: var(--accent-strong);
}

.generate-hero h1,
.card-heading h2 {
  margin: 0;
  line-height: 0.96;
  letter-spacing: -0.05em;
}

.generate-hero h1 {
  font-size: clamp(2.5rem, 6vw, 4.5rem);
}

.lead,
.copy,
.field-note,
.empty-state p,
.image-revised,
.choice-button span,
.size-button span {
  color: var(--muted);
}

.lead {
  max-width: 45rem;
  margin: 18px 0 0;
  line-height: 1.75;
}

.hero-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px;
}

.ghost-link,
.inline-link {
  color: inherit;
  text-decoration: none;
}

.ghost-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  padding: 0 18px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.6);
  font-weight: 700;
  transition: transform 160ms ease, border-color 160ms ease;
}

.ghost-link:hover,
.primary-button:hover,
.secondary-button:hover,
.download-button:hover,
.suggestion-chip:hover {
  transform: translateY(-1px);
}

.status-ribbon {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin-top: 24px;
}

.stat-pill,
.composer-card,
.results-card {
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel);
  backdrop-filter: blur(14px);
  box-shadow: 0 20px 60px rgba(18, 35, 30, 0.08);
}

.stat-pill {
  padding: 18px 20px;
}

.stat-pill span {
  display: block;
  font-size: 0.78rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: rgba(18, 35, 30, 0.55);
}

.stat-pill strong {
  display: block;
  margin-top: 8px;
  font-size: 1.15rem;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(320px, 420px) minmax(0, 1fr);
  gap: 20px;
  margin-top: 20px;
  align-items: start;
}

.composer-card,
.results-card {
  padding: 28px;
}

.card-heading h2 {
  font-size: clamp(1.8rem, 4vw, 2.6rem);
}

.copy {
  margin: 14px 0 0;
  line-height: 1.72;
}

.field-block + .field-block,
.field-block + .alert-box,
.alert-box + .submit-row {
  margin-top: 22px;
}

.results-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 18px;
}

.composer-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 18px;
}

.field-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.field-label {
  display: block;
  margin-bottom: 10px;
  font-size: 0.95rem;
  font-weight: 700;
}

.field-tag {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(18, 35, 30, 0.08);
  font-size: 0.78rem;
  font-weight: 700;
}

.field-note {
  margin: 10px 0 0;
  font-size: 0.92rem;
  line-height: 1.6;
}

:deep(.select-trigger) {
  min-height: 50px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.86);
  border-color: rgba(18, 35, 30, 0.12);
}

.choice-grid,
.size-grid,
.suggestion-row,
.gallery-grid {
  display: grid;
  gap: 12px;
}

.choice-grid,
.size-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.choice-button,
.size-button,
.suggestion-chip,
.secondary-button,
.download-button {
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.72);
  color: inherit;
}

.choice-button,
.size-button {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 14px 16px;
  border-radius: 20px;
  text-align: left;
  transition: border-color 160ms ease, transform 160ms ease, background 160ms ease;
}

.choice-button strong,
.size-button strong {
  font-size: 0.95rem;
}

.choice-button-active,
.size-button-active {
  border-color: rgba(198, 92, 44, 0.46);
  background: linear-gradient(180deg, rgba(198, 92, 44, 0.12) 0%, rgba(255, 255, 255, 0.94) 100%);
  box-shadow: 0 14px 28px rgba(198, 92, 44, 0.12);
}

.prompt-box {
  width: 100%;
  padding: 18px;
  border: 1px solid var(--line);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.84);
  color: var(--ink);
  font: inherit;
  line-height: 1.72;
  resize: vertical;
  min-height: 180px;
}

.prompt-box:focus {
  outline: 2px solid rgba(198, 92, 44, 0.22);
  border-color: rgba(198, 92, 44, 0.4);
}

.suggestion-row {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 12px;
}

.suggestion-chip {
  padding: 12px 14px;
  border-radius: 18px;
  text-align: left;
  line-height: 1.55;
  font: inherit;
  transition: border-color 160ms ease, transform 160ms ease;
}

.alert-box {
  padding: 14px 16px;
  border-radius: 18px;
  line-height: 1.65;
}

.alert-danger {
  background: rgba(163, 42, 42, 0.1);
  color: #7b251e;
}

.alert-warning {
  background: rgba(198, 92, 44, 0.11);
  color: #7b371c;
}

.reason-list {
  margin: 10px 0;
  padding-left: 1.1rem;
}

.submit-row {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}

.primary-button,
.secondary-button,
.download-button,
.card-action-button {
  min-height: 48px;
  padding: 0 18px;
  border-radius: 999px;
  font: inherit;
  font-weight: 700;
  transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.primary-button {
  flex: 1;
  border: none;
  background: linear-gradient(135deg, var(--accent) 0%, #ef8e52 100%);
  color: #fff8f3;
  box-shadow: 0 18px 34px rgba(198, 92, 44, 0.24);
}

.primary-button:disabled,
.secondary-button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.secondary-button {
  min-width: 120px;
}

.card-action-button {
  min-height: 40px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.72);
  color: inherit;
}

.loading-grid,
.gallery-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 22px;
}

.loading-card,
.image-card,
.empty-state {
  border: 1px solid var(--line);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.74);
}

.loading-card {
  min-height: 340px;
  background:
    linear-gradient(110deg, rgba(255, 255, 255, 0.25) 8%, rgba(198, 92, 44, 0.16) 18%, rgba(255, 255, 255, 0.24) 33%),
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
  font-size: 1.1rem;
}

.image-card {
  overflow: hidden;
}

.image-frame {
  background:
    radial-gradient(circle at top left, rgba(198, 92, 44, 0.14), transparent 55%),
    rgba(18, 35, 30, 0.04);
}

.image-frame img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-body {
  padding: 18px 18px 20px;
}

.image-meta,
.image-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.image-meta {
  font-size: 0.8rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(18, 35, 30, 0.55);
}

.image-prompt {
  margin: 14px 0 0;
  line-height: 1.7;
  font-size: 0.98rem;
}

.image-revised {
  margin: 10px 0 0;
  font-size: 0.88rem;
  line-height: 1.6;
}

.image-footer {
  margin-top: 16px;
}

.image-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.image-tags span {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(18, 35, 30, 0.07);
  font-size: 0.78rem;
}

.image-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.download-button {
  min-height: 40px;
  padding: 0 16px;
}

.inline-link {
  font-weight: 700;
}

@keyframes shimmer {
  to {
    background-position-x: -200%;
  }
}

@media (max-width: 1080px) {
  .generate-hero,
  .workspace-grid {
    display: grid;
  }

  .hero-links,
  .submit-row,
  .results-actions,
  .composer-actions {
    justify-content: flex-start;
  }

  .workspace-grid,
  .status-ribbon,
  .loading-grid,
  .gallery-grid {
    grid-template-columns: 1fr;
  }

  .image-actions {
    justify-content: flex-start;
  }
}

@media (max-width: 720px) {
  .generate-shell {
    padding-left: 16px;
    padding-right: 16px;
  }

  .composer-card,
  .results-card {
    padding: 22px;
  }

  .choice-grid,
  .size-grid,
  .suggestion-row {
    grid-template-columns: 1fr;
  }

  .submit-row,
  .image-footer,
  .image-actions {
    flex-direction: column;
  }
}
</style>
