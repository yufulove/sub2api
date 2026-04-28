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
  { key: 'image_price_1k', label: '1K', detail: '1024x1024 default square' },
  { key: 'image_price_2k', label: '2K', detail: '1536x1024, 1024x1536, 2048x2048' },
  { key: 'image_price_4k', label: '4K', detail: '4096x4096 large format' }
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
  authStore.isAuthenticated ? 'Open generate workspace' : 'Sign in to generate'
)

const surfaceStatusLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return 'Sign in to see your account-specific group pricing'
  }
  if (loading.value) {
    return 'Loading available groups'
  }
  return `${readyGroups.value.length} ready route${readyGroups.value.length === 1 ? '' : 's'}`
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
    errorMessage.value = 'Failed to load your available groups.'
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
  return group.status === 'active' && group.platform !== 'openai' && hasAnyImagePrice(group)
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
    return 'Not configured'
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
      return 'Unknown'
  }
}

function accessLabel(group: Group): string {
  if (group.subscription_type === 'subscription') {
    return 'Subscription'
  }
  if (group.is_exclusive) {
    return 'Allowlisted'
  }
  return 'Public bindable'
}

function routeStatus(group: Group): string {
  if (group.status !== 'active') {
    return 'Inactive group'
  }
  if (group.platform === 'openai') {
    return 'Waiting for OpenAI upstream image service'
  }
  if (!hasAnyImagePrice(group)) {
    return 'No image pricing configured'
  }
  return 'Ready on /v1/images/generations'
}
</script>

<template>
  <main class="pricing-shell">
    <StudioShell max-width="1160px" />

    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">Image Pricing</p>
        <h1>Know what each image route costs before you generate</h1>
        <p class="lead">
          Compare ready image routes, check 1K, 2K, and 4K pricing, and see which groups are blocked
          before you spend wallet balance on a generation run.
        </p>

        <div class="hero-actions">
          <button class="primary-button" type="button" @click="handlePrimaryAction">
            {{ primaryActionLabel }}
          </button>
          <a class="ghost-link" :href="mainSiteKeysURL">Manage API keys</a>
          <RouterLink class="ghost-link" to="/studio/history">Open history</RouterLink>
        </div>
      </div>

      <aside class="hero-panel">
        <span class="panel-tag">Dedicated route</span>
        <dl class="panel-stats">
          <div>
            <dt>Visible pricing groups</dt>
            <dd>{{ authStore.isAuthenticated ? totalPricedGroups : '--' }}</dd>
          </div>
          <div>
            <dt>Ready now</dt>
            <dd>{{ authStore.isAuthenticated ? readyGroups.length : '--' }}</dd>
          </div>
          <div>
            <dt>Lowest 1K tier</dt>
            <dd>{{ authStore.isAuthenticated ? formatPrice(lowestEntryPrice) : 'Sign in' }}</dd>
          </div>
          <div>
            <dt>Billing surface</dt>
            <dd>{{ surfaceStatusLabel }}</dd>
          </div>
        </dl>
      </aside>
    </section>

    <section class="notes-grid">
      <article class="note-card">
        <h2>Billing model</h2>
        <p>Image requests are billed by size tier, so cost stays obvious before you hit generate.</p>
      </article>
      <article class="note-card">
        <h2>Wallet model</h2>
        <p>The image site uses the same wallet and account balance you already manage on the main site.</p>
      </article>
      <article class="note-card">
        <h2>Routing model</h2>
        <p>Only image-capable routes should be used here, so pricing and availability are shown explicitly.</p>
      </article>
    </section>

    <section v-if="!authStore.isAuthenticated" class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Sign In</p>
          <h2>Account pricing depends on your available groups</h2>
          <p class="copy">
            This page can only show real prices after login because pricing is tied to the groups your account
            may bind to API keys. Sign in, then the matrix below will switch from notes to your live routes.
          </p>
        </div>
        <button class="primary-button" type="button" @click="handlePrimaryAction">
          Sign in and load pricing
        </button>
      </div>
    </section>

    <section v-else class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Ready Groups</p>
          <h2>Price matrix for image-capable routes</h2>
          <p class="copy">
            These groups are available to your account and ready for image generation right now. Prices below
            are the live group rates surfaced from the backend.
          </p>
        </div>
        <div class="panel-actions">
          <button class="ghost-button" type="button" :disabled="loading" @click="loadAvailableGroups">
            Refresh groups
          </button>
          <a class="ghost-link" :href="mainSiteUsageURL">Usage ledger</a>
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
        <strong>No ready image routes for this account</strong>
        <p>
          Your available groups are either missing image pricing, inactive, or waiting for an upstream image
          implementation. Check the blocked list below or bind a different group from the main site.
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
            <RouterLink class="inline-link" to="/studio/generate">Generate</RouterLink>
          </div>
        </article>
      </div>
    </section>

    <section v-if="authStore.isAuthenticated && blockedGroups.length > 0" class="panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Blocked Groups</p>
          <h2>Visible to your account, but not ready to generate</h2>
          <p class="copy">
            These groups exist for your account, but they cannot handle direct image generation yet. Keeping them
            listed here makes the difference between usable and blocked routes explicit.
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
  --paper: #f6f0e4;
  --panel: rgba(255, 252, 247, 0.8);
  --card: rgba(255, 255, 255, 0.76);
  --ink: #12271f;
  --muted: rgba(18, 39, 31, 0.7);
  --line: rgba(18, 39, 31, 0.12);
  --accent: #0f7a6d;
  --accent-soft: #d8f0e8;
  --accent-strong: #08574f;
  min-height: 100vh;
  padding: 40px 24px 72px;
  background:
    radial-gradient(circle at top left, rgba(15, 122, 109, 0.15), transparent 30rem),
    radial-gradient(circle at right center, rgba(224, 157, 83, 0.18), transparent 26rem),
    linear-gradient(180deg, #fcf8f1 0%, var(--paper) 100%);
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
  gap: 20px;
  align-items: stretch;
}

.hero-copy,
.hero-panel,
.note-card,
.panel {
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel);
  backdrop-filter: blur(14px);
  box-shadow: 0 20px 60px rgba(18, 39, 31, 0.08);
}

.hero-copy,
.hero-panel,
.panel,
.note-card {
  padding: 28px;
}

.eyebrow {
  margin: 0 0 12px;
  font-size: 0.78rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: var(--accent-strong);
}

.hero-copy h1,
.panel-header h2 {
  margin: 0;
  line-height: 0.94;
  letter-spacing: -0.05em;
}

.hero-copy h1 {
  font-size: clamp(2.5rem, 5vw, 4.2rem);
}

.lead,
.copy,
.card-copy,
.blocked-card p,
.tier-row span {
  color: var(--muted);
}

.lead {
  margin: 18px 0 0;
  max-width: 42rem;
  line-height: 1.74;
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
  border-radius: 999px;
  text-decoration: none;
  font: inherit;
  font-weight: 700;
  color: inherit;
  transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.primary-button {
  border: none;
  background: linear-gradient(135deg, var(--accent) 0%, #16a08e 100%);
  color: #f8fffd;
  box-shadow: 0 18px 34px rgba(15, 122, 109, 0.22);
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
  border-radius: 999px;
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
  gap: 16px;
  margin: 16px 0 0;
}

.panel-stats div {
  padding-top: 14px;
  border-top: 1px solid var(--line);
}

.panel-stats dt {
  font-size: 0.8rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
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
  margin-top: 20px;
}

.panel-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
}

.panel-header h2 {
  font-size: clamp(1.9rem, 4vw, 3rem);
}

.copy {
  margin: 14px 0 0;
  max-width: 46rem;
  line-height: 1.72;
}

.alert-box {
  margin-top: 18px;
  padding: 14px 16px;
  border-radius: 18px;
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
  border-radius: 24px;
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
  border-radius: 20px;
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
