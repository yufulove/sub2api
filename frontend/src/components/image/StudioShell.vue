<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useAppStore, useAuthStore, useImageStudioStore } from '@/stores'
import { getMainSiteHomePath, resolveMainSiteURL } from '@/utils/siteMode'
import { formatWalletMoneyFromInternal } from '@/utils/walletDisplay'

const props = withDefaults(
  defineProps<{
    maxWidth?: string
  }>(),
  {
    maxWidth: '1160px'
  }
)

interface StudioNavItem {
  path: string
  label: string
}

const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const imageStudioStore = useImageStudioStore()

const navItems: StudioNavItem[] = [
  { path: '/studio/generate', label: 'Generate' },
  { path: '/studio/history', label: 'History' },
  { path: '/studio/pricing', label: 'Pricing' },
  { path: '/studio', label: 'Overview' }
]

const brandName = computed(() => appStore.siteName || 'FionaAI')
const sessionLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return 'Sign in to restore cards'
  }
  if (imageStudioStore.isHydrating) {
    return 'Restoring session'
  }
  const count = imageStudioStore.sessionGenerations.length
  if (count === 0) {
    return 'No recent renders'
  }
  return `${count} saved card${count === 1 ? '' : 's'}`
})
const walletLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return 'Shared after sign-in'
  }
  return formatWalletMoneyFromInternal(authStore.user?.balance ?? 0, appStore.cachedPublicSettings)
})
const mainSiteURL = computed(() =>
  resolveMainSiteURL(
    getMainSiteHomePath({
      isAuthenticated: authStore.isAuthenticated,
      isAdmin: authStore.isAdmin
    })
  )
)
const mainSiteLabel = computed(() =>
  authStore.isAuthenticated ? 'Main dashboard' : 'Main site'
)

function isActive(path: string): boolean {
  return route.path === path || (path !== '/studio' && route.path.startsWith(`${path}/`))
}
</script>

<template>
  <header class="studio-shell-header" :style="{ '--studio-shell-max-width': props.maxWidth }">
    <div class="studio-shell-card">
      <div class="brand-cluster">
        <RouterLink to="/studio" class="brand-lockup">
          <span class="brand-mark">IS</span>
          <div>
            <p class="brand-kicker">Image Workspace</p>
            <strong>{{ brandName }} Studio</strong>
          </div>
        </RouterLink>
        <p class="brand-note">
          Start generation here. Wallet, login, and key management stay shared with the main site.
        </p>
      </div>

      <div class="meta-cluster">
        <div class="meta-pill">
          <span>Session</span>
          <strong>{{ sessionLabel }}</strong>
        </div>
        <div class="meta-pill">
          <span>Wallet</span>
          <strong>{{ walletLabel }}</strong>
        </div>
        <a :href="mainSiteURL" class="main-site-link">
          {{ mainSiteLabel }}
        </a>
      </div>
    </div>

    <nav class="studio-nav" aria-label="Studio navigation">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :class="['nav-pill', isActive(item.path) && 'nav-pill-active']"
      >
        {{ item.label }}
      </RouterLink>
    </nav>
  </header>
</template>

<style scoped>
.studio-shell-header {
  width: min(var(--studio-shell-max-width), 100%);
  margin: 0 auto 20px;
}

.studio-shell-card,
.studio-nav {
  border: 1px solid rgba(18, 35, 30, 0.12);
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.68);
  backdrop-filter: blur(16px);
  box-shadow: 0 20px 60px rgba(18, 35, 30, 0.07);
}

.studio-shell-card {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  padding: 20px 22px;
}

.brand-cluster {
  display: grid;
  gap: 10px;
}

.brand-lockup {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  color: inherit;
  text-decoration: none;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(198, 92, 44, 0.94) 0%, rgba(218, 124, 67, 0.94) 100%);
  color: #fffaf5;
  font-size: 0.94rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  box-shadow: 0 14px 28px rgba(198, 92, 44, 0.22);
}

.brand-kicker {
  margin: 0 0 6px;
  font-size: 0.72rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(18, 35, 30, 0.54);
}

.brand-lockup strong {
  font-size: 1.15rem;
  letter-spacing: -0.03em;
}

.brand-note {
  margin: 0;
  max-width: 40rem;
  color: rgba(18, 35, 30, 0.72);
  line-height: 1.58;
}

.meta-cluster {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.meta-pill,
.main-site-link {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 16px;
  border: 1px solid rgba(18, 35, 30, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.76);
  color: inherit;
  text-decoration: none;
}

.meta-pill span {
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: rgba(18, 35, 30, 0.52);
}

.meta-pill strong,
.main-site-link {
  font-size: 0.92rem;
  font-weight: 700;
}

.main-site-link {
  transition: transform 160ms ease, border-color 160ms ease;
}

.main-site-link:hover,
.nav-pill:hover {
  transform: translateY(-1px);
}

.studio-nav {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
  padding: 10px;
}

.nav-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 42px;
  padding: 0 16px;
  border-radius: 999px;
  color: rgba(18, 35, 30, 0.78);
  font-weight: 700;
  text-decoration: none;
  transition: transform 160ms ease, background-color 160ms ease, color 160ms ease, box-shadow 160ms ease;
}

.nav-pill-active {
  background: linear-gradient(135deg, rgba(18, 35, 30, 0.94) 0%, rgba(34, 71, 58, 0.94) 100%);
  color: #fbf7f1;
  box-shadow: 0 14px 28px rgba(18, 35, 30, 0.18);
}

@media (max-width: 900px) {
  .studio-shell-card {
    flex-direction: column;
    align-items: stretch;
  }

  .meta-cluster {
    justify-content: flex-start;
  }
}

@media (max-width: 760px) {
  .studio-shell-header {
    margin-bottom: 16px;
  }

  .studio-shell-card,
  .studio-nav {
    border-radius: 24px;
  }

  .studio-shell-card {
    padding: 18px;
  }

  .studio-nav {
    padding: 8px;
  }

  .nav-pill,
  .meta-pill,
  .main-site-link {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
