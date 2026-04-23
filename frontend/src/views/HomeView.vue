<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- SECURITY: homeContent is an admin-only setting. -->
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="min-h-screen bg-slate-50 text-slate-950 dark:bg-slate-950 dark:text-white"
  >
    <header
      class="sticky top-0 z-40 border-b border-white/70 bg-white/85 backdrop-blur-xl dark:border-white/10 dark:bg-slate-950/85"
    >
      <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <span
            class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-slate-900"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-sm font-semibold text-slate-900 dark:text-white">
            {{ siteName }}
          </span>
        </router-link>

        <div class="hidden items-center gap-6 text-sm font-medium text-slate-600 dark:text-slate-300 md:flex">
          <a href="#models" class="transition hover:text-slate-950 dark:hover:text-white">模型</a>
          <a href="#pricing" class="transition hover:text-slate-950 dark:hover:text-white">价格</a>
          <a href="#quickstart" class="transition hover:text-slate-950 dark:hover:text-white">接入</a>
          <a href="#faq" class="transition hover:text-slate-950 dark:hover:text-white">FAQ</a>
        </div>

        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="hidden rounded-lg p-2 text-slate-500 transition hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-white sm:inline-flex"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-slate-500 transition hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-800 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 dark:border-white/10 dark:bg-slate-900 dark:text-white dark:hover:bg-slate-800"
          >
            <Icon :name="isAuthenticated ? 'grid' : 'login'" size="sm" />
            <span class="hidden sm:inline">
              {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
            </span>
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="relative isolate overflow-hidden bg-slate-950">
        <div class="absolute inset-0 hero-grid opacity-80"></div>
        <div class="absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-slate-950 to-transparent"></div>

        <div
          aria-hidden="true"
          class="pointer-events-none absolute inset-y-0 right-0 hidden w-[58%] min-w-[720px] overflow-hidden lg:block"
        >
          <div class="gateway-scene">
            <div class="gateway-topbar">
              <span></span>
              <span></span>
              <span></span>
              <strong>{{ openaiBaseUrl }}</strong>
            </div>
            <div class="gateway-body">
              <div class="gateway-sidebar">
                <span class="active"></span>
                <span></span>
                <span></span>
                <span></span>
              </div>
              <div class="gateway-main">
                <div class="gateway-metrics">
                  <div>
                    <p>Requests</p>
                    <strong>24.8K</strong>
                  </div>
                  <div>
                    <p>Latency</p>
                    <strong>1.2s</strong>
                  </div>
                  <div>
                    <p>Models</p>
                    <strong>40+</strong>
                  </div>
                </div>
                <div class="gateway-flow">
                  <span>API Key</span>
                  <i></i>
                  <span>Router</span>
                  <i></i>
                  <span>Claude</span>
                  <span>GPT</span>
                  <span>Gemini</span>
                </div>
                <div class="gateway-log">
                  <p><b>POST</b> /v1/chat/completions <em>200 OK</em></p>
                  <p><b>MODEL</b> gpt-4o-mini <em>$0.0021</em></p>
                  <p><b>FAILOVER</b> upstream-b -> upstream-c <em>89ms</em></p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="relative mx-auto max-w-7xl px-4 pb-10 pt-14 sm:px-6 sm:pb-12 sm:pt-20 lg:px-8">
          <div class="max-w-3xl">
            <div
              class="mb-6 inline-flex items-center gap-2 rounded-lg border border-cyan-300/20 bg-cyan-300/10 px-3 py-2 text-sm font-medium text-cyan-100"
            >
              <Icon name="server" size="sm" />
              <span>{{ siteSubtitle || '统一接入主流 AI 模型的 API 网关与 Token 中转平台' }}</span>
            </div>

            <h1
              class="max-w-3xl text-4xl font-bold leading-tight text-white sm:text-5xl lg:text-6xl"
            >
              一个 Key，接入多家 AI 模型 API
            </h1>
            <p class="mt-6 max-w-2xl text-base leading-8 text-slate-300 sm:text-lg">
              统一 OpenAI 兼容接口，支持多模型中转、余额计费、快速调用和日志追踪。充值后即可创建 Key，把模型切换和用量控制交给网关。
            </p>

            <div class="mt-8 flex flex-col gap-3 sm:flex-row">
              <router-link
                :to="entryRoute"
                class="inline-flex items-center justify-center gap-2 rounded-lg bg-cyan-400 px-5 py-3 text-sm font-bold text-slate-950 shadow-lg shadow-cyan-500/20 transition hover:bg-cyan-300"
              >
                <Icon name="key" size="md" />
                {{ entryLabel }}
              </router-link>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center justify-center gap-2 rounded-lg border border-white/15 bg-white/10 px-5 py-3 text-sm font-semibold text-white transition hover:bg-white/15"
              >
                <Icon name="book" size="md" />
                API 文档
              </a>
              <a
                v-else
                href="#quickstart"
                class="inline-flex items-center justify-center gap-2 rounded-lg border border-white/15 bg-white/10 px-5 py-3 text-sm font-semibold text-white transition hover:bg-white/15"
              >
                <Icon name="terminal" size="md" />
                查看接入示例
              </a>
              <router-link
                to="/key-usage"
                class="inline-flex items-center justify-center gap-2 rounded-lg px-5 py-3 text-sm font-semibold text-slate-300 transition hover:bg-white/10 hover:text-white"
              >
                <Icon name="chart" size="md" />
                在线查用量
              </router-link>
            </div>

            <p class="mt-4 text-sm text-slate-400">
              兼容 OpenAI SDK / 支持多模型 / 余额与调用记录实时可查
            </p>
          </div>
        </div>
      </section>

      <section class="border-b border-slate-200 bg-white dark:border-white/10 dark:bg-slate-950">
        <div class="mx-auto grid max-w-7xl gap-3 px-4 py-5 sm:grid-cols-2 sm:px-6 lg:grid-cols-6 lg:px-8">
          <div
            v-for="item in trustItems"
            :key="item"
            class="flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-medium text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-200"
          >
            <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
            {{ item }}
          </div>
        </div>
      </section>

      <section id="models" class="bg-slate-50 py-16 dark:bg-slate-950">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="flex flex-col justify-between gap-6 lg:flex-row lg:items-end">
            <div class="max-w-2xl">
              <p class="text-sm font-semibold uppercase tracking-[0.2em] text-cyan-700 dark:text-cyan-300">
                Model Routing
              </p>
              <h2 class="mt-3 text-3xl font-bold text-slate-950 dark:text-white">
                模型不是堆 Logo，要让用户知道能怎么用
              </h2>
              <p class="mt-4 text-base leading-7 text-slate-600 dark:text-slate-300">
                首页直接展示厂商、场景、接口兼容性和状态，减少开发者试错成本。
              </p>
            </div>

            <div class="flex flex-wrap gap-2">
              <button
                v-for="filter in modelFilters"
                :key="filter.id"
                type="button"
                @click="activeModelFilter = filter.id"
                :class="[
                  'inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-semibold transition',
                  activeModelFilter === filter.id
                    ? 'border-cyan-500 bg-cyan-500 text-white shadow-sm shadow-cyan-500/20'
                    : 'border-slate-200 bg-white text-slate-700 hover:border-slate-300 dark:border-white/10 dark:bg-white/5 dark:text-slate-300 dark:hover:bg-white/10'
                ]"
              >
                <Icon :name="filter.icon" size="sm" />
                {{ filter.label }}
              </button>
            </div>
          </div>

          <div class="mt-8 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-white/5">
            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-slate-200 text-sm dark:divide-white/10">
                <thead class="bg-slate-100 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:bg-white/5 dark:text-slate-400">
                  <tr>
                    <th class="px-4 py-3">模型</th>
                    <th class="px-4 py-3">厂商</th>
                    <th class="px-4 py-3">适用场景</th>
                    <th class="px-4 py-3">接口</th>
                    <th class="px-4 py-3">状态</th>
                    <th class="px-4 py-3">计费提示</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100 dark:divide-white/10">
                  <tr
                    v-for="model in filteredModels"
                    :key="model.name"
                    class="transition hover:bg-slate-50 dark:hover:bg-white/[0.03]"
                  >
                    <td class="whitespace-nowrap px-4 py-4 font-semibold text-slate-950 dark:text-white">
                      {{ model.name }}
                    </td>
                    <td class="whitespace-nowrap px-4 py-4 text-slate-600 dark:text-slate-300">
                      {{ model.vendor }}
                    </td>
                    <td class="px-4 py-4">
                      <div class="flex flex-wrap gap-2">
                        <span
                          v-for="tag in model.tags"
                          :key="tag"
                          class="rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700 dark:bg-white/10 dark:text-slate-200"
                        >
                          {{ tag }}
                        </span>
                      </div>
                    </td>
                    <td class="whitespace-nowrap px-4 py-4 text-slate-600 dark:text-slate-300">
                      {{ model.protocol }}
                    </td>
                    <td class="whitespace-nowrap px-4 py-4">
                      <span
                        class="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/20"
                      >
                        <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                        {{ model.status }}
                      </span>
                    </td>
                    <td class="whitespace-nowrap px-4 py-4 text-slate-600 dark:text-slate-300">
                      {{ model.price }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>

      <section id="pricing" class="bg-slate-50 py-16 dark:bg-slate-950">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="max-w-2xl">
            <p class="text-sm font-semibold uppercase tracking-[0.2em] text-amber-700 dark:text-amber-300">
              Transparent Billing
            </p>
            <h2 class="mt-3 text-3xl font-bold text-slate-950 dark:text-white">
              价格区要能看懂，也要能下手
            </h2>
            <p class="mt-4 text-base leading-7 text-slate-600 dark:text-slate-300">
              首页先讲清楚计费方式和成本构成，精确单价交给控制台实时价格表，避免过期价格误导用户。
            </p>
          </div>

          <div class="mt-8 grid gap-4 md:grid-cols-3">
            <div
              v-for="plan in pricingBlocks"
              :key="plan.title"
              class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-white/10 dark:bg-white/5"
            >
              <div class="flex items-center justify-between gap-4">
                <h3 class="text-lg font-bold text-slate-950 dark:text-white">{{ plan.title }}</h3>
                <Icon :name="plan.icon" size="lg" :class="plan.iconClass" />
              </div>
              <p class="mt-4 text-2xl font-bold text-slate-950 dark:text-white">{{ plan.metric }}</p>
              <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ plan.description }}</p>
              <p class="mt-4 rounded-lg bg-slate-100 px-3 py-2 text-sm font-medium text-slate-700 dark:bg-white/10 dark:text-slate-200">
                {{ plan.example }}
              </p>
            </div>
          </div>

          <div class="mt-6 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm leading-6 text-emerald-900 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200">
            按实际 token 或任务用量扣费，余额实时可查，无隐藏月费。模型费率、倍率、充值规则以控制台展示为准，变动应通过公告提前说明。
          </div>
        </div>
      </section>

      <section id="quickstart" class="bg-white py-16 dark:bg-slate-900">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-start">
            <div>
              <p class="text-sm font-semibold uppercase tracking-[0.2em] text-cyan-700 dark:text-cyan-300">
                3-Minute Setup
              </p>
              <h2 class="mt-3 text-3xl font-bold text-slate-950 dark:text-white">
                注册、充值、拿 Key、开始调用
              </h2>
              <div class="mt-8 grid gap-3">
                <div
                  v-for="(step, index) in setupSteps"
                  :key="step.title"
                  class="flex gap-4 rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-white/10 dark:bg-white/5"
                >
                  <div
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-950 text-sm font-bold text-white dark:bg-cyan-400 dark:text-slate-950"
                  >
                    {{ index + 1 }}
                  </div>
                  <div>
                    <h3 class="font-bold text-slate-950 dark:text-white">{{ step.title }}</h3>
                    <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">
                      {{ step.description }}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div class="overflow-hidden rounded-lg border border-slate-800 bg-slate-950 shadow-xl">
              <div class="flex items-center justify-between border-b border-white/10 bg-slate-900 px-4 py-3">
                <div class="flex gap-2">
                  <span class="h-3 w-3 rounded-full bg-red-400"></span>
                  <span class="h-3 w-3 rounded-full bg-amber-400"></span>
                  <span class="h-3 w-3 rounded-full bg-emerald-400"></span>
                </div>
                <div class="flex gap-2">
                  <button
                    v-for="tab in codeTabs"
                    :key="tab.id"
                    type="button"
                    @click="activeCodeTab = tab.id"
                    :class="[
                      'rounded-md px-3 py-1.5 text-xs font-semibold transition',
                      activeCodeTab === tab.id
                        ? 'bg-cyan-400 text-slate-950'
                        : 'bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white'
                    ]"
                  >
                    {{ tab.label }}
                  </button>
                </div>
              </div>
              <pre class="overflow-x-auto p-5 text-sm leading-7 text-slate-100"><code>{{ activeCodeSnippet }}</code></pre>
            </div>
          </div>
        </div>
      </section>

      <section id="faq" class="bg-white py-16 dark:bg-slate-900">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="max-w-2xl">
            <p class="text-sm font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
              FAQ
            </p>
            <h2 class="mt-3 text-3xl font-bold text-slate-950 dark:text-white">
              敏感问题直接回答
            </h2>
          </div>

          <div class="mt-8 grid gap-4 md:grid-cols-2">
            <details
              v-for="item in faqItems"
              :key="item.question"
              class="group rounded-lg border border-slate-200 bg-slate-50 p-5 dark:border-white/10 dark:bg-white/5"
            >
              <summary class="flex cursor-pointer list-none items-center justify-between gap-4 font-bold text-slate-950 dark:text-white">
                {{ item.question }}
                <Icon name="chevronDown" size="sm" class="shrink-0 transition group-open:rotate-180" />
              </summary>
              <p class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">
                {{ item.answer }}
              </p>
            </details>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-slate-200 bg-slate-950 text-slate-300 dark:border-white/10">
      <div class="mx-auto grid max-w-7xl gap-8 px-4 py-10 sm:px-6 md:grid-cols-4 lg:px-8">
        <div class="md:col-span-2">
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg bg-white">
              <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
            </span>
            <span class="font-bold text-white">{{ siteName }}</span>
          </div>
          <p class="mt-4 max-w-lg text-sm leading-6 text-slate-400">
            统一接入主流 AI 模型的 API 网关与 Token 中转平台。让开发者更快拿到 Key，更清楚地看到余额、日志和模型状态。
          </p>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">入口</h3>
          <div class="mt-4 grid gap-3 text-sm">
            <router-link to="/login" class="hover:text-white">登录</router-link>
            <router-link to="/key-usage" class="hover:text-white">用量查询</router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-white">
              API 文档
            </a>
          </div>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white">联系与公告</h3>
          <div class="mt-4 grid gap-3 text-sm">
            <p class="leading-6">{{ contactInfo || '客服方式登录后查看' }}</p>
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="hover:text-white">
              控制台公告
            </router-link>
            <router-link :to="isAuthenticated ? '/usage' : '/login'" class="hover:text-white">
              调用日志
            </router-link>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

type ModelFilter = 'all' | 'text' | 'code' | 'image' | 'value'
type CodeTab = 'python' | 'javascript'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'FionaAI')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')
const apiBaseUrl = computed(() => appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')
const registrationEnabled = computed(() => appStore.cachedPublicSettings?.registration_enabled ?? true)

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const activeModelFilter = ref<ModelFilter>('all')
const activeCodeTab = ref<CodeTab>('python')


const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const entryRoute = computed(() => {
  if (isAuthenticated.value) return '/keys'
  return registrationEnabled.value ? '/register' : '/login'
})
const entryLabel = computed(() => {
  if (isAuthenticated.value) return '创建 / 管理 API Key'
  return registrationEnabled.value ? '立即获取 Key' : '登录获取 Key'
})


const normalizedApiRoot = computed(() => {
  const fallback = typeof window !== 'undefined' ? window.location.origin : 'https://fionalovs.com'
  return (apiBaseUrl.value || fallback).replace(/\/+$/, '')
})

const openaiBaseUrl = computed(() =>
  normalizedApiRoot.value.endsWith('/v1') ? normalizedApiRoot.value : `${normalizedApiRoot.value}/v1`
)

const trustItems = [
  'OpenAI 兼容协议',
  '多模型切换',
  '实时余额扣费',
  '请求日志可查',
  '状态与公告入口',
  '文档接入 3 分钟'
]

const modelFilters: Array<{ id: ModelFilter; label: string; icon: 'grid' | 'chat' | 'terminal' | 'sparkles' | 'dollar' }> = [
  { id: 'all', label: '全部', icon: 'grid' },
  { id: 'text', label: '文本', icon: 'chat' },
  { id: 'code', label: '代码', icon: 'terminal' },
  { id: 'image', label: '图像', icon: 'sparkles' },
  { id: 'value', label: '高性价比', icon: 'dollar' }
]

const modelRows = [
  {
    name: 'GPT 系列',
    vendor: 'OpenAI',
    tags: ['聊天', '代码', '推理', '工具调用'],
    protocol: 'OpenAI 兼容',
    status: '可用',
    price: '按 input / output token',
    categories: ['text', 'code', 'value']
  },
  {
    name: 'Claude 系列',
    vendor: 'Anthropic',
    tags: ['长文', '代码', '推理', 'Claude Code'],
    protocol: '网关适配',
    status: '可用',
    price: '按模型与分组倍率',
    categories: ['text', 'code']
  },
  {
    name: 'Gemini 系列',
    vendor: 'Google',
    tags: ['多模态', '长上下文', '高性价比'],
    protocol: 'OpenAI / Gemini 兼容',
    status: '可用',
    price: '按 token 或任务',
    categories: ['text', 'image', 'value']
  },
  {
    name: 'Antigravity',
    vendor: 'Google / Anthropic',
    tags: ['代码代理', '图像', '实验能力'],
    protocol: '专用路由',
    status: '可用',
    price: '按分组规则',
    categories: ['code', 'image']
  }
]

const filteredModels = computed(() => {
  if (activeModelFilter.value === 'all') return modelRows
  return modelRows.filter((model) => model.categories.includes(activeModelFilter.value))
})

const pricingBlocks: Array<{
  title: string
  metric: string
  description: string
  example: string
  icon: 'chat' | 'sparkles' | 'database'
  iconClass: string
}> = [
  {
    title: '文本 / 代码',
    metric: '按 1M tokens 计',
    description: 'input、output、cache 分开计算，适合聊天、代码生成、代理任务。',
    example: '成本 = 输入 × 单价 + 输出 × 单价 + cache × 单价',
    icon: 'chat',
    iconClass: 'text-cyan-600 dark:text-cyan-300'
  },
  {
    title: '图像 / 多模态',
    metric: '按次或按任务计',
    description: '图像输入、图像生成和多模态模型按对应上游规则与分组倍率扣费。',
    example: '适合海报生成、识图、视觉问答和素材处理',
    icon: 'sparkles',
    iconClass: 'text-rose-600 dark:text-rose-300'
  },
  {
    title: 'Embedding / 批量',
    metric: '按 token 批量扣',
    description: '向量化、检索增强和批处理任务统一进入用量日志，方便对账。',
    example: '适合知识库、搜索召回和批量文本处理',
    icon: 'database',
    iconClass: 'text-emerald-600 dark:text-emerald-300'
  }
]

const setupSteps = [
  {
    title: '注册账号',
    description: '登录后进入控制台，查看可用模型、公告和账户状态。'
  },
  {
    title: '充值余额',
    description: '余额实时可查，按实际用量扣费，适合先小额试用。'
  },
  {
    title: '创建 API Key',
    description: '选择模型分组，设置额度、过期时间和访问限制。'
  },
  {
    title: '复制代码开始调用',
    description: '使用 OpenAI SDK 的 base_url 和 api_key 即可跑通第一条请求。'
  }
]

const codeTabs: Array<{ id: CodeTab; label: string }> = [
  { id: 'python', label: 'Python' },
  { id: 'javascript', label: 'JavaScript' }
]

const activeCodeSnippet = computed(() => {
  if (activeCodeTab.value === 'javascript') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "sk-...",
  baseURL: "${openaiBaseUrl.value}"
});

const response = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "测试一下网关是否可用" }]
});

console.log(response.choices[0].message.content);`
  }

  return `from openai import OpenAI

client = OpenAI(
    api_key="sk-...",
    base_url="${openaiBaseUrl.value}",
)

response = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "测试一下网关是否可用"}],
)

print(response.choices[0].message.content)`
})

const faqItems = [
  {
    question: '是否兼容 OpenAI SDK？',
    answer: '支持 OpenAI 兼容接口。把 SDK 的 base_url 指向本站 API 地址，再使用平台生成的 API Key 即可调用。'
  },
  {
    question: '支持哪些模型？',
    answer: '当前首页展示 GPT、Claude、Gemini、Antigravity 等模型族。实际可用模型以控制台分组和后台配置为准。'
  },
  {
    question: '余额如何扣费？',
    answer: '按模型、token、任务类型和分组倍率计算费用。余额、用量和请求记录可在控制台查看。'
  },
  {
    question: '接口失败是否扣费？',
    answer: '以服务端实际记录的用量日志为准。建议用户通过用量页核对请求状态、token 和成本。'
  },
  {
    question: 'API Key 和速率限制怎么管？',
    answer: '可以创建新 Key、停用旧 Key，并配置额度、过期时间、分组权限和时间窗口限制。'
  },
  {
    question: '充值或调用异常怎么办？',
    answer: '保留充值记录或请求日志，在页脚/控制台查看客服和公告信息；余额、用量和公告入口会在首页明确展示。'
  }
]

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()

  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.hero-grid {
  background:
    linear-gradient(rgba(15, 23, 42, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 23, 42, 0.12) 1px, transparent 1px),
    linear-gradient(135deg, #020617 0%, #0f172a 42%, #082f49 100%);
  background-size: 56px 56px, 56px 56px, auto;
}

.gateway-scene {
  position: absolute;
  right: -8%;
  top: 50%;
  width: 760px;
  transform: translateY(-50%) rotateX(58deg) rotateZ(-35deg);
  transform-style: preserve-3d;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.9);
  box-shadow: 0 36px 90px rgba(8, 47, 73, 0.45);
}

.gateway-topbar {
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
  padding: 14px 16px;
}

.gateway-topbar span {
  height: 10px;
  width: 10px;
  border-radius: 999px;
  background: #38bdf8;
}

.gateway-topbar span:nth-child(2) {
  background: #f59e0b;
}

.gateway-topbar span:nth-child(3) {
  background: #10b981;
}

.gateway-topbar strong {
  margin-left: 12px;
  color: #94a3b8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  font-weight: 600;
}

.gateway-body {
  display: grid;
  grid-template-columns: 84px 1fr;
  min-height: 430px;
}

.gateway-sidebar {
  display: grid;
  align-content: start;
  gap: 14px;
  border-right: 1px solid rgba(148, 163, 184, 0.16);
  padding: 18px;
}

.gateway-sidebar span {
  height: 34px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.12);
}

.gateway-sidebar span.active {
  background: rgba(34, 211, 238, 0.35);
}

.gateway-main {
  display: grid;
  gap: 18px;
  padding: 20px;
}

.gateway-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}

.gateway-metrics div,
.gateway-flow,
.gateway-log {
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 8px;
  background: rgba(2, 6, 23, 0.54);
}

.gateway-metrics div {
  padding: 16px;
}

.gateway-metrics p,
.gateway-log p {
  margin: 0;
  color: #64748b;
  font-size: 12px;
}

.gateway-metrics strong {
  display: block;
  margin-top: 8px;
  color: #e2e8f0;
  font-size: 26px;
}

.gateway-flow {
  display: grid;
  grid-template-columns: 1fr 32px 1fr 32px repeat(3, 1fr);
  align-items: center;
  gap: 10px;
  padding: 20px;
}

.gateway-flow span {
  border-radius: 8px;
  background: rgba(14, 165, 233, 0.14);
  color: #bae6fd;
  font-size: 12px;
  font-weight: 700;
  padding: 12px 10px;
  text-align: center;
}

.gateway-flow span:nth-last-child(-n + 3) {
  background: rgba(16, 185, 129, 0.14);
  color: #bbf7d0;
}

.gateway-flow i {
  height: 2px;
  background: #38bdf8;
}

.gateway-log {
  display: grid;
  gap: 12px;
  padding: 18px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.gateway-log b {
  color: #67e8f9;
}

.gateway-log em {
  color: #34d399;
  font-style: normal;
}

@media (prefers-reduced-motion: no-preference) {
  .gateway-scene {
    animation: gateway-drift 9s ease-in-out infinite;
  }
}

@keyframes gateway-drift {
  0%,
  100% {
    transform: translateY(-50%) rotateX(58deg) rotateZ(-35deg) translate3d(0, 0, 0);
  }
  50% {
    transform: translateY(-52%) rotateX(58deg) rotateZ(-35deg) translate3d(-10px, 8px, 0);
  }
}
</style>
