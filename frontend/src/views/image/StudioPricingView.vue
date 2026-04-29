<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { userGroupsAPI } from '@/api/groups'
import StudioShell from '@/components/image/StudioShell.vue'
import { useAppStore, useAuthStore } from '@/stores'
import type { Group } from '@/types'
import { resolveMainSiteURL } from '@/utils/siteMode'
import { formatWalletMoneyFromInternal } from '@/utils/walletDisplay'

type PriceKey = 'image_price_1k' | 'image_price_2k' | 'image_price_4k'

interface PriceTier {
  key: PriceKey
  label: string
  detail: string
}

const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const priceTiers: PriceTier[] = [
  { key: 'image_price_1k', label: '1K', detail: '1024x1024 默认方图' },
  { key: 'image_price_2k', label: '2K', detail: '1536x1024、1024x1536、2048x2048' },
  { key: 'image_price_4k', label: '4K', detail: '4096x4096 大图' }
]

const availableGroups = ref<Group[]>([])
const loading = ref(false)
const errorMessage = ref('')

let activeRequestId = 0

const mainSiteKeysURL = computed(() => resolveMainSiteURL('/keys'))
const mainSiteUsageURL = computed(() => resolveMainSiteURL('/usage'))

const readyGroups = computed(() =>
  [...availableGroups.value]
    .filter((group) => isReadyImageGroup(group))
    .sort(compareGroups)
)

const blockedGroups = computed(() =>
  [...availableGroups.value]
    .filter((group) => !isReadyImageGroup(group))
    .sort(compareGroups)
)

const totalPricedGroups = computed(() =>
  availableGroups.value.filter((group) => hasAnyImagePrice(group)).length
)

const lowestEntryPrice = computed<number | null>(() => {
  const prices = readyGroups.value
    .map((group) => group.image_price_1k)
    .filter((value): value is number => typeof value === 'number' && Number.isFinite(value))

  if (prices.length === 0) {
    return null
  }

  return Math.min(...prices)
})

const primaryActionLabel = computed(() =>
  authStore.isAuthenticated ? '去生成图片' : '登录后生成'
)

const surfaceStatusLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return '登录后显示账号价格'
  }
  if (loading.value) {
    return '正在加载可用分组'
  }
  return `${readyGroups.value.length} 条可用路由`
})

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated) => {
    if (!isAuthenticated) {
      activeRequestId += 1
      availableGroups.value = []
      loading.value = false
      errorMessage.value = ''
      return
    }

    loadAvailableGroups()
  },
  { immediate: true }
)

async function loadAvailableGroups() {
  if (!authStore.isAuthenticated) {
    return
  }

  const requestId = ++activeRequestId
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await userGroupsAPI.getAvailable()
    if (requestId !== activeRequestId) {
      return
    }
    availableGroups.value = response
  } catch {
    if (requestId !== activeRequestId) {
      return
    }
    errorMessage.value = '可用分组加载失败。'
    appStore.showError(errorMessage.value)
  } finally {
    if (requestId === activeRequestId) {
      loading.value = false
    }
  }
}

async function handlePrimaryAction() {
  await router.push('/studio/generate')
}

function hasAnyImagePrice(group: Group): boolean {
  return priceTiers.some(({ key }) => typeof group[key] === 'number' && Number.isFinite(group[key]))
}

function isReadyImageGroup(group: Group): boolean {
  return group.status === 'active' && hasAnyImagePrice(group)
}

function compareGroups(left: Group, right: Group): number {
  const leftPrice = left.image_price_1k ?? Number.MAX_SAFE_INTEGER
  const rightPrice = right.image_price_1k ?? Number.MAX_SAFE_INTEGER
  if (leftPrice !== rightPrice) {
    return leftPrice - rightPrice
  }
  return left.name.localeCompare(right.name)
}

function formatPrice(value: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return '未配置'
  }

  return formatWalletMoneyFromInternal(value, appStore.cachedPublicSettings, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4
  })
}

function platformLabel(platform?: string): string {
  switch (platform) {
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'openai':
      return 'OpenAI'
    case 'sora':
      return 'Sora'
    default:
      return '未知'
  }
}

function accessLabel(group: Group): string {
  if (group.subscription_type === 'subscription') {
    return '订阅'
  }
  if (group.is_exclusive) {
    return '白名单'
  }
  return '可绑定'
}

function routeStatus(group: Group): string {
  if (group.status !== 'active') {
    return '分组未启用'
  }
  if (!hasAnyImagePrice(group)) {
    return '未配置图片价格'
  }
  return '可用于 /v1/images/generations'
}
</script>

<template>
  <main class="pricing-shell">
    <StudioShell max-width="1160px" />

    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">价格与路由</p>
        <h1>图片价格矩阵</h1>
        <p class="lead">
          对比当前账号可用图片分组的 1K、2K、4K 价格，并确认哪些路由现在可用。
        </p>

        <div class="hero-actions">
          <button class="primary-button" type="button" @click="handlePrimaryAction">
            {{ primaryActionLabel }}
          </button>
          <a class="ghost-link" :href="mainSiteKeysURL">配置 API Key</a>
          <RouterLink class="ghost-link" to="/studio/history">历史记录</RouterLink>
        </div>
      </div>

      <aside class="hero-panel">
        <span class="panel-tag">当前账号</span>
        <dl class="panel-stats">
          <div>
            <dt>已配置价格</dt>
            <dd>{{ authStore.isAuthenticated ? totalPricedGroups : '--' }}</dd>
          </div>
          <div>
            <dt>可用路由</dt>
            <dd>{{ authStore.isAuthenticated ? readyGroups.length : '--' }}</dd>
          </div>
          <div>
            <dt>最低 1K 价格</dt>
            <dd>{{ authStore.isAuthenticated ? formatPrice(lowestEntryPrice) : '请登录' }}</dd>
          </div>
          <div>
            <dt>状态</dt>
            <dd>{{ surfaceStatusLabel }}</dd>
          </div>
        </dl>
      </aside>
    </section>

    <section v-if="!authStore.isAuthenticated" class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">登录后查看</p>
          <h2>账号价格取决于可用分组</h2>
          <p class="copy">
            图片价格来自当前账号可绑定的分组。登录后会显示真实路由和价格。
          </p>
        </div>
        <button class="primary-button" type="button" @click="handlePrimaryAction">
          登录并加载价格
        </button>
      </div>
    </section>

    <section v-else class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">可用分组</p>
          <h2>图片路由价格</h2>
          <p class="copy">
            下列分组已对当前账号开放，并且可以用于图片生成。
          </p>
        </div>
        <div class="panel-actions">
          <button class="ghost-button" type="button" :disabled="loading" @click="loadAvailableGroups">
            刷新分组
          </button>
          <a class="ghost-link" :href="mainSiteUsageURL">用量明细</a>
        </div>
      </div>

      <div v-if="errorMessage" class="alert-box">
        {{ errorMessage }}
      </div>

      <div v-if="loading" class="pricing-grid">
        <div class="pricing-card pricing-skeleton" />
        <div class="pricing-card pricing-skeleton" />
        <div class="pricing-card pricing-skeleton" />
      </div>

      <div v-else-if="readyGroups.length === 0" class="empty-state">
        <strong>当前账号暂无可用图片路由</strong>
        <p>
          可用分组可能未配置图片价格、未启用，或者上游图片能力暂未接入。
        </p>
      </div>

      <div v-else class="pricing-grid">
        <article
          v-for="group in readyGroups"
          :key="group.id"
          class="pricing-card"
        >
          <div class="card-top">
            <div>
              <span class="platform-pill">{{ platformLabel(group.platform) }}</span>
              <h3>{{ group.name }}</h3>
            </div>
            <span class="access-pill">{{ accessLabel(group) }}</span>
          </div>

          <p class="card-copy">
            {{ group.description || routeStatus(group) }}
          </p>

          <div class="tier-list">
            <div
              v-for="tier in priceTiers"
              :key="tier.key"
              class="tier-row"
            >
              <div>
                <strong>{{ tier.label }}</strong>
                <span>{{ tier.detail }}</span>
              </div>
              <b>{{ formatPrice(group[tier.key]) }}</b>
            </div>
          </div>

          <div class="card-footer">
            <span>{{ routeStatus(group) }}</span>
            <RouterLink class="inline-link" to="/studio/generate">生成</RouterLink>
          </div>
        </article>
      </div>
    </section>

    <section v-if="authStore.isAuthenticated && blockedGroups.length > 0" class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">不可用分组</p>
          <h2>当前可见但不能生成</h2>
          <p class="copy">
            这些分组对账号可见，但暂时不能处理图片生成。
          </p>
        </div>
      </div>

      <div class="blocked-list">
        <article
          v-for="group in blockedGroups"
          :key="group.id"
          class="blocked-card"
        >
          <div>
            <span class="platform-pill">{{ platformLabel(group.platform) }}</span>
            <h3>{{ group.name }}</h3>
            <p>{{ group.description || routeStatus(group) }}</p>
          </div>
          <div class="blocked-side">
            <strong>{{ routeStatus(group) }}</strong>
            <span>{{ accessLabel(group) }}</span>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>

<style scoped>
.pricing-shell {
  --paper: #f6f7f5;
  --panel: #ffffff;
  --card: #ffffff;
  --ink: #12271f;
  --muted: rgba(18, 39, 31, 0.7);
  --line: rgba(18, 39, 31, 0.12);
  --accent: #0f766e;
  --accent-soft: #d8f0e8;
  --accent-strong: #0b5f59;
  min-height: 100vh;
  padding: 24px 24px 56px;
  background: linear-gradient(180deg, #fbfcfb 0%, var(--paper) 100%);
  color: var(--ink);
  font-family: "Space Grotesk", "Noto Sans SC", sans-serif;
}

.hero,
.notes-grid,
.panel {
  width: min(1160px, 100%);
  margin: 0 auto;
}

.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(300px, 0.8fr);
  gap: 16px;
  align-items: stretch;
}

.hero-copy,
.hero-panel,
.note-card,
.panel {
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  box-shadow: 0 12px 32px rgba(18, 39, 31, 0.06);
}

.hero-copy,
.hero-panel,
.panel,
.note-card {
  padding: 18px;
}

.eyebrow {
  margin: 0 0 6px;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--accent-strong);
}

.hero-copy h1,
.panel-header h2 {
  margin: 0;
  line-height: 1.18;
}

.hero-copy h1 {
  font-size: 1.65rem;
}

.lead,
.copy,
.card-copy,
.blocked-card p,
.tier-row span {
  color: var(--muted);
}

.lead {
  margin: 8px 0 0;
  max-width: 42rem;
  line-height: 1.6;
  font-size: 0.95rem;
}

.hero-actions,
.panel-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.hero-actions {
  margin-top: 24px;
}

.primary-button,
.ghost-button,
.ghost-link,
.inline-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 46px;
  padding: 0 18px;
  border-radius: 8px;
  text-decoration: none;
  font: inherit;
  font-weight: 700;
  color: inherit;
  transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.primary-button {
  border: none;
  background: var(--accent);
  color: #f8fffd;
  box-shadow: 0 12px 24px rgba(15, 118, 110, 0.18);
  cursor: pointer;
}

.ghost-button,
.ghost-link,
.inline-link {
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.58);
}

.primary-button:hover,
.ghost-button:hover,
.ghost-link:hover,
.inline-link:hover {
  transform: translateY(-1px);
}

.primary-button:disabled,
.ghost-button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.panel-tag,
.platform-pill,
.access-pill {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  font-size: 0.78rem;
  font-weight: 700;
}

.panel-tag,
.platform-pill {
  background: var(--accent-soft);
  color: var(--accent-strong);
}

.access-pill {
  border: 1px solid var(--line);
  background: rgba(18, 39, 31, 0.05);
}

.panel-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 16px 0 0;
}

.panel-stats div {
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: rgba(18, 39, 31, 0.03);
}

.panel-stats dt {
  font-size: 0.8rem;
  color: rgba(18, 39, 31, 0.55);
}

.panel-stats dd {
  margin: 8px 0 0;
  font-size: 1.08rem;
  line-height: 1.5;
}

.notes-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.note-card h2,
.pricing-card h3,
.blocked-card h3 {
  margin: 0;
}

.note-card p {
  margin: 10px 0 0;
  line-height: 1.68;
  color: var(--muted);
}

.panel {
  margin-top: 16px;
}

.panel-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
}

.panel-header h2 {
  font-size: 1.35rem;
}

.copy {
  margin: 8px 0 0;
  max-width: 46rem;
  line-height: 1.6;
  font-size: 0.95rem;
}

.alert-box {
  margin-top: 18px;
  padding: 14px 16px;
  border-radius: 8px;
  background: rgba(163, 42, 42, 0.1);
  color: #7b251e;
}

.pricing-grid,
.blocked-list {
  display: grid;
  gap: 16px;
  margin-top: 22px;
}

.pricing-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.pricing-card,
.blocked-card,
.empty-state {
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--card);
}

.pricing-card,
.blocked-card {
  padding: 22px;
}

.pricing-skeleton {
  min-height: 340px;
  background:
    linear-gradient(110deg, rgba(255, 255, 255, 0.28) 8%, rgba(15, 122, 109, 0.16) 18%, rgba(255, 255, 255, 0.26) 33%),
    rgba(255, 255, 255, 0.72);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

.card-top,
.card-footer,
.blocked-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.card-top h3,
.blocked-card h3 {
  margin-top: 12px;
  font-size: 1.14rem;
}

.card-copy {
  margin: 14px 0 0;
  min-height: 3.2rem;
  line-height: 1.66;
}

.tier-list {
  display: grid;
  gap: 10px;
  margin-top: 18px;
}

.tier-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-radius: 8px;
  background: rgba(18, 39, 31, 0.04);
}

.tier-row strong {
  display: block;
  font-size: 0.96rem;
}

.tier-row span {
  display: block;
  margin-top: 6px;
  font-size: 0.84rem;
  line-height: 1.5;
}

.tier-row b {
  font-size: 0.95rem;
  text-align: right;
}

.card-footer {
  margin-top: 18px;
  font-size: 0.88rem;
  color: rgba(18, 39, 31, 0.6);
}

.empty-state {
  margin-top: 22px;
  padding: 24px;
}

.empty-state strong {
  display: block;
  font-size: 1.08rem;
}

.empty-state p {
  margin: 12px 0 0;
  line-height: 1.68;
  color: var(--muted);
}

.blocked-card p {
  margin: 12px 0 0;
  line-height: 1.68;
}

.blocked-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  text-align: right;
}

.blocked-side strong {
  max-width: 16rem;
  line-height: 1.5;
}

.blocked-side span {
  color: var(--muted);
}

@keyframes shimmer {
  to {
    background-position-x: -200%;
  }
}

@media (max-width: 1120px) {
  .hero,
  .notes-grid,
  .pricing-grid {
    grid-template-columns: 1fr;
  }

  .panel-header {
    display: grid;
  }
}

@media (max-width: 760px) {
  .pricing-shell {
    padding-left: 16px;
    padding-right: 16px;
  }

  .hero-copy,
  .hero-panel,
  .note-card,
  .panel {
    padding: 22px;
  }

  .hero-actions,
  .panel-actions,
  .blocked-card,
  .card-top,
  .card-footer,
  .tier-row {
    display: grid;
  }

  .blocked-side {
    align-items: flex-start;
    text-align: left;
  }

  .tier-row b {
    text-align: left;
  }
}
</style>
