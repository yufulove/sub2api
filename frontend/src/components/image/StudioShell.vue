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
  { path: '/studio/generate', label: '生成' },
  { path: '/studio/history', label: '历史' },
  { path: '/studio/pricing', label: '价格' },
  { path: '/studio', label: '概览' }
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

function isActive(path: string): boolean {
  return route.path === path || (path !== '/studio' && route.path.startsWith(`${path}/`))
}
</script>

<template>
  <header class="studio-shell-header" :style="{ '--studio-shell-max-width': props.maxWidth }">
    <div class="studio-shell-card">
      <div class="brand-cluster">
        <RouterLink to="/studio" class="brand-lockup">
          <span class="brand-mark">图</span>
          <div>
            <p class="brand-kicker">图片工作台</p>
            <strong>{{ brandName }} Studio</strong>
          </div>
        </RouterLink>
        <p class="brand-note">
          独立处理图片生成，登录、钱包和 API Key 继续沿用主站账号体系。
        </p>
      </div>

      <div class="meta-cluster">
        <div class="meta-pill">
          <span>会话</span>
          <strong>{{ sessionLabel }}</strong>
        </div>
        <div class="meta-pill">
          <span>余额</span>
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
  margin: 0 auto 16px;
}

.studio-shell-card,
.studio-nav {
  border: 1px solid rgba(18, 35, 30, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 12px 32px rgba(18, 35, 30, 0.06);
}

.studio-shell-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
}

.brand-cluster {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.brand-lockup {
  display: inline-flex;
  align-items: center;
  gap: 12px;
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
  font-size: 1rem;
  font-weight: 800;
}

.brand-kicker {
  margin: 0 0 4px;
  font-size: 0.78rem;
  color: rgba(18, 35, 30, 0.54);
}

.brand-lockup strong {
  font-size: 1rem;
}

.brand-note {
  margin: 0;
  max-width: 36rem;
  color: rgba(18, 35, 30, 0.72);
  line-height: 1.5;
  font-size: 0.92rem;
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

.main-site-link:hover,
.nav-pill:hover {
  transform: translateY(-1px);
}

.studio-nav {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
  padding: 6px;
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
  .studio-shell-card {
    flex-direction: column;
    align-items: stretch;
  }

  .brand-cluster {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .meta-cluster {
    justify-content: flex-start;
  }
}

@media (max-width: 760px) {
  .studio-shell-header {
    margin-bottom: 16px;
  }

  .studio-shell-card {
    padding: 14px;
  }

  .studio-nav {
    padding: 6px;
  }

  .nav-pill,
  .meta-pill,
  .main-site-link {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
