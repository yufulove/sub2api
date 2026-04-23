<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-slate-50 text-slate-950 dark:bg-slate-950 dark:text-white"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(15,23,42,0.06)_1px,transparent_1px),linear-gradient(90deg,rgba(15,23,42,0.06)_1px,transparent_1px)] bg-[size:44px_44px] dark:bg-[linear-gradient(rgba(148,163,184,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.08)_1px,transparent_1px)]"
      ></div>
      <div class="absolute inset-x-0 top-0 h-32 bg-gradient-to-b from-white via-white/80 to-transparent dark:from-slate-950 dark:via-slate-950/80"></div>
      <div class="absolute right-0 top-0 h-full w-1/2 bg-[linear-gradient(135deg,rgba(20,184,166,0.14),rgba(59,130,246,0.08)_45%,transparent_70%)]"></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 overflow-hidden rounded-lg shadow-md shadow-slate-900/10 dark:shadow-black/40">
            <img :src="siteLogo || '/brand-mark.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="hidden text-sm font-semibold text-slate-900 dark:text-white sm:inline">{{ siteName }}</span>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-gray-900 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-primary-400 to-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium text-white">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3 w-3 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-6 py-16">
      <div class="mx-auto max-w-6xl">
        <!-- Hero Section -->
        <section class="brand-hero mb-10">
          <div class="hero-visual" aria-hidden="true">
            <div class="route-node route-source">
              <span>API Key</span>
              <strong>sk-live</strong>
            </div>
            <div class="route-node route-gateway">
              <span>Gateway</span>
              <strong>Router</strong>
            </div>
            <div class="route-node route-model route-claude">Claude</div>
            <div class="route-node route-model route-gpt">GPT</div>
            <div class="route-node route-model route-gemini">Gemini</div>
            <div class="route-line route-line-a"></div>
            <div class="route-line route-line-b"></div>
            <div class="route-line route-line-c"></div>

            <div class="gateway-console">
              <div class="console-header">
                <span class="console-dot bg-sky-400"></span>
                <span class="console-dot bg-emerald-400"></span>
                <span class="console-dot bg-amber-400"></span>
                <span class="ml-auto text-[11px] uppercase text-slate-400">live route</span>
              </div>
              <div class="console-row">
                <span class="text-emerald-300">POST</span>
                <span>/v1/chat/completions</span>
                <strong>200</strong>
              </div>
              <div class="console-row">
                <span class="text-sky-300">MODEL</span>
                <span>gpt-5.4 -> fallback ready</span>
                <strong>34ms</strong>
              </div>
              <div class="console-row">
                <span class="text-violet-300">COST</span>
                <span>quota, limits, team usage</span>
                <strong>$0.002</strong>
              </div>
            </div>
          </div>

          <div class="hero-copy">
            <div class="mb-5 inline-flex items-center gap-2 rounded-full border border-teal-200 bg-white/80 px-3 py-1.5 text-sm font-medium text-teal-700 shadow-sm backdrop-blur dark:border-teal-500/30 dark:bg-slate-900/70 dark:text-teal-200">
              <Icon name="terminal" size="sm" />
              {{ t('home.heroEyebrow') }}
            </div>
            <h1 class="text-5xl font-bold text-slate-950 dark:text-white md:text-6xl lg:text-7xl">
              {{ siteName }}
            </h1>
            <p class="mt-5 max-w-2xl text-xl leading-8 text-slate-700 dark:text-slate-200 md:text-2xl">
              {{ siteSubtitle }}
            </p>
            <p class="mt-4 max-w-2xl text-base leading-7 text-slate-500 dark:text-slate-400">
              {{ t('home.heroDescription') }}
            </p>

            <div class="mt-8 flex flex-col gap-3 sm:flex-row">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="inline-flex items-center justify-center rounded-lg bg-slate-950 px-6 py-3 text-sm font-semibold text-white shadow-xl shadow-slate-900/15 transition hover:-translate-y-0.5 hover:bg-slate-800 dark:bg-white dark:text-slate-950 dark:hover:bg-slate-200"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="sm" class="ml-2" :stroke-width="2" />
              </router-link>
              <router-link
                to="/key-usage"
                class="inline-flex items-center justify-center rounded-lg border border-slate-200 bg-white/75 px-6 py-3 text-sm font-semibold text-slate-700 shadow-sm backdrop-blur transition hover:-translate-y-0.5 hover:border-teal-300 hover:text-teal-700 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-200 dark:hover:border-teal-500"
              >
                {{ t('home.reviewUsage') }}
              </router-link>
            </div>
          </div>
        </section>

        <!-- Feature Tags - Centered -->
        <div class="mb-12 flex flex-wrap items-center justify-center gap-4 md:gap-6">
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="swap" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.subscriptionToApi')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="shield" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.stickySession')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="chart" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.realtimeBilling')
            }}</span>
          </div>
        </div>

        <!-- Features Grid -->
        <div class="mb-12 grid gap-6 md:grid-cols-3">
          <!-- Feature 1: Unified Gateway -->
          <div
            class="group rounded-2xl border border-gray-200/50 bg-white/60 p-6 backdrop-blur-sm transition-all duration-300 hover:shadow-xl hover:shadow-primary-500/10 dark:border-dark-700/50 dark:bg-dark-800/60"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 shadow-lg shadow-blue-500/30 transition-transform group-hover:scale-110"
            >
              <Icon name="server" size="lg" class="text-white" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.unifiedGateway') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
          </div>

          <!-- Feature 2: Account Pool -->
          <div
            class="group rounded-2xl border border-gray-200/50 bg-white/60 p-6 backdrop-blur-sm transition-all duration-300 hover:shadow-xl hover:shadow-primary-500/10 dark:border-dark-700/50 dark:bg-dark-800/60"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 shadow-lg shadow-primary-500/30 transition-transform group-hover:scale-110"
            >
              <svg
                class="h-6 w-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.multiAccount') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.multiAccountDesc') }}
            </p>
          </div>

          <!-- Feature 3: Billing & Quota -->
          <div
            class="group rounded-2xl border border-gray-200/50 bg-white/60 p-6 backdrop-blur-sm transition-all duration-300 hover:shadow-xl hover:shadow-primary-500/10 dark:border-dark-700/50 dark:bg-dark-800/60"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-purple-500 to-purple-600 shadow-lg shadow-purple-500/30 transition-transform group-hover:scale-110"
            >
              <svg
                class="h-6 w-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.balanceQuota') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.balanceQuotaDesc') }}
            </p>
          </div>
        </div>

        <!-- Supported Providers -->
        <div class="mb-8 text-center">
          <h2 class="mb-3 text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('home.providers.title') }}
          </h2>
          <p class="text-sm text-gray-600 dark:text-dark-400">
            {{ t('home.providers.description') }}
          </p>
        </div>

        <div class="mb-16 flex flex-wrap items-center justify-center gap-4">
          <!-- Claude - Supported -->
          <div
            class="flex items-center gap-2 rounded-xl border border-primary-200 bg-white/60 px-5 py-3 ring-1 ring-primary-500/20 backdrop-blur-sm dark:border-primary-800 dark:bg-dark-800/60"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-orange-400 to-orange-500"
            >
              <span class="text-xs font-bold text-white">C</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.claude') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- GPT - Supported -->
          <div
            class="flex items-center gap-2 rounded-xl border border-primary-200 bg-white/60 px-5 py-3 ring-1 ring-primary-500/20 backdrop-blur-sm dark:border-primary-800 dark:bg-dark-800/60"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-green-500 to-green-600"
            >
              <span class="text-xs font-bold text-white">G</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">GPT</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Gemini - Supported -->
          <div
            class="flex items-center gap-2 rounded-xl border border-primary-200 bg-white/60 px-5 py-3 ring-1 ring-primary-500/20 backdrop-blur-sm dark:border-primary-800 dark:bg-dark-800/60"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-blue-600"
            >
              <span class="text-xs font-bold text-white">G</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.gemini') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Antigravity - Supported -->
          <div
            class="flex items-center gap-2 rounded-xl border border-primary-200 bg-white/60 px-5 py-3 ring-1 ring-primary-500/20 backdrop-blur-sm dark:border-primary-800 dark:bg-dark-800/60"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-pink-600"
            >
              <span class="text-xs font-bold text-white">A</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.antigravity') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- More - Coming Soon -->
          <div
            class="flex items-center gap-2 rounded-xl border border-gray-200/50 bg-white/40 px-5 py-3 opacity-60 backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/40"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-gray-500 to-gray-600"
            >
              <span class="text-xs font-bold text-white">+</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.more') }}</span>
            <span
              class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-400"
              >{{ t('home.providers.soon') }}</span
            >
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const genericSiteNames = new Set(['Sub2API'])
const genericSubtitles = new Set([
  'Subscription to API Conversion Platform',
  'AI API Gateway Platform'
])

const siteName = computed(() => {
  const configuredName = (appStore.cachedPublicSettings?.site_name || appStore.siteName || '').trim()
  return configuredName && !genericSiteNames.has(configuredName) ? configuredName : 'API中转'
})
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => {
  const configuredSubtitle = (appStore.cachedPublicSettings?.site_subtitle || '').trim()
  return configuredSubtitle && !genericSubtitles.has(configuredSubtitle)
    ? configuredSubtitle
    : t('home.heroSubtitle')
})
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
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

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.brand-hero {
  position: relative;
  min-height: 460px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 12px;
  background:
    linear-gradient(120deg, rgba(255, 255, 255, 0.94), rgba(240, 253, 250, 0.74) 52%, rgba(239, 246, 255, 0.72)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(255, 255, 255, 0.72));
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.08);
}

.hero-copy {
  position: relative;
  z-index: 2;
  max-width: 42rem;
  padding: 4.5rem 3rem;
}

.hero-visual {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, transparent 0, transparent 39%, rgba(15, 23, 42, 0.05) 39%, rgba(15, 23, 42, 0.05) 40%, transparent 40%),
    repeating-linear-gradient(0deg, transparent 0 47px, rgba(15, 23, 42, 0.05) 48px);
}

.gateway-console {
  position: absolute;
  top: 4rem;
  right: 3rem;
  width: min(30rem, 44vw);
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.96);
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.28);
}

.console-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.14);
  padding: 0.8rem 1rem;
}

.console-dot {
  width: 0.65rem;
  height: 0.65rem;
  border-radius: 999px;
}

.console-row {
  display: grid;
  grid-template-columns: 4.5rem minmax(0, 1fr) auto;
  gap: 1rem;
  padding: 0.9rem 1rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.78rem;
  color: #cbd5e1;
}

.console-row + .console-row {
  border-top: 1px solid rgba(148, 163, 184, 0.1);
}

.console-row strong {
  color: #f8fafc;
}

.route-node {
  position: absolute;
  z-index: 1;
  display: inline-flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 7.5rem;
  border: 1px solid rgba(20, 184, 166, 0.2);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.76);
  padding: 0.8rem 1rem;
  box-shadow: 0 14px 45px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(14px);
  color: #0f172a;
}

.route-node span {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  color: #64748b;
}

.route-node strong {
  font-size: 1rem;
}

.route-source {
  right: 31rem;
  bottom: 5rem;
}

.route-gateway {
  right: 18rem;
  bottom: 7.2rem;
  border-color: rgba(14, 165, 233, 0.28);
}

.route-model {
  min-width: auto;
  padding: 0.65rem 0.9rem;
  font-weight: 700;
}

.route-claude {
  right: 5rem;
  bottom: 10.5rem;
  border-color: rgba(249, 115, 22, 0.25);
}

.route-gpt {
  right: 7.2rem;
  bottom: 6.5rem;
  border-color: rgba(34, 197, 94, 0.25);
}

.route-gemini {
  right: 4rem;
  bottom: 2.8rem;
  border-color: rgba(99, 102, 241, 0.25);
}

.route-line {
  position: absolute;
  right: 8rem;
  bottom: 8rem;
  width: 24rem;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(20, 184, 166, 0.7), transparent);
  transform-origin: right center;
}

.route-line-a {
  transform: rotate(-10deg);
}

.route-line-b {
  transform: rotate(0deg);
}

.route-line-c {
  transform: rotate(12deg);
}

:deep(.dark) .brand-hero {
  border-color: rgba(51, 65, 85, 0.85);
  background:
    linear-gradient(120deg, rgba(15, 23, 42, 0.94), rgba(8, 47, 73, 0.72) 55%, rgba(6, 78, 59, 0.58)),
    linear-gradient(180deg, rgba(15, 23, 42, 0.92), rgba(15, 23, 42, 0.78));
  box-shadow: 0 24px 90px rgba(0, 0, 0, 0.32);
}

:deep(.dark) .route-node {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.74);
  color: #f8fafc;
}

:deep(.dark) .route-node span {
  color: #94a3b8;
}

@media (max-width: 1024px) {
  .hero-copy {
    padding: 3.5rem 2rem;
  }

  .gateway-console {
    right: -4rem;
    width: 28rem;
    opacity: 0.34;
  }

  .route-source,
  .route-gateway,
  .route-model,
  .route-line {
    opacity: 0.28;
  }
}

@media (max-width: 640px) {
  .brand-hero {
    min-height: 560px;
    border-radius: 12px;
  }

  .hero-copy {
    padding: 2.5rem 1.25rem;
  }

  .gateway-console {
    top: auto;
    right: -8rem;
    bottom: 1.5rem;
    width: 26rem;
    opacity: 0.22;
  }
}

/* Terminal Container */
.terminal-container {
  position: relative;
  display: inline-block;
}

/* Terminal Window */
.terminal-window {
  width: 420px;
  background: linear-gradient(145deg, #1e293b 0%, #0f172a 100%);
  border-radius: 14px;
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.4),
    0 0 0 1px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  overflow: hidden;
  transform: perspective(1000px) rotateX(2deg) rotateY(-2deg);
  transition: transform 0.3s ease;
}

.terminal-window:hover {
  transform: perspective(1000px) rotateX(0deg) rotateY(0deg) translateY(-4px);
}

/* Terminal Header */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: rgba(30, 41, 59, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-buttons span {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.btn-close {
  background: #ef4444;
}
.btn-minimize {
  background: #eab308;
}
.btn-maximize {
  background: #22c55e;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 12px;
  font-family: ui-monospace, monospace;
  color: #64748b;
  margin-right: 52px;
}

/* Terminal Body */
.terminal-body {
  padding: 20px 24px;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 14px;
  line-height: 2;
}

.code-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  opacity: 0;
  animation: line-appear 0.5s ease forwards;
}

.line-1 {
  animation-delay: 0.3s;
}
.line-2 {
  animation-delay: 1s;
}
.line-3 {
  animation-delay: 1.8s;
}
.line-4 {
  animation-delay: 2.5s;
}

@keyframes line-appear {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.code-prompt {
  color: #22c55e;
  font-weight: bold;
}
.code-cmd {
  color: #38bdf8;
}
.code-flag {
  color: #a78bfa;
}
.code-url {
  color: #14b8a6;
}
.code-comment {
  color: #64748b;
  font-style: italic;
}
.code-success {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}
.code-response {
  color: #fbbf24;
}

/* Blinking Cursor */
.cursor {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: #22c55e;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

/* Dark mode adjustments */
:deep(.dark) .terminal-window {
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.6),
    0 0 0 1px rgba(20, 184, 166, 0.2),
    0 0 40px rgba(20, 184, 166, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}
</style>
