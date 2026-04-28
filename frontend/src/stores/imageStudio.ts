import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useAuthStore } from './auth'

export interface StudioSessionGeneration {
  id: string
  created: number
  prompt: string
  revisedPrompt: string
  model: string
  size: string
  keyName: string
  imageSrc: string
}

interface PersistedStudioSessionRecord {
  version: number
  items: StudioSessionGeneration[]
}

const MAX_SESSION_GENERATIONS = 24
const IMAGE_STUDIO_DB_NAME = 'sub2api-image-studio'
const IMAGE_STUDIO_DB_VERSION = 1
const IMAGE_STUDIO_STORE_NAME = 'session-generations'
const IMAGE_STUDIO_RECORD_VERSION = 1

function isStudioSessionGeneration(value: unknown): value is StudioSessionGeneration {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.created === 'number' &&
    Number.isFinite(candidate.created) &&
    typeof candidate.prompt === 'string' &&
    typeof candidate.revisedPrompt === 'string' &&
    typeof candidate.model === 'string' &&
    typeof candidate.size === 'string' &&
    typeof candidate.keyName === 'string' &&
    typeof candidate.imageSrc === 'string'
  )
}

function mergeGenerations(primary: unknown[], secondary: unknown[]): StudioSessionGeneration[] {
  const merged: StudioSessionGeneration[] = []
  const seenIds = new Set<string>()

  for (const item of [...primary, ...secondary]) {
    if (!isStudioSessionGeneration(item) || seenIds.has(item.id)) {
      continue
    }
    seenIds.add(item.id)
    merged.push(item)
    if (merged.length >= MAX_SESSION_GENERATIONS) {
      break
    }
  }

  return merged
}

function normalizePersistedGenerations(value: unknown): StudioSessionGeneration[] {
  if (Array.isArray(value)) {
    return mergeGenerations(value, [])
  }

  if (!value || typeof value !== 'object') {
    return []
  }

  const record = value as Partial<PersistedStudioSessionRecord>
  return Array.isArray(record.items) ? mergeGenerations(record.items, []) : []
}

function openImageStudioDatabase(): Promise<IDBDatabase | null> {
  if (typeof window === 'undefined' || !window.indexedDB) {
    return Promise.resolve(null)
  }

  return new Promise((resolve, reject) => {
    const request = window.indexedDB.open(IMAGE_STUDIO_DB_NAME, IMAGE_STUDIO_DB_VERSION)

    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(IMAGE_STUDIO_STORE_NAME)) {
        database.createObjectStore(IMAGE_STUDIO_STORE_NAME)
      }
    }

    request.onsuccess = () => {
      resolve(request.result)
    }

    request.onerror = () => {
      reject(request.error ?? new Error('Failed to open image studio database'))
    }
  })
}

async function readPersistedGenerations(ownerKey: string): Promise<StudioSessionGeneration[]> {
  const database = await openImageStudioDatabase()
  if (!database) {
    return []
  }

  return new Promise((resolve, reject) => {
    const transaction = database.transaction(IMAGE_STUDIO_STORE_NAME, 'readonly')
    const store = transaction.objectStore(IMAGE_STUDIO_STORE_NAME)
    const request = store.get(ownerKey)
    let settled = false

    transaction.oncomplete = () => {
      database.close()
    }

    transaction.onabort = () => {
      database.close()
      if (!settled) {
        reject(transaction.error ?? new Error('Failed to load stored image session'))
      }
    }

    transaction.onerror = () => {
      if (!settled) {
        reject(transaction.error ?? new Error('Failed to load stored image session'))
      }
    }

    request.onsuccess = () => {
      settled = true
      resolve(normalizePersistedGenerations(request.result))
    }

    request.onerror = () => {
      settled = true
      reject(request.error ?? new Error('Failed to read stored image session'))
    }
  })
}

async function writePersistedGenerations(ownerKey: string, items: StudioSessionGeneration[]): Promise<void> {
  const database = await openImageStudioDatabase()
  if (!database) {
    return
  }

  return new Promise((resolve, reject) => {
    const transaction = database.transaction(IMAGE_STUDIO_STORE_NAME, 'readwrite')
    const store = transaction.objectStore(IMAGE_STUDIO_STORE_NAME)

    if (items.length === 0) {
      store.delete(ownerKey)
    } else {
      store.put(
        {
          version: IMAGE_STUDIO_RECORD_VERSION,
          items
        } satisfies PersistedStudioSessionRecord,
        ownerKey
      )
    }

    transaction.oncomplete = () => {
      database.close()
      resolve()
    }

    transaction.onabort = () => {
      database.close()
      reject(transaction.error ?? new Error('Failed to persist image session'))
    }

    transaction.onerror = () => {
      reject(transaction.error ?? new Error('Failed to persist image session'))
    }
  })
}

export const useImageStudioStore = defineStore('imageStudio', () => {
  const authStore = useAuthStore()

  const sessionGenerations = ref<StudioSessionGeneration[]>([])
  const isHydrating = ref(false)
  const hasSessionGenerations = computed(() => sessionGenerations.value.length > 0)
  const currentOwnerKey = computed(() => (authStore.user?.id ? `user:${authStore.user.id}` : null))

  let hydratedOwnerKey: string | null = null
  let hydrationPromise: Promise<void> | null = null
  let persistQueue: Promise<void> = Promise.resolve()

  watch(
    currentOwnerKey,
    (ownerKey, previousOwnerKey) => {
      if (ownerKey === previousOwnerKey) {
        return
      }

      sessionGenerations.value = []
      hydratedOwnerKey = null
      hydrationPromise = null
      isHydrating.value = false

      if (ownerKey) {
        void ensureHydrated()
      }
    },
    { immediate: true }
  )

  async function ensureHydrated(): Promise<void> {
    const ownerKey = currentOwnerKey.value
    if (!ownerKey) {
      sessionGenerations.value = []
      hydratedOwnerKey = null
      isHydrating.value = false
      return
    }

    if (hydratedOwnerKey === ownerKey) {
      return
    }

    if (hydrationPromise) {
      return hydrationPromise
    }

    isHydrating.value = true

    const currentPromise = (async () => {
      const persistedItems = await readPersistedGenerations(ownerKey)
      if (currentOwnerKey.value !== ownerKey) {
        return
      }

      sessionGenerations.value = mergeGenerations(sessionGenerations.value, persistedItems)
      hydratedOwnerKey = ownerKey
    })()
      .catch((error) => {
        console.error('Failed to restore persisted image studio session:', error)
        if (currentOwnerKey.value === ownerKey) {
          hydratedOwnerKey = ownerKey
        }
      })
      .finally(() => {
        if (hydrationPromise === currentPromise) {
          hydrationPromise = null
        }
        if (currentOwnerKey.value === ownerKey) {
          isHydrating.value = false
        }
      })

    hydrationPromise = currentPromise
    return currentPromise
  }

  function queuePersist(): void {
    const ownerKey = currentOwnerKey.value
    if (!ownerKey) {
      return
    }

    const snapshot = mergeGenerations(sessionGenerations.value, [])
    persistQueue = persistQueue
      .catch(() => undefined)
      .then(async () => {
        try {
          await writePersistedGenerations(ownerKey, snapshot)
        } catch (error) {
          console.error('Failed to persist image studio session:', error)
        }
      })
  }

  function prependGenerations(items: StudioSessionGeneration[]): void {
    if (!Array.isArray(items) || items.length === 0) {
      return
    }

    sessionGenerations.value = mergeGenerations(items, sessionGenerations.value)
    queuePersist()
  }

  function clearSessionGenerations(): void {
    sessionGenerations.value = []
    hydratedOwnerKey = currentOwnerKey.value
    queuePersist()
  }

  return {
    sessionGenerations,
    hasSessionGenerations,
    isHydrating,
    ensureHydrated,
    prependGenerations,
    clearSessionGenerations
  }
})
