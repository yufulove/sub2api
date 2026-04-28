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
    description: '稳定默认路线'
  },
  {
    value: 'gemini-2.5-flash-image-preview',
    label: 'Gemini 2.5 Flash Image Preview',
    description: '快速验证预览路线'
  },
  {
    value: 'gemini-3.1-flash-image',
    label: 'Gemini 3.1 Flash Image',
    description: '更新的快速图片模型'
  },
  {
    value: 'gemini-3.1-flash-image-preview',
    label: 'Gemini 3.1 Flash Image Preview',
    description: '3.1 预览路线'
  },
  {
    value: 'gemini-3-pro-image',
    label: 'Gemini 3 Pro Image',
    description: '质量优先图片模型'
  },
  {
    value: 'gemini-3-pro-image-preview',
    label: 'Gemini 3 Pro Image Preview',
    description: 'Pro 预览路线'
  }
]

const imageSizeOptions: ImageSizeOption[] = [
  {
    value: '1024x1024',
    label: '1024x1024',
    description: '方图 / 1K',
    tier: '1K',
    aspectRatio: '1 / 1'
  },
  {
    value: '1536x1024',
    label: '1536x1024',
    description: '横图 / 2K',
    tier: '2K',
    aspectRatio: '3 / 2'
  },
  {
    value: '1024x1536',
    label: '1024x1536',
    description: '竖图 / 2K',
    tier: '2K',
    aspectRatio: '2 / 3'
  },
  {
    value: '2048x2048',
    label: '2048x2048',
    description: '大方图 / 2K',
    tier: '2K',
    aspectRatio: '1 / 1'
  },
  {
    value: '4096x4096',
    label: '4096x4096',
    description: '高分辨率 / 4K',
    tier: '4K',
    aspectRatio: '1 / 1'
  }
]

const promptSuggestions = [
  '北欧护肤品牌极简海报，柔和纸张质感，暖白背景，深绿色点缀，保留充足标题空间',
  '雨夜城市街景，霓虹反射，湿润路面，电影感构图，高对比光影',
  '咖啡店室内宣传图，暖色自然光，木质家具，真实生活方式摄影，浅景深',
  '传统中国山水长卷，薄雾、远山、层次水墨和留白，画面安静雅致'
]

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const imageStudioStore = useImageStudioStore()
const { copyToClipboard } = useClipboard()

const walletBalance = computed(() =>
  formatWalletMoneyFromInternal(authStore.user?.balance ?? 0, appStore.cachedPublicSettings)
)
const mainSiteKeysURL = computed(() =>
  resolveMainSiteURL('/keys?from=studio&return=%2Fstudio%2Fgenerate')
)
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
    const groupName = key.group?.name || '未绑定分组'
    return {
      value: key.id,
      label: availability.usable
        ? `${key.name} / ${groupName}`
        : `${key.name} / ${groupName} / ${availability.reason}`,
      disabled: !availability.usable
    }
  })
)

const modelSelectOptions = computed<SelectOption[]>(() =>
  imageModelOptions.map((option) => ({
    value: option.value,
    label: option.label,
    description: option.description
  }))
)

const selectedModelOption = computed(() =>
  imageModelOptions.find((item) => item.value === selectedModel.value) ?? imageModelOptions[0]
)

const selectedGroupLabel = computed(() => {
  const group = selectedKey.value?.group
  if (!group) {
    return '未绑定分组'
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
    return '待选择可计费路线'
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
    keysErrorMessage.value = 'API Key 加载失败，请刷新后重试。'
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
    appStore.showSuccess(`已生成 ${newCards.length} 张图片`)
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    const message = error instanceof Error ? error.message : '图片生成失败。'
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
    appStore.showError('请先填写提示词，再复制 Studio 链接')
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
    return { usable: false, reason: '未绑定分组' }
  }
  if (key.group.platform === 'openai') {
    return { usable: false, reason: 'OpenAI 图片上游暂未接入' }
  }
  return { usable: true }
}

function statusLabel(status: ApiKey['status']): string {
  switch (status) {
    case 'inactive':
      return '已停用'
    case 'quota_exhausted':
      return '额度已用完'
    case 'expired':
      return '已过期'
    default:
      return '不可用'
  }
}

function platformLabel(platform?: string): string {
  switch (platform) {
    case 'anthropic':
      return 'Anthropic 分组'
    case 'gemini':
      return 'Gemini 分组'
    case 'antigravity':
      return 'Antigravity 分组'
    case 'openai':
      return 'OpenAI 分组'
    default:
      return '未知分组'
  }
}

function resolveSizeLabel(size: string): string {
  return imageSizeOptions.find((item) => item.value === size)?.label || size
}

function resolveAspectRatio(size: string): string {
  return imageSizeOptions.find((item) => item.value === size)?.aspectRatio || '1 / 1'
}

function formatCreatedAt(created: number): string {
  return new Intl.DateTimeFormat('zh-CN', {
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

    <section class="workspace-grid">
      <form class="composer-panel" @submit.prevent="handleGenerate">
        <div class="card-heading">
          <div>
            <p class="eyebrow">图片生成</p>
            <h1>生成图片</h1>
          </div>
          <button type="button" class="secondary-button compact-button" @click="copyCurrentStudioLink">
            复制配置链接
          </button>
        </div>

        <div class="field-block">
          <div class="field-label-row key-label-row">
            <label class="field-label">API Key</label>
            <a class="inline-link" :href="mainSiteKeysURL">配置 API Key</a>
          </div>
          <Select
            v-model="selectedApiKeyId"
            :options="keySelectOptions"
            :disabled="keysLoading || keySelectOptions.length === 0"
            :searchable="keySelectOptions.length > 8"
            placeholder="选择可用的图片 Key"
            search-placeholder="搜索 Key 或分组"
          />
          <p v-if="selectedKey" class="field-note">
            当前路线：{{ selectedGroupLabel }} / {{ selectedKey.name }}
          </p>
          <p v-else-if="keysLoading" class="field-note">正在加载 API Key...</p>
          <p v-else class="field-note">
            请先到主站创建用户 API Key，并绑定 Gemini 或 Antigravity 图片分组。
          </p>
        </div>

        <div class="compact-stats">
          <div>
            <span>余额</span>
            <strong>{{ walletBalance }}</strong>
          </div>
          <div>
            <span>本次规格</span>
            <strong>{{ selectedSizeOption.tier }}</strong>
          </div>
          <div>
            <span>预计费用</span>
            <strong>{{ estimatedCostLabel }}</strong>
          </div>
          <div>
            <span>可用 Key</span>
            <strong>{{ availableKeys.length }}</strong>
          </div>
        </div>

        <div class="control-grid">
          <div class="field-block">
            <div class="field-label-row">
              <label class="field-label">模型</label>
              <span class="field-tag">{{ selectedModelOption.description }}</span>
            </div>
            <Select
              v-model="selectedModel"
              :options="modelSelectOptions"
              searchable
              placeholder="选择图片模型"
              search-placeholder="搜索图片模型"
            />
          </div>

          <div class="field-block">
            <div class="field-label-row">
              <label class="field-label">尺寸</label>
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
                <span>{{ option.tier }}</span>
              </button>
            </div>
          </div>
        </div>

        <div class="field-block">
          <div class="field-label-row">
            <label class="field-label" for="studio-prompt">提示词</label>
            <span class="field-tag">{{ prompt.trim().length }} 字符</span>
          </div>
          <textarea
            id="studio-prompt"
            v-model="prompt"
            ref="promptInputRef"
            class="prompt-box"
            rows="7"
            placeholder="描述主体、风格、构图、光线、画面情绪，以及需要避免的内容。"
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
          <p class="field-note">草稿会保存在当前浏览器，刷新页面不会丢失。</p>
        </div>

        <div v-if="keysErrorMessage" class="alert-box alert-danger">
          {{ keysErrorMessage }}
        </div>
        <div v-if="generateErrorMessage" class="alert-box alert-danger">
          {{ generateErrorMessage }}
        </div>

        <div v-if="!keysLoading && availableKeys.length === 0" class="alert-box alert-warning">
          <strong>当前没有可用的图片 Key。</strong>
          <ul class="reason-list">
            <li v-for="item in unavailableKeyReasons.slice(0, 5)" :key="item.key.id">
              {{ item.key.name }}: {{ item.availability.reason }}
            </li>
          </ul>
          <a class="inline-link" :href="mainSiteKeysURL">去主站管理 API Key</a>
        </div>

        <div class="submit-row">
          <button class="primary-button" type="submit" :disabled="!canGenerate">
            {{ isGenerating ? '生成中...' : '生成图片' }}
          </button>
          <button class="secondary-button" type="button" :disabled="keysLoading" @click="loadKeys">
            刷新 Key
          </button>
          <a class="secondary-button config-key-button" :href="mainSiteKeysURL">
            配置 API Key
          </a>
        </div>
      </form>

      <section class="results-panel">
        <div class="card-heading">
          <div>
            <p class="eyebrow">结果</p>
            <h2>图片预览</h2>
          </div>
          <div class="heading-actions">
            <RouterLink class="secondary-button compact-button" to="/studio/history">历史记录</RouterLink>
            <RouterLink class="secondary-button compact-button" to="/studio/pricing">价格</RouterLink>
          </div>
        </div>

        <div v-if="generationCards.length > 0" class="results-actions">
          <button
            type="button"
            class="secondary-button"
            :disabled="isGenerating"
            @click="imageStudioStore.clearSessionGenerations()"
          >
            清空最近结果
          </button>
          <a class="inline-link" :href="mainSiteUsageURL">查看主站用量</a>
        </div>

        <div v-if="isGenerating" class="loading-grid">
          <div class="loading-card" />
          <div class="loading-card" />
        </div>

        <div v-else-if="isSessionHydrating && generationCards.length === 0" class="empty-state">
          <strong>正在恢复会话</strong>
          <p>正在从当前浏览器读取最近生成的图片。</p>
        </div>

        <div v-else-if="generationCards.length === 0" class="empty-state">
          <strong>还没有生成结果</strong>
          <p>选择可用 Key，填写提示词后，图片会显示在这里。</p>
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
                修订提示词：{{ card.revisedPrompt }}
              </p>
              <div class="image-footer">
                <div class="image-tags">
                  <span>{{ card.model }}</span>
                  <span>{{ card.keyName }}</span>
                </div>
                <div class="image-actions">
                  <button type="button" class="card-action-button" @click="reuseCardPrompt(card)">
                    复用
                  </button>
                  <button type="button" class="card-action-button" @click="copyCardPrompt(card)">
                    复制提示词
                  </button>
                  <button type="button" class="card-action-button" @click="copyCardStudioLink(card)">
                    复制链接
                  </button>
                  <button type="button" class="download-button" @click="downloadCard(card, index)">
                    下载 PNG
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
  --paper: #f6f7f5;
  --panel: #ffffff;
  --ink: #12231e;
  --muted: rgba(18, 35, 30, 0.7);
  --line: rgba(18, 35, 30, 0.12);
  --accent: #0f766e;
  --accent-strong: #0b5f59;
  min-height: 100vh;
  padding: 24px 24px 56px;
  background: linear-gradient(180deg, #fbfcfb 0%, var(--paper) 100%);
  color: var(--ink);
  font-family: "Space Grotesk", "Noto Sans SC", sans-serif;
}

.workspace-grid {
  width: min(1180px, 100%);
  margin: 0 auto;
}

.eyebrow {
  margin: 0 0 6px;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--accent-strong);
}

.card-heading h1,
.card-heading h2 {
  margin: 0;
  line-height: 1.18;
}

.card-heading h1 {
  font-size: 1.85rem;
}

.card-heading h2 {
  font-size: 1.45rem;
}

.field-note,
.empty-state p,
.image-revised,
.size-button span {
  color: var(--muted);
}

.inline-link {
  color: inherit;
  text-decoration: none;
}

.workspace-grid {
  display: grid;
  grid-template-columns: minmax(360px, 460px) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.composer-panel,
.results-panel {
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  box-shadow: 0 12px 32px rgba(18, 35, 30, 0.06);
  padding: 18px;
}

.card-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.heading-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.compact-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 16px;
}

.field-block + .compact-stats {
  margin-top: 16px;
}

.compact-stats div {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #f7f9f8;
}

.compact-stats span {
  display: block;
  color: rgba(18, 35, 30, 0.58);
  font-size: 0.76rem;
}

.compact-stats strong {
  display: block;
  margin-top: 4px;
  overflow-wrap: anywhere;
  font-size: 0.94rem;
}

.field-block + .field-block,
.field-block + .alert-box,
.alert-box + .submit-row {
  margin-top: 16px;
}

.results-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 16px;
}

.control-grid {
  display: grid;
  gap: 14px;
}

.field-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.key-label-row {
  margin-bottom: 8px;
}

.field-label {
  display: block;
  margin-bottom: 8px;
  font-size: 0.95rem;
  font-weight: 700;
}

.key-label-row .field-label {
  margin-bottom: 0;
}

.field-tag {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 6px;
  background: rgba(18, 35, 30, 0.08);
  font-size: 0.76rem;
  font-weight: 700;
}

.field-note {
  margin: 8px 0 0;
  font-size: 0.88rem;
  line-height: 1.5;
}

:deep(.select-trigger) {
  min-height: 42px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.86);
  border-color: rgba(18, 35, 30, 0.12);
}

.size-grid,
.suggestion-row,
.gallery-grid {
  display: grid;
  gap: 8px;
}

.size-grid {
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
}

.size-button,
.suggestion-chip,
.secondary-button,
.download-button {
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.72);
  color: inherit;
}

.size-button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  text-align: left;
  transition: border-color 160ms ease, transform 160ms ease, background 160ms ease;
}

.size-button strong {
  font-size: 0.95rem;
}

.size-button-active {
  border-color: rgba(15, 118, 110, 0.5);
  background: rgba(15, 118, 110, 0.09);
}

.prompt-box {
  width: 100%;
  padding: 14px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.84);
  color: var(--ink);
  font: inherit;
  line-height: 1.6;
  resize: vertical;
  min-height: 160px;
}

.prompt-box:focus {
  outline: 2px solid rgba(15, 118, 110, 0.22);
  border-color: rgba(15, 118, 110, 0.4);
}

.suggestion-row {
  grid-template-columns: 1fr;
  margin-top: 10px;
}

.suggestion-chip {
  padding: 10px 12px;
  border-radius: 8px;
  text-align: left;
  line-height: 1.45;
  font: inherit;
  font-size: 0.88rem;
  transition: border-color 160ms ease, transform 160ms ease;
}

.alert-box {
  padding: 12px 14px;
  border-radius: 8px;
  line-height: 1.55;
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
  margin: 8px 0;
  padding-left: 1.1rem;
}

.submit-row {
  display: flex;
  gap: 10px;
  margin-top: 18px;
}

.primary-button,
.secondary-button,
.download-button,
.card-action-button {
  min-height: 42px;
  padding: 0 14px;
  border-radius: 8px;
  font: inherit;
  font-weight: 700;
  transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.primary-button {
  flex: 1;
  border: none;
  background: var(--accent);
  color: #ffffff;
  box-shadow: 0 12px 24px rgba(15, 118, 110, 0.18);
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

.config-key-button {
  text-decoration: none;
}

.compact-button {
  min-width: 0;
  min-height: 36px;
  white-space: nowrap;
}

.primary-button:hover,
.secondary-button:hover,
.download-button:hover,
.suggestion-chip:hover,
.size-button:hover,
.card-action-button:hover {
  transform: translateY(-1px);
}

.card-action-button {
  min-height: 36px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.72);
  color: inherit;
}

.loading-grid,
.gallery-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 16px;
}

.loading-card,
.image-card,
.empty-state {
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.74);
}

.loading-card {
  min-height: 300px;
  background:
    linear-gradient(110deg, rgba(255, 255, 255, 0.25) 8%, rgba(15, 118, 110, 0.14) 18%, rgba(255, 255, 255, 0.24) 33%),
    rgba(255, 255, 255, 0.72);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

.empty-state {
  padding: 22px;
}

.empty-state strong {
  display: block;
  font-size: 1.1rem;
}

.image-card {
  overflow: hidden;
}

.image-frame {
  background: rgba(18, 35, 30, 0.04);
}

.image-frame img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-body {
  padding: 14px;
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
  color: rgba(18, 35, 30, 0.55);
}

.image-prompt {
  margin: 12px 0 0;
  line-height: 1.6;
  font-size: 0.94rem;
}

.image-revised {
  margin: 8px 0 0;
  font-size: 0.88rem;
  line-height: 1.55;
}

.image-footer {
  margin-top: 14px;
}

.image-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.image-tags span {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 8px;
  border-radius: 6px;
  background: rgba(18, 35, 30, 0.07);
  font-size: 0.78rem;
  overflow-wrap: anywhere;
}

.image-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.download-button {
  min-height: 36px;
  padding: 0 12px;
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
  .workspace-grid {
    display: grid;
  }

  .submit-row,
  .results-actions,
  .heading-actions {
    justify-content: flex-start;
  }

  .workspace-grid,
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
    padding-top: 16px;
    padding-left: 16px;
    padding-right: 16px;
  }

  .composer-panel,
  .results-panel {
    padding: 14px;
  }

  .card-heading,
  .image-meta {
    display: grid;
  }

  .size-grid,
  .compact-stats {
    grid-template-columns: 1fr;
  }

  .submit-row,
  .image-footer,
  .image-actions {
    flex-direction: column;
  }
}
</style>
