<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import StudioShell from '@/components/image/StudioShell.vue'
import { useAuthStore, useImageStudioStore } from '@/stores'
import { resolveMainSiteURL } from '@/utils/siteMode'

const router = useRouter()
const authStore = useAuthStore()
const imageStudioStore = useImageStudioStore()

const mainSiteDashboardURL = computed(() => resolveMainSiteURL('/dashboard'))

const primaryActionLabel = computed(() =>
  authStore.isAuthenticated ? 'Open generate workspace' : 'Sign in to start'
)

const sessionCardCount = computed(() => imageStudioStore.sessionGenerations.length)
const sessionStatusLabel = computed(() => {
  if (!authStore.isAuthenticated) {
    return 'Sign in to restore browser session cards'
  }
  if (imageStudioStore.isHydrating) {
    return 'Restoring browser session'
  }
  return `${sessionCardCount.value} saved browser card${sessionCardCount.value === 1 ? '' : 's'}`
})

async function handlePrimaryAction() {
  await router.push('/studio/generate')
}

onMounted(() => {
  void imageStudioStore.ensureHydrated()
})
</script>

<template>
  <main class="studio-shell">
    <StudioShell max-width="1160px" />

    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">Image Studio</p>
        <h1>Open the generate workspace first, then track price and history</h1>
        <p class="lead">
          This subsite is for making images, not reading backend notes. Start with generation, keep recent
          renders in the same account session, and check route pricing before you spend balance.
        </p>

        <div class="hero-actions">
          <button class="primary-button" type="button" @click="handlePrimaryAction">
            {{ primaryActionLabel }}
          </button>
          <RouterLink class="secondary-link" to="/studio/history">Recent history</RouterLink>
          <RouterLink class="secondary-link" to="/studio/pricing">View pricing</RouterLink>
        </div>
      </div>

      <aside class="hero-panel">
        <span class="panel-tag">Start Here</span>
        <dl class="panel-stats">
          <div>
            <dt>Generate</dt>
            <dd>Prompt, route, model, size, and cost in one workspace</dd>
          </div>
          <div>
            <dt>History</dt>
            <dd>{{ sessionStatusLabel }}</dd>
          </div>
          <div>
            <dt>Billing</dt>
            <dd>Live group pricing with the shared wallet balance</dd>
          </div>
          <div>
            <dt>Main Site</dt>
            <dd>Use it for top-up and account settings</dd>
          </div>
        </dl>
      </aside>
    </section>

    <section class="route-grid">
      <article class="route-card">
        <span class="route-index">01</span>
        <h2>Generate</h2>
        <p>
          Pick a usable route, write a prompt, choose size and model, and render images without leaving the page.
        </p>
        <RouterLink class="inline-link" to="/studio/generate">Start generating</RouterLink>
      </article>

      <article class="route-card">
        <span class="route-index">02</span>
        <h2>History</h2>
        <p>
          See the renders you just made, then use billed request records to audit older image activity.
        </p>
        <RouterLink class="inline-link" to="/studio/history">Open history</RouterLink>
      </article>

      <article class="route-card">
        <span class="route-index">03</span>
        <h2>Pricing</h2>
        <p>
          Compare ready routes, see which groups are blocked, and check 1K, 2K, and 4K image pricing before you run.
        </p>
        <RouterLink class="inline-link" to="/studio/pricing">Check pricing</RouterLink>
      </article>
    </section>

    <section class="workflow">
      <div class="workflow-copy">
        <p class="eyebrow">Workflow</p>
        <h2>One image workspace, one shared account system</h2>
        <p>
          The image workflow lives here, while balance and login still come from the same
          account. That keeps the image entry clean without splitting the wallet or the auth model.
        </p>
      </div>

      <div class="workflow-steps">
        <article class="step-card">
          <strong>Generate Fast</strong>
          <p>Go straight into the render workspace instead of landing on a generic dashboard.</p>
        </article>
        <article class="step-card">
          <strong>Review Cost</strong>
          <p>Use route pricing and wallet balance together so size and group choice stay visible.</p>
        </article>
        <article class="step-card">
          <strong>Keep Context</strong>
          <p>Recent renders stay available in the same browser session, then move into history for later review.</p>
        </article>
      </div>
    </section>

    <section class="footer-panel">
      <div>
        <p class="eyebrow">Main Site</p>
        <h2>Top up balance and manage your account on the main site</h2>
        <p>
          The main dashboard still owns wallet operations and account settings. Come back here when you want to generate.
        </p>
      </div>
      <div class="footer-actions">
        <a class="secondary-link" :href="mainSiteDashboardURL">Open dashboard</a>
      </div>
    </section>
  </main>
</template>

<style scoped>
.studio-shell {
  --paper: #f4efe5;
  --panel: rgba(255, 252, 248, 0.8);
  --card: rgba(255, 255, 255, 0.75);
  --ink: #10231b;
  --muted: rgba(16, 35, 27, 0.7);
  --line: rgba(16, 35, 27, 0.12);
  --accent: #d9623a;
  --accent-deep: #9f381d;
  min-height: 100vh;
  padding: 40px 24px 72px;
  background: linear-gradient(180deg, #faf5ee 0%, var(--paper) 100%);
  color: var(--ink);
  font-family: "Space Grotesk", "Noto Sans SC", sans-serif;
}

.hero,
.route-grid,
.workflow,
.footer-panel {
  width: min(1160px, 100%);
  margin: 0 auto;
}

.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(300px, 0.82fr);
  gap: 20px;
  align-items: stretch;
}

.hero-copy,
.hero-panel,
.route-card,
.step-card,
.footer-panel {
  border: 1px solid var(--line);
  border-radius: 28px;
  background: var(--panel);
  backdrop-filter: blur(14px);
  box-shadow: 0 20px 60px rgba(16, 35, 27, 0.08);
}

.hero-copy,
.hero-panel,
.route-card,
.step-card,
.footer-panel {
  padding: 28px;
}

.eyebrow {
  margin: 0 0 12px;
  font-size: 0.78rem;
  letter-spacing: 0;
  text-transform: uppercase;
  color: var(--accent-deep);
}

.hero-copy h1,
.workflow-copy h2,
.footer-panel h2 {
  margin: 0;
  line-height: 0.94;
  letter-spacing: 0;
}

.hero-copy h1 {
  font-size: clamp(2.5rem, 5vw, 4.4rem);
}

.lead,
.workflow-copy p,
.route-card p,
.step-card p {
  color: var(--muted);
}

.lead {
  margin: 18px 0 0;
  max-width: 42rem;
  line-height: 1.76;
}

.hero-actions,
.footer-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.hero-actions {
  margin-top: 24px;
}

.primary-button,
.secondary-link,
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
  transition: transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease;
}

.primary-button {
  border: none;
  background: linear-gradient(135deg, var(--accent) 0%, #f09a62 100%);
  color: #fff8f2;
  box-shadow: 0 18px 34px rgba(217, 98, 58, 0.22);
  cursor: pointer;
}

.secondary-link,
.inline-link {
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.6);
}

.primary-button:hover,
.secondary-link:hover,
.inline-link:hover {
  transform: translateY(-1px);
}

.panel-tag,
.route-index {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(16, 35, 27, 0.08);
  font-size: 0.78rem;
  font-weight: 700;
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
  letter-spacing: 0;
  text-transform: uppercase;
  color: rgba(16, 35, 27, 0.55);
}

.panel-stats dd {
  margin: 8px 0 0;
  font-size: 1.06rem;
  line-height: 1.5;
}

.route-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.route-index {
  color: var(--accent-deep);
}

.route-card h2,
.step-card strong {
  margin: 18px 0 0;
}

.route-card p,
.step-card p,
.workflow-copy p,
.footer-panel p {
  margin: 12px 0 0;
  line-height: 1.72;
}

.inline-link {
  margin-top: 18px;
}

.workflow {
  margin-top: 20px;
}

.workflow-copy h2,
.footer-panel h2 {
  font-size: clamp(1.9rem, 4vw, 3rem);
}

.workflow-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.step-card strong {
  display: block;
  color: var(--accent-deep);
  font-size: 0.92rem;
  letter-spacing: 0;
  text-transform: uppercase;
}

.footer-panel {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-top: 20px;
}

@media (max-width: 1120px) {
  .hero,
  .route-grid,
  .workflow-steps,
  .footer-panel {
    grid-template-columns: 1fr;
  }

  .footer-panel {
    display: grid;
  }
}

@media (max-width: 760px) {
  .studio-shell {
    padding-left: 16px;
    padding-right: 16px;
  }

  .hero-copy,
  .hero-panel,
  .route-card,
  .step-card,
  .footer-panel {
    padding: 22px;
  }
}
</style>
