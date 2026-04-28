<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getAuthBridgeSource, readAuthBridgeSnapshot } from '@/utils/crossSiteAuth'
import { resolveImageSiteURL, resolveMainSiteURL } from '@/utils/siteMode'

const route = useRoute()

const requestId = computed(() =>
  typeof route.query.request_id === 'string' ? route.query.request_id.trim() : ''
)

const targetOrigin = computed(() => {
  const raw = typeof route.query.origin === 'string' ? route.query.origin.trim() : ''
  if (!raw) {
    return ''
  }

  try {
    const normalized = new URL(raw).origin
    return allowedOrigins.value.has(normalized) ? normalized : ''
  } catch {
    return ''
  }
})

const allowedOrigins = computed(() => {
  const origins = new Set<string>()
  if (typeof window === 'undefined') {
    return origins
  }

  try {
    origins.add(new URL(resolveMainSiteURL('/'), window.location.href).origin)
  } catch {
    // Ignore malformed config
  }

  try {
    origins.add(new URL(resolveImageSiteURL('/studio'), window.location.href).origin)
  } catch {
    // Ignore malformed config
  }

  return origins
})

onMounted(() => {
  const destination = targetOrigin.value
  const correlationId = requestId.value
  const receiver = window.parent !== window ? window.parent : window.opener

  if (!destination || !correlationId || !receiver) {
    return
  }

  receiver.postMessage(
    {
      source: getAuthBridgeSource(),
      type: 'auth-transfer',
      requestId: correlationId,
      payload: readAuthBridgeSnapshot()
    },
    destination
  )
})
</script>

<template>
  <main class="bridge-shell">
    <p>Auth bridge</p>
  </main>
</template>

<style scoped>
.bridge-shell {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: #0f172a;
  color: #cbd5e1;
  font: 500 12px/1.4 "Space Grotesk", "Noto Sans SC", sans-serif;
}
</style>
