<template>
  <div class="card overflow-hidden">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('dashboard.gettingStarted.title') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('dashboard.gettingStarted.description') }}
          </p>
        </div>
        <div class="inline-flex items-center rounded-full bg-primary-50 px-3 py-1 text-sm font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
          {{ t('dashboard.gettingStarted.completed', { done: completedCount, total: steps.length }) }}
        </div>
      </div>
    </div>

    <div class="p-4 md:p-6">
      <div class="mb-5 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="min-w-0 flex-1">
          <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
            <div
              class="h-full rounded-full bg-primary-500 transition-all duration-300"
              :style="{ width: `${completionPercent}%` }"
            />
          </div>
          <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">
            {{ headline }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-primary justify-center"
          @click="goPrimaryAction"
        >
          {{ primaryActionLabel }}
          <Icon name="arrowRight" size="sm" class="ml-2" />
        </button>
      </div>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <button
          v-for="step in steps"
          :key="step.id"
          type="button"
          class="group flex min-h-[118px] flex-col rounded-xl border p-4 text-left transition-all duration-200"
          :class="step.done
            ? 'border-green-200 bg-green-50/60 hover:bg-green-50 dark:border-green-900/60 dark:bg-green-900/10 dark:hover:bg-green-900/20'
            : step.isNext
              ? 'border-primary-200 bg-primary-50/70 shadow-sm hover:bg-primary-50 dark:border-primary-900/70 dark:bg-primary-900/10 dark:hover:bg-primary-900/20'
              : 'border-gray-200 bg-gray-50/80 hover:bg-gray-100 dark:border-dark-700 dark:bg-dark-800/50 dark:hover:bg-dark-800'"
          @click="goStep(step)"
        >
          <div class="mb-3 flex items-center justify-between gap-3">
            <span
              class="flex h-9 w-9 items-center justify-center rounded-lg"
              :class="step.done
                ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                : step.isNext
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
                  : 'bg-white text-gray-500 ring-1 ring-inset ring-gray-200 dark:bg-dark-900 dark:text-dark-300 dark:ring-dark-700'"
            >
              <Icon :name="step.done ? 'check' : step.icon" size="md" :stroke-width="2" />
            </span>
            <span
              class="rounded-full px-2 py-0.5 text-xs font-medium"
              :class="step.done
                ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                : step.isNext
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'"
            >
              {{ step.done ? t('dashboard.gettingStarted.done') : step.isNext ? t('dashboard.gettingStarted.next') : t('dashboard.gettingStarted.pending') }}
            </span>
          </div>

          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ step.title }}
          </p>
          <p class="mt-1 flex-1 text-sm leading-5 text-gray-600 dark:text-dark-400">
            {{ step.description }}
          </p>
          <span class="mt-3 inline-flex items-center text-sm font-medium text-primary-600 group-hover:text-primary-700 dark:text-primary-400 dark:group-hover:text-primary-300">
            {{ step.actionLabel }}
            <Icon name="chevronRight" size="xs" class="ml-1" />
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserDashboardStats } from '@/api/usage'
import type { User } from '@/types'

type StepId = 'funding' | 'key' | 'client' | 'usage'
type StepIcon = 'creditCard' | 'key' | 'terminal' | 'chart'

interface StepItem {
  id: StepId
  icon: StepIcon
  title: string
  description: string
  actionLabel: string
  path: string
  done: boolean
  isNext: boolean
}

const props = defineProps<{
  stats: UserDashboardStats
  balance: number
  user: User | null
  isSimple: boolean
  purchaseEnabled: boolean
}>()

const router = useRouter()
const { t } = useI18n()

const hasActiveSubscription = computed(() =>
  props.user?.subscriptions?.some((subscription) => subscription.status === 'active') ?? false
)

const fundingPath = computed(() => (props.purchaseEnabled ? '/purchase' : '/redeem'))
const fundingActionLabel = computed(() =>
  props.purchaseEnabled
    ? t('dashboard.gettingStarted.steps.funding.ctaPurchase')
    : t('dashboard.gettingStarted.steps.funding.ctaRedeem')
)

const rawSteps = computed(() => {
  const items: Array<Omit<StepItem, 'isNext'>> = []

  if (!props.isSimple) {
    items.push({
      id: 'funding',
      icon: 'creditCard',
      title: t('dashboard.gettingStarted.steps.funding.title'),
      description: t('dashboard.gettingStarted.steps.funding.description'),
      actionLabel: fundingActionLabel.value,
      path: fundingPath.value,
      done: props.balance > 0 || hasActiveSubscription.value
    })
  }

  items.push(
    {
      id: 'key',
      icon: 'key',
      title: t('dashboard.gettingStarted.steps.key.title'),
      description: t('dashboard.gettingStarted.steps.key.description'),
      actionLabel: t('dashboard.gettingStarted.steps.key.cta'),
      path: '/keys',
      done: props.stats.active_api_keys > 0
    },
    {
      id: 'client',
      icon: 'terminal',
      title: t('dashboard.gettingStarted.steps.client.title'),
      description: t('dashboard.gettingStarted.steps.client.description'),
      actionLabel: t('dashboard.gettingStarted.steps.client.cta'),
      path: '/keys',
      done: props.stats.total_requests > 0
    },
    {
      id: 'usage',
      icon: 'chart',
      title: t('dashboard.gettingStarted.steps.usage.title'),
      description: t('dashboard.gettingStarted.steps.usage.description'),
      actionLabel: t('dashboard.gettingStarted.steps.usage.cta'),
      path: '/usage',
      done: props.stats.total_requests > 0
    }
  )

  return items
})

const firstIncompleteId = computed(() => rawSteps.value.find((step) => !step.done)?.id ?? null)

const steps = computed<StepItem[]>(() =>
  rawSteps.value.map((step) => ({
    ...step,
    isNext: step.id === firstIncompleteId.value
  }))
)

const completedCount = computed(() => steps.value.filter((step) => step.done).length)
const completionPercent = computed(() => Math.round((completedCount.value / steps.value.length) * 100))
const nextStep = computed(() => steps.value.find((step) => step.isNext) ?? null)

const headline = computed(() => {
  if (!nextStep.value) {
    return t('dashboard.gettingStarted.allDoneDescription')
  }
  return t('dashboard.gettingStarted.nextStep', { step: nextStep.value.title })
})

const primaryActionLabel = computed(() =>
  nextStep.value?.actionLabel ?? t('dashboard.gettingStarted.reviewUsage')
)

function goPrimaryAction(): void {
  router.push(nextStep.value?.path ?? '/usage')
}

function goStep(step: StepItem): void {
  router.push(step.path)
}
</script>
