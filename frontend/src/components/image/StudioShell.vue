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
  { path: '/studio/generate', label: '生成图片' },
  { path: '/studio/history', label: '历史记录' },
  { path: '/studio/pricing', label: '价格与路由' }
]

const brandName = computed(() => appStore.siteName || 'FionaAI')
const sessionLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return '登录后恢复'
  }
  if (imageStudioStore.isHydrating) {
    return '恢复中'
  }
  const count = imageStudioStore.sessionGenerations.length
  if (count === 0) {
    return '暂无图片'
  }
  return `${count} 张图片`
})
const walletLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return '登录后显示'
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
  authStore.isAuthenticated ? '主站工作台' : '返回主站'
)
const keySetupRoute = computed(() => ({
  path: '/studio/generate',
  query:
    route.path === '/studio/generate'
      ? { ...route.query, configure: 'key' }
      : { configure: 'key' }
}))

function isActive(path: string): boolean {
  return route.path === path || (path !== '/studio' && route.path.startsWith(`${path}/`))
}
</script>

<template>
  <header class="studio-shell-header" :style="{ '--studio-shell-max-width': props.maxWidth }">
    <div class="studio-topbar">
      <RouterLink to="/studio/generate" class="brand-lockup">
        <span class="brand-mark">IS</span>
        <div>
          <strong>{{ brandName }} Studio</strong>
          <span>图片工作台</span>
        </div>
      </RouterLink>

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

      <div class="meta-cluster">
        <div class="meta-pill">
          <span>会话</span>
          <strong>{{ sessionLabel }}</strong>
        </div>
        <div class="meta-pill">
          <span>余额</span>
          <strong>{{ walletLabel }}</strong>
        </div>
        <RouterLink
          :to="keySetupRoute"
          class="main-site-link main-site-link-primary"
        >
          配置 API Key
        </RouterLink>
        <a :href="mainSiteURL" class="main-site-link">
          {{ mainSiteLabel }}
        </a>
      </div>
    </div>
  </header>
</template>

<style scoped>
.studio-shell-header {
  width: min(var(--studio-shell-max-width), 100%);
  margin: 0 auto 16px;
}

.studio-topbar {
  border: 1px solid rgba(18, 35, 30, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 12px 32px rgba(18, 35, 30, 0.06);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 8px;
}

.brand-lockup {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 190px;
  padding: 6px 8px;
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: #0f766e;
  color: #ffffff;
  font-size: 0.92rem;
  font-weight: 800;
}

.brand-lockup strong {
  display: block;
  font-size: 1rem;
}

.brand-lockup span {
  display: block;
  margin-top: 2px;
  color: rgba(18, 35, 30, 0.56);
  font-size: 0.78rem;
  font-weight: 700;
}

.studio-nav {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  flex: 1;
  padding: 4px;
  border-radius: 8px;
  background: rgba(18, 35, 30, 0.04);
}

.meta-cluster {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  flex-shrink: 0;
}

.meta-pill,
.main-site-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(18, 35, 30, 0.12);
  border-radius: 8px;
  background: #ffffff;
  color: inherit;
  text-decoration: none;
}

.meta-pill span {
  font-size: 0.78rem;
  color: rgba(18, 35, 30, 0.52);
}

.meta-pill strong,
.main-site-link {
  font-size: 0.88rem;
  font-weight: 700;
}

.main-site-link {
  transition: transform 160ms ease, border-color 160ms ease;
}

.main-site-link-primary {
  border-color: rgba(15, 118, 110, 0.32);
  background: rgba(15, 118, 110, 0.09);
  color: #0b5f59;
}

.main-site-link:hover,
.nav-pill:hover {
  transform: translateY(-1px);
}

.nav-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  padding: 0 14px;
  border-radius: 6px;
  color: rgba(18, 35, 30, 0.78);
  font-weight: 700;
  font-size: 0.92rem;
  text-decoration: none;
  transition: transform 160ms ease, background-color 160ms ease, color 160ms ease, box-shadow 160ms ease;
}

.nav-pill-active {
  background: #12352d;
  color: #ffffff;
}

@media (max-width: 900px) {
  .studio-topbar {
    display: grid;
    grid-template-columns: 1fr;
  }

  .meta-cluster {
    justify-content: flex-start;
  }
}

@media (max-width: 760px) {
  .studio-shell-header {
    margin-bottom: 16px;
  }

  .nav-pill,
  .meta-pill,
  .main-site-link {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
