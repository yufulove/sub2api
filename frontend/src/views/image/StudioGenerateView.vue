<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { userGroupsAPI } from '@/api'
import { editImage, generateImage } from '@/api/image'
import StudioShell from '@/components/image/StudioShell.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore, useAuthStore, useImageStudioStore } from '@/stores'
import type { StudioSessionGeneration } from '@/stores/imageStudio'
import type { Group } from '@/types'
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

const ROUTE_STORAGE_KEY = 'studio_image_group_id'

const imageModelOptions: ImageModelOption[] = [
  {
    value: 'gemini-2.5-flash-image',
    label: 'Gemini 2.5 Flash Image',
    description: 'Gemini 快速路线'
  },
  {
    value: 'gemini-2.5-flash-image-preview',
    label: 'Gemini 2.5 Flash Image Preview',
    description: 'Gemini 预览路线'
  },
  {
    value: 'gemini-3.1-flash-image',
    label: 'Gemini 3.1 Flash Image',
    description: 'Gemini 高速路线'
  },
  {
    value: 'gemini-3.1-flash-image-preview',
    label: 'Gemini 3.1 Flash Image Preview',
    description: 'Gemini 3.1 预览路线'
  },
  {
    value: 'gemini-3-pro-image',
    label: 'Gemini 3 Pro Image',
    description: 'Gemini Pro 图片路线'
  },
  {
    value: 'gpt-image-1',
    label: 'GPT Image 1',
    description: 'OpenAI 图片路线'
  },
  {
    value: 'gpt-image-1.5',
    label: 'GPT Image 1.5',
    description: 'OpenAI 图片路线'
  },
  {
    value: 'gpt-image-2',
    label: 'GPT Image 2',
    description: 'OpenAI 新图片路线'
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
    value: '3840x2160',
    label: '3840x2160',
    description: '横图 / 4K',
    tier: '4K',
    aspectRatio: '16 / 9'
  },
  {
    value: '2160x3840',
    label: '2160x3840',
    description: '竖图 / 4K',
    tier: '4K',
    aspectRatio: '9 / 16'
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

const availableGroups = ref<Group[]>([])
const groupsLoading = ref(false)
const routeErrorMessage = ref('')
const generateErrorMessage = ref('')
const selectedGroupId = ref<number | null>(readStoredGroupId())
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
const previewCard = ref<StudioSessionGeneration | null>(null)
const activeIterationCard = ref<StudioSessionGeneration | null>(null)

let pendingController: AbortController | null = null
let storedGroupCandidate = readStoredGroupId()

const walletBalance = computed(() =>
  formatWalletMoneyFromInternal(authStore.user?.balance ?? 0, appStore.cachedPublicSettings)
)
const generationCards = computed(() => imageStudioStore.sessionGenerations)
const isSessionHydrating = computed(() => imageStudioStore.isHydrating)
const imageGroups = computed(() => availableGroups.value.filter((group) => isImageCapableGroup(group)))
const selectedGroup = computed(() => imageGroups.value.find((group) => group.id === selectedGroupId.value) ?? null)
const compatibleSizeOptions = computed(() =>
  imageSizeOptions.filter((item) => isSizeCompatibleWithModel(item.value, selectedModel.value))
)
const selectedSizeOption = computed(() =>
  compatibleSizeOptions.value.find((item) => item.value === selectedSize.value) ??
  compatibleSizeOptions.value[0] ??
  imageSizeOptions[0]
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
const routeSelectOptions = computed<SelectOption[]>(() =>
  imageGroups.value.map((group) => ({
    value: group.id,
    label: `${group.name} / ${platformLabel(group.platform)}`,
    disabled: !isGroupCompatibleWithModel(group, selectedModel.value)
  }))
)
const selectedGroupLabel = computed(() => {
  const group = selectedGroup.value
  return group ? `${group.name} / ${platformLabel(group.platform)}` : '未选择图片路线'
})
const estimatedCost = computed<number | null>(() => {
  const group = selectedGroup.value
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
    return '按路线配置扣费'
  }
  return formatWalletMoneyFromInternal(estimatedCost.value, appStore.cachedPublicSettings, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4
  })
})
const canGenerate = computed(() => {
  const group = selectedGroup.value
  return (
    !!group &&
    isGroupCompatibleWithModel(group, selectedModel.value) &&
    prompt.value.trim().length > 0 &&
    !isGenerating.value
  )
})
const submitButtonLabel = computed(() => {
  if (isGenerating.value) {
    return '生成中...'
  }
  return activeIterationCard.value && selectedGroup.value?.platform === 'openai' ? '继续编辑图片' : '生成图片'
})

watch(
  imageGroups,
  (groups) => {
    if (groups.length === 0) {
      selectedGroupId.value = null
      return
    }

    if (selectedGroupId.value && groups.some((group) => group.id === selectedGroupId.value)) {
      ensureModelCompatibleWithGroup(selectedGroup.value)
      return
    }

    if (storedGroupCandidate && groups.some((group) => group.id === storedGroupCandidate)) {
      selectedGroupId.value = storedGroupCandidate
      storedGroupCandidate = null
      return
    }

    selectedGroupId.value = groups.find((group) => isGroupCompatibleWithModel(group, selectedModel.value))?.id ?? groups[0].id
  },
  { immediate: true }
)

watch(selectedGroupId, (value) => {
  persistGroupId(value)
  ensureModelCompatibleWithGroup(selectedGroup.value)
})

watch(
  selectedModel,
  (value) => {
    ensureSizeCompatibleWithModel(value)
    selectCompatibleRouteForModel(value)
    persistStoredChoice(IMAGE_STUDIO_MODEL_STORAGE_KEY, value)
  },
  { immediate: true }
)

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

async function loadGroups() {
  groupsLoading.value = true
  routeErrorMessage.value = ''

  try {
    availableGroups.value = await userGroupsAPI.getAvailable()
  } catch {
    routeErrorMessage.value = '图片路线加载失败，请刷新后重试。'
    appStore.showError(routeErrorMessage.value)
  } finally {
    groupsLoading.value = false
  }
}

async function handleGenerate() {
  const group = selectedGroup.value
  const cleanPrompt = prompt.value.trim()

  if (!group || !cleanPrompt) {
    return
  }

  generateErrorMessage.value = ''
  pendingController?.abort()
  const controller = new AbortController()
  pendingController = controller
  isGenerating.value = true

  try {
    const editSource = activeIterationCard.value
    const shouldEditFromImage = !!editSource && group.platform === 'openai'
    const response = shouldEditFromImage
      ? await editImage({
          groupId: group.id,
          model: selectedModel.value,
          prompt: cleanPrompt,
          size: selectedSize.value,
          image: dataUrlToBlob(editSource.imageSrc),
          signal: controller.signal
        })
      : await generateImage({
          groupId: group.id,
          model: selectedModel.value,
          prompt: cleanPrompt,
          size: selectedSize.value,
          signal: controller.signal
        })

    const items = Array.isArray(response.data) ? response.data : []
    const renderItems = items.filter((item) => typeof item.b64_json === 'string' && item.b64_json.trim() !== '')
    const newCards: StudioSessionGeneration[] = renderItems.map((item, index) => ({
      id: `${response.created}-${index}-${Math.random().toString(36).slice(2, 8)}`,
      created: response.created,
      prompt: cleanPrompt,
      revisedPrompt: item.revised_prompt?.trim() || '',
      model: selectedModel.value,
      size: selectedSize.value,
      keyName: group.name,
      imageSrc: `data:image/png;base64,${item.b64_json}`
    }))

    if (newCards.length === 0) {
      throw new Error('服务端没有返回可渲染的图片数据。')
    }

    imageStudioStore.prependGenerations(newCards)
    if (shouldEditFromImage) {
      activeIterationCard.value = newCards[0]
    }
    appStore.showSuccess(`已生成 ${newCards.length} 张图片`)
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    const message = decorateGenerateError(
      error instanceof Error ? error.message : '图片生成失败。',
      group,
      selectedModel.value
    )
    generateErrorMessage.value = message
    appStore.showError(message)
  } finally {
    if (pendingController === controller) {
      pendingController = null
    }
    isGenerating.value = false
  }
}

function resolveCardIndex(card: StudioSessionGeneration): number {
  return Math.max(
    generationCards.value.findIndex((item) => item.id === card.id),
    0
  )
}

function downloadCard(card: StudioSessionGeneration, index = resolveCardIndex(card)) {
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

function openPreviewCard(card: StudioSessionGeneration) {
  previewCard.value = card
}

function closePreviewCard() {
  previewCard.value = null
}

async function reuseCardPrompt(card: StudioSessionGeneration) {
  activeIterationCard.value = card
  prompt.value = buildIterationPrompt(card)

  if (imageModelOptions.some((option) => option.value === card.model)) {
    selectedModel.value = card.model
  }
  if (imageSizeOptions.some((option) => option.value === card.size)) {
    selectedSize.value = card.size
  }
  const matchingGroup = imageGroups.value.find(
    (group) => group.name === card.keyName && isGroupCompatibleWithModel(group, card.model)
  )
  if (matchingGroup) {
    selectedGroupId.value = matchingGroup.id
  }

  previewCard.value = null
  await nextTick()
  promptInputRef.value?.focus()
  promptInputRef.value?.setSelectionRange(prompt.value.length, prompt.value.length)
  appStore.showSuccess('已载入左侧编辑区，可以继续调试')
}

function clearActiveIterationCard() {
  activeIterationCard.value = null
}

function buildIterationPrompt(card: StudioSessionGeneration): string {
  const cleanPrompt = card.prompt.trim()
  const suffix = '继续优化：保留主体构图和风格，调整：'
  if (!cleanPrompt) {
    return suffix
  }
  if (cleanPrompt.includes('继续优化：')) {
    return cleanPrompt
  }
  return `${cleanPrompt}\n\n${suffix}`
}

function copyCardPrompt(card: StudioSessionGeneration) {
  void copyToClipboard(card.prompt, '提示词已复制')
}

function copyCardStudioLink(card: StudioSessionGeneration) {
  void copyToClipboard(buildStudioLink(card.prompt, card.model, card.size), '配置链接已复制')
}

function copyCurrentStudioLink() {
  const cleanPrompt = prompt.value.trim()
  if (!cleanPrompt) {
    appStore.showError('请先填写提示词')
    return
  }

  void copyToClipboard(
    buildStudioLink(cleanPrompt, selectedModel.value, selectedSize.value),
    '配置链接已复制'
  )
}

function isImageCapableGroup(group?: Group | null): group is Group {
  return (
    !!group &&
    group.status === 'active' &&
    (group.platform === 'openai' || group.platform === 'gemini' || group.platform === 'antigravity')
  )
}

function isOpenAIImageModel(model: string): boolean {
  const normalized = model.toLowerCase()
  return normalized.startsWith('gpt-image-') || normalized === 'dall-e-2' || normalized === 'dall-e-3'
}

function isGPTImage2Model(model: string): boolean {
  return model.toLowerCase().trim() === 'gpt-image-2'
}

function isGeminiImageModel(model: string): boolean {
  const normalized = model.toLowerCase()
  return normalized.startsWith('gemini-') && normalized.includes('-image')
}

function isGroupCompatibleWithModel(group: Group, model: string): boolean {
  if (!isImageCapableGroup(group)) {
    return false
  }
  if (isOpenAIImageModel(model)) {
    return group.platform === 'openai'
  }
  if (isGeminiImageModel(model)) {
    return group.platform === 'gemini' || group.platform === 'antigravity'
  }
  return false
}

function isSizeCompatibleWithModel(size: string, model: string): boolean {
  const normalizedSize = size.toLowerCase().trim()
  if (isOpenAIImageModel(model)) {
    if (normalizedSize === '4096x4096') {
      return false
    }
    if (normalizedSize === '3840x2160' || normalizedSize === '2160x3840') {
      return isGPTImage2Model(model)
    }
    return true
  }
  return normalizedSize !== '3840x2160' && normalizedSize !== '2160x3840'
}

function ensureSizeCompatibleWithModel(model: string) {
  if (isSizeCompatibleWithModel(selectedSize.value, model)) {
    return
  }
  selectedSize.value = '2048x2048'
}

function ensureModelCompatibleWithGroup(group?: Group | null) {
  if (!group || isGroupCompatibleWithModel(group, selectedModel.value)) {
    return
  }
  const preferredModel = preferredModelForGroup(group)
  if (preferredModel) {
    selectedModel.value = preferredModel
  }
}

function preferredModelForGroup(group: Group): string | null {
  const preferredOrder =
    group.platform === 'openai'
      ? ['gpt-image-1', 'gpt-image-1.5', 'gpt-image-2']
      : [
          'gemini-3.1-flash-image',
          'gemini-3.1-flash-image-preview',
          'gemini-3-pro-image',
          'gemini-2.5-flash-image',
          'gemini-2.5-flash-image-preview'
        ]

  return (
    preferredOrder.find((model) => isGroupCompatibleWithModel(group, model)) ??
    imageModelOptions.find((option) => isGroupCompatibleWithModel(group, option.value))?.value ??
    null
  )
}

function selectCompatibleRouteForModel(model: string) {
  const group = selectedGroup.value
  if (group && isGroupCompatibleWithModel(group, model)) {
    return
  }
  const compatibleGroup = imageGroups.value.find((item) => isGroupCompatibleWithModel(item, model))
  if (compatibleGroup) {
    selectedGroupId.value = compatibleGroup.id
  }
}

function decorateGenerateError(message: string, group: Group, model: string): string {
  const lowerMessage = message.toLowerCase()
  if (group.platform === 'antigravity') {
    if (lowerMessage.includes('requested entity was not found')) {
      return `当前 Antigravity 上游不支持 ${model}，请切换 Gemini 图片模型或选择 OpenAI 图片路线。`
    }
    if (lowerMessage.includes('503') || lowerMessage.includes('capacity') || lowerMessage.includes('failover')) {
      return '当前 Antigravity 图片路线容量不足，请稍后重试，或切换到 OpenAI 图片路线。'
    }
  }
  if (lowerMessage.includes('selected image route is not compatible')) {
    return `当前模型需要 ${modelRouteRequirementLabel(model)}，请切换路线后重试。`
  }
  return message
}

function modelRouteRequirementLabel(model: string): string {
  if (isOpenAIImageModel(model)) {
    return 'OpenAI 图片路线'
  }
  if (isGeminiImageModel(model)) {
    return 'Gemini 或 Antigravity 图片路线'
  }
  return '图片路线'
}

function platformLabel(platform?: string): string {
  switch (platform) {
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'openai':
      return 'OpenAI'
    default:
      return '未知平台'
  }
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

function dataUrlToBlob(dataUrl: string): Blob {
  const commaIndex = dataUrl.indexOf(',')
  if (!dataUrl.startsWith('data:') || commaIndex < 0) {
    throw new Error('无法读取来源图片')
  }
  const metadata = dataUrl.slice(0, commaIndex)
  const mimeMatch = metadata.match(/^data:([^;]+);base64$/i)
  if (!mimeMatch) {
    throw new Error('来源图片格式不支持')
  }
  const binary = window.atob(dataUrl.slice(commaIndex + 1))
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new Blob([bytes], { type: mimeMatch[1] })
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

function readStoredGroupId(): number | null {
  if (typeof window === 'undefined') {
    return null
  }
  const raw = window.localStorage.getItem(ROUTE_STORAGE_KEY)
  const numeric = Number(raw)
  return Number.isInteger(numeric) && numeric > 0 ? numeric : null
}

function persistGroupId(value: number | null) {
  if (typeof window === 'undefined') {
    return
  }
  if (value == null) {
    window.localStorage.removeItem(ROUTE_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(ROUTE_STORAGE_KEY, String(value))
}

function handleWindowKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && previewCard.value) {
    closePreviewCard()
  }
}

onMounted(() => {
  void imageStudioStore.ensureHydrated()
  void loadGroups()
  window.addEventListener('keydown', handleWindowKeydown)
})

onUnmounted(() => {
  pendingController?.abort()
  window.removeEventListener('keydown', handleWindowKeydown)
})
</script>

<template>
  <main class="generate-shell">
    <StudioShell max-width="1180px" />

    <section class="workspace-grid">
      <form class="composer-panel" @submit.prevent="handleGenerate">
        <div class="panel-heading">
          <div>
            <p class="eyebrow">图片生成</p>
            <h1>生成图片</h1>
          </div>
          <button class="utility-button" type="button" @click="copyCurrentStudioLink">
            复制配置链接
          </button>
        </div>

        <div class="field-block">
          <label class="field-label">图片路线</label>
          <Select
            v-model="selectedGroupId"
            :options="routeSelectOptions"
            :disabled="groupsLoading || routeSelectOptions.length === 0"
            :searchable="routeSelectOptions.length > 8"
            placeholder="选择图片生成路线"
            search-placeholder="搜索路线或分组"
            empty-text="没有可用图片路线"
          />
          <p v-if="selectedGroup" class="field-note">当前路线：{{ selectedGroupLabel }}</p>
          <p v-else-if="groupsLoading" class="field-note">正在加载图片路线...</p>
          <p v-else class="field-note">当前账号没有可用图片路线。</p>
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
            <span>可用路线</span>
            <strong>{{ imageGroups.length }}</strong>
          </div>
        </div>

        <div class="field-block">
          <label class="field-label">模型</label>
          <Select
            v-model="selectedModel"
            :options="modelSelectOptions"
            searchable
            placeholder="选择模型"
            search-placeholder="搜索模型"
          />
          <p class="field-note">{{ selectedModelOption.description }}</p>
        </div>

        <div class="field-block">
          <div class="field-label-row">
            <label class="field-label">尺寸</label>
            <span class="route-badge">{{ selectedSizeOption.description }}</span>
          </div>
          <div class="size-grid">
            <button
              v-for="item in imageSizeOptions"
              :key="item.value"
              type="button"
              :class="['size-button', selectedSize === item.value && 'size-button-active']"
              :disabled="!isSizeCompatibleWithModel(item.value, selectedModel)"
              @click="selectedSize = item.value"
            >
              <strong>{{ item.label }}</strong>
              <span>{{ item.tier }}</span>
            </button>
          </div>
        </div>

        <div class="field-block prompt-block">
          <div class="field-label-row">
            <label class="field-label" for="image-prompt">提示词</label>
            <span class="route-badge">{{ prompt.trim().length }} 字符</span>
          </div>
          <textarea
            id="image-prompt"
            ref="promptInputRef"
            v-model="prompt"
            class="prompt-input"
            rows="8"
            placeholder="描述要生成的画面、风格、构图和用途"
          />
          <div v-if="activeIterationCard" class="iteration-source">
            <button
              class="iteration-thumb-button"
              type="button"
              aria-label="查看当前迭代来源图片"
              @click="openPreviewCard(activeIterationCard)"
            >
              <img :src="activeIterationCard.imageSrc" :alt="activeIterationCard.prompt" class="iteration-thumb" />
            </button>
            <div class="iteration-copy">
              <strong>继续调试这张图</strong>
              <span>{{ activeIterationCard.model }} / {{ activeIterationCard.size }} / {{ activeIterationCard.keyName }}</span>
            </div>
            <button class="iteration-clear" type="button" @click="clearActiveIterationCard">清除</button>
          </div>
          <div class="suggestion-list">
            <button
              v-for="suggestion in promptSuggestions"
              :key="suggestion"
              type="button"
              @click="choosePromptSuggestion(suggestion)"
            >
              {{ suggestion }}
            </button>
          </div>
        </div>

        <div v-if="routeErrorMessage" class="alert-box alert-warning">
          {{ routeErrorMessage }}
        </div>
        <div v-if="generateErrorMessage" class="alert-box alert-error">
          {{ generateErrorMessage }}
        </div>
        <div v-if="!groupsLoading && imageGroups.length === 0" class="alert-box alert-warning">
          当前账号没有可用图片路线，请在主站分组里开通 OpenAI、Gemini 或 Antigravity 图片分组。
        </div>

        <div class="submit-row">
          <button class="primary-button" type="submit" :disabled="!canGenerate">
            {{ submitButtonLabel }}
          </button>
          <button class="secondary-button" type="button" :disabled="groupsLoading" @click="loadGroups">
            刷新路线
          </button>
        </div>
      </form>

      <aside class="result-panel">
        <div class="panel-heading">
          <div>
            <p class="eyebrow">结果</p>
            <h2>图片预览</h2>
          </div>
          <div class="panel-actions">
            <RouterLink class="utility-button" to="/studio/history">历史记录</RouterLink>
            <RouterLink class="utility-button" to="/studio/pricing">价格</RouterLink>
          </div>
        </div>

        <div v-if="isGenerating" class="rendering-state">
          <div class="loading-preview" :style="{ aspectRatio: selectedSizeOption.aspectRatio }">
            <span>正在生成</span>
          </div>
        </div>

        <div v-else-if="isSessionHydrating" class="empty-state">
          <strong>正在恢复最近图片</strong>
        </div>

        <div v-else-if="generationCards.length === 0" class="empty-state">
          <strong>还没有生成结果</strong>
          <p>选择图片路线并填写提示词后，结果会显示在这里。</p>
        </div>

        <div v-else class="gallery-grid">
          <article
            v-for="(card, index) in generationCards"
            :key="card.id"
            class="render-card"
          >
            <button
              class="render-image-button"
              type="button"
              :style="{ aspectRatio: resolveAspectRatio(card.size) }"
              aria-label="放大查看图片"
              @click="openPreviewCard(card)"
            >
              <img
                :src="card.thumbnailSrc || card.imageSrc"
                :alt="card.prompt"
                class="render-image"
              />
              <span class="render-image-action">放大查看</span>
            </button>
            <div class="render-meta">
              <div>
                <strong>{{ card.model }}</strong>
                <span>{{ card.size }} / {{ card.keyName }} / {{ formatCreatedAt(card.created) }}</span>
              </div>
              <div class="render-actions">
                <button type="button" @click="reuseCardPrompt(card)">继续编辑</button>
                <button type="button" @click="copyCardPrompt(card)">提示词</button>
                <button type="button" @click="copyCardStudioLink(card)">链接</button>
                <button type="button" @click="downloadCard(card, index)">下载</button>
              </div>
            </div>
          </article>
        </div>
      </aside>
    </section>

    <div v-if="previewCard" class="image-lightbox" @click.self="closePreviewCard">
      <section class="lightbox-panel" role="dialog" aria-modal="true" aria-label="查看生成图片">
        <div class="lightbox-stage">
          <img :src="previewCard.imageSrc" :alt="previewCard.prompt" class="lightbox-image" />
        </div>
        <aside class="lightbox-sidebar">
          <div class="lightbox-header">
            <div>
              <p class="eyebrow">图片结果</p>
              <h2>{{ previewCard.model }}</h2>
            </div>
            <button class="lightbox-close" type="button" aria-label="关闭预览" @click="closePreviewCard">×</button>
          </div>
          <dl class="lightbox-meta">
            <div>
              <dt>尺寸</dt>
              <dd>{{ previewCard.size }}</dd>
            </div>
            <div>
              <dt>路线</dt>
              <dd>{{ previewCard.keyName }}</dd>
            </div>
            <div>
              <dt>时间</dt>
              <dd>{{ formatCreatedAt(previewCard.created) }}</dd>
            </div>
          </dl>
          <div class="lightbox-prompt">
            <strong>提示词</strong>
            <p>{{ previewCard.prompt }}</p>
          </div>
          <div v-if="previewCard.revisedPrompt" class="lightbox-prompt">
            <strong>上游优化提示词</strong>
            <p>{{ previewCard.revisedPrompt }}</p>
          </div>
          <div class="lightbox-actions">
            <button class="primary-button" type="button" @click="reuseCardPrompt(previewCard)">继续编辑</button>
            <button class="secondary-button" type="button" @click="copyCardPrompt(previewCard)">复制提示词</button>
            <button class="secondary-button" type="button" @click="copyCardStudioLink(previewCard)">复制链接</button>
            <button class="secondary-button" type="button" @click="downloadCard(previewCard)">下载</button>
          </div>
        </aside>
      </section>
    </div>
  </main>
</template>

<style scoped>
.generate-shell {
  min-height: 100vh;
  padding: 52px 20px 64px;
  background:
    linear-gradient(180deg, rgba(239, 246, 243, 0.88) 0%, rgba(250, 250, 247, 0.92) 36%, #ffffff 100%);
  color: #10231f;
}

.workspace-grid {
  width: min(1180px, 100%);
  margin: 0 auto;
  display: grid;
  grid-template-columns: minmax(340px, 0.95fr) minmax(0, 1.45fr);
  gap: 16px;
  align-items: start;
}

.composer-panel,
.result-panel {
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 18px 42px rgba(16, 35, 31, 0.08);
}

.composer-panel {
  padding: 18px;
}

.result-panel {
  padding: 18px;
  min-height: 310px;
}

.panel-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 18px;
}

.panel-heading h1,
.panel-heading h2 {
  margin: 4px 0 0;
  font-size: clamp(1.55rem, 2vw, 2.05rem);
  line-height: 1.06;
  font-weight: 800;
}

.eyebrow {
  margin: 0;
  color: #0f766e;
  font-size: 0.86rem;
  font-weight: 800;
  letter-spacing: 0;
}

.field-block {
  margin-top: 16px;
}

.field-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.field-label {
  display: block;
  margin-bottom: 8px;
  color: #10231f;
  font-size: 0.95rem;
  font-weight: 800;
}

.field-label-row .field-label {
  margin-bottom: 0;
}

.field-note {
  margin: 8px 0 0;
  color: rgba(16, 35, 31, 0.66);
  font-size: 0.9rem;
  line-height: 1.5;
}

.route-badge {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 6px;
  background: rgba(16, 35, 31, 0.07);
  color: rgba(16, 35, 31, 0.76);
  font-size: 0.82rem;
  font-weight: 800;
}

.compact-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 16px;
}

.compact-stats div {
  border: 1px solid rgba(16, 35, 31, 0.1);
  border-radius: 8px;
  background: #f8faf8;
  padding: 12px;
  min-height: 72px;
}

.compact-stats span {
  display: block;
  color: rgba(16, 35, 31, 0.58);
  font-size: 0.82rem;
}

.compact-stats strong {
  display: block;
  margin-top: 8px;
  font-size: 1.04rem;
  line-height: 1.25;
}

.size-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.size-button {
  min-height: 52px;
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: #ffffff;
  color: #10231f;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 12px;
  cursor: pointer;
}

.size-button strong,
.size-button span {
  font-size: 0.95rem;
}

.size-button span {
  color: rgba(16, 35, 31, 0.58);
}

.size-button-active {
  border-color: rgba(15, 118, 110, 0.58);
  background: rgba(15, 118, 110, 0.08);
}

.size-button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.prompt-input {
  width: 100%;
  min-height: 190px;
  resize: vertical;
  border: 1px solid rgba(16, 35, 31, 0.14);
  border-radius: 8px;
  background: #ffffff;
  color: #10231f;
  padding: 14px;
  font-size: 1rem;
  line-height: 1.65;
  outline: none;
}

.prompt-input:focus {
  border-color: rgba(15, 118, 110, 0.64);
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}

.iteration-source {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  padding: 10px;
  border: 1px solid rgba(15, 118, 110, 0.22);
  border-radius: 8px;
  background: rgba(15, 118, 110, 0.06);
}

.iteration-thumb-button {
  width: 72px;
  height: 72px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #edf4f2;
  cursor: zoom-in;
}

.iteration-thumb {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.iteration-copy {
  min-width: 0;
}

.iteration-copy strong,
.iteration-copy span {
  display: block;
}

.iteration-copy span {
  margin-top: 4px;
  color: rgba(16, 35, 31, 0.62);
  font-size: 0.84rem;
  line-height: 1.4;
}

.iteration-clear {
  min-height: 32px;
  padding: 0 10px;
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: #ffffff;
  color: #10231f;
  font-weight: 800;
  cursor: pointer;
}

.suggestion-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.suggestion-list button {
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: #ffffff;
  color: #10231f;
  padding: 10px 12px;
  text-align: left;
  line-height: 1.45;
  cursor: pointer;
}

.submit-row,
.panel-actions,
.render-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.submit-row {
  margin-top: 18px;
}

.primary-button,
.secondary-button,
.utility-button,
.render-actions button {
  min-height: 40px;
  border-radius: 8px;
  padding: 0 14px;
  border: 1px solid rgba(16, 35, 31, 0.12);
  font-weight: 800;
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.primary-button {
  flex: 1;
  min-width: 180px;
  background: #0f766e;
  border-color: #0f766e;
  color: #ffffff;
}

.secondary-button,
.utility-button,
.render-actions button {
  background: #ffffff;
  color: #10231f;
}

.primary-button:disabled,
.secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.56;
}

.alert-box {
  margin-top: 14px;
  border-radius: 8px;
  padding: 12px 14px;
  font-weight: 700;
  line-height: 1.5;
}

.alert-warning {
  border: 1px solid rgba(185, 116, 20, 0.24);
  background: #fff7e8;
  color: #82440c;
}

.alert-error {
  border: 1px solid rgba(190, 18, 60, 0.24);
  background: #fff0f3;
  color: #9f1239;
}

.empty-state,
.loading-preview {
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: #fbfcfb;
  min-height: 142px;
  display: grid;
  place-items: center;
  text-align: center;
  padding: 24px;
}

.empty-state strong,
.loading-preview span {
  display: block;
  font-size: 1.05rem;
}

.empty-state p {
  margin: 8px 0 0;
  color: rgba(16, 35, 31, 0.64);
}

.loading-preview {
  min-height: 300px;
  color: #0f766e;
  background:
    linear-gradient(90deg, rgba(15, 118, 110, 0.08), rgba(185, 116, 20, 0.08), rgba(15, 118, 110, 0.08));
  background-size: 220% 100%;
  animation: studio-loading 1.6s ease-in-out infinite;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.render-card {
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  overflow: hidden;
  background: #ffffff;
}

.render-image-button {
  position: relative;
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  overflow: hidden;
  background: #f3f6f4;
  cursor: zoom-in;
}

.render-image {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: #f3f6f4;
}

.render-image-action {
  position: absolute;
  right: 10px;
  bottom: 10px;
  min-height: 28px;
  padding: 0 9px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.9);
  color: #10231f;
  display: inline-flex;
  align-items: center;
  font-size: 0.78rem;
  font-weight: 800;
  box-shadow: 0 8px 20px rgba(16, 35, 31, 0.14);
  opacity: 0;
  transform: translateY(4px);
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.render-image-button:hover .render-image-action,
.render-image-button:focus-visible .render-image-action {
  opacity: 1;
  transform: translateY(0);
}

.render-meta {
  padding: 12px;
}

.render-meta strong,
.render-meta span {
  display: block;
}

.render-meta span {
  margin-top: 4px;
  color: rgba(16, 35, 31, 0.62);
  font-size: 0.84rem;
  line-height: 1.45;
}

.render-actions {
  margin-top: 10px;
}

.render-actions button {
  min-height: 32px;
  padding: 0 10px;
  font-size: 0.82rem;
}

.image-lightbox {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(8, 15, 14, 0.72);
}

.lightbox-panel {
  width: min(1180px, 100%);
  max-height: calc(100vh - 48px);
  border-radius: 8px;
  background: #ffffff;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  overflow: hidden;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.32);
}

.lightbox-stage {
  min-height: 420px;
  background: #0b1210;
  display: grid;
  place-items: center;
  overflow: auto;
}

.lightbox-image {
  display: block;
  max-width: 100%;
  max-height: calc(100vh - 48px);
  object-fit: contain;
}

.lightbox-sidebar {
  min-width: 0;
  padding: 18px;
  overflow: auto;
}

.lightbox-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.lightbox-header h2 {
  margin: 4px 0 0;
  font-size: 1.25rem;
  line-height: 1.2;
}

.lightbox-close {
  width: 36px;
  height: 36px;
  border: 1px solid rgba(16, 35, 31, 0.12);
  border-radius: 8px;
  background: #ffffff;
  color: rgba(16, 35, 31, 0.68);
  font-size: 1.2rem;
  cursor: pointer;
}

.lightbox-meta {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
  margin: 18px 0 0;
}

.lightbox-meta div {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid rgba(16, 35, 31, 0.08);
}

.lightbox-meta dt,
.lightbox-meta dd {
  margin: 0;
}

.lightbox-meta dt {
  color: rgba(16, 35, 31, 0.58);
}

.lightbox-meta dd {
  color: #10231f;
  font-weight: 800;
  overflow-wrap: anywhere;
}

.lightbox-prompt {
  margin-top: 16px;
}

.lightbox-prompt strong {
  display: block;
  margin-bottom: 6px;
}

.lightbox-prompt p {
  margin: 0;
  color: rgba(16, 35, 31, 0.72);
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.lightbox-actions {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
  margin-top: 18px;
}

.lightbox-actions .primary-button,
.lightbox-actions .secondary-button {
  width: 100%;
  min-width: 0;
}

@keyframes studio-loading {
  0% {
    background-position: 0% 50%;
  }
  100% {
    background-position: 100% 50%;
  }
}

@media (max-width: 980px) {
  .workspace-grid {
    grid-template-columns: 1fr;
  }

  .lightbox-panel {
    grid-template-columns: 1fr;
  }

  .lightbox-stage {
    min-height: 260px;
  }
}

@media (max-width: 640px) {
  .generate-shell {
    padding: 24px 12px 44px;
  }

  .panel-heading {
    display: grid;
  }

  .compact-stats,
  .size-grid,
  .gallery-grid {
    grid-template-columns: 1fr;
  }

  .iteration-source {
    grid-template-columns: 56px minmax(0, 1fr);
  }

  .iteration-thumb-button {
    width: 56px;
    height: 56px;
  }

  .iteration-clear {
    grid-column: 1 / -1;
  }

  .image-lightbox {
    padding: 10px;
  }

  .lightbox-panel {
    max-height: calc(100vh - 20px);
  }

  .lightbox-sidebar {
    padding: 14px;
  }
}
</style>
