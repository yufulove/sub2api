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
  thumbnailSrc?: string
}

interface PersistedStudioSessionRecord {
  version: number
  items: Array<StudioSessionGeneration | StoredStudioSessionMetadata>
}

type StoredStudioSessionMetadata = Omit<StudioSessionGeneration, 'imageSrc'> & {
  imageStoreKey: string
}

interface PersistedStudioImagePayload {
  version: number
  image: Blob | string
}

const MAX_SESSION_GENERATIONS = 24
const PERSIST_GENERATION_LIMITS = [24, 12, 6, 3]
const IMAGE_STUDIO_DB_NAME = 'sub2api-image-studio'
const IMAGE_STUDIO_DB_VERSION = 1
const IMAGE_STUDIO_STORE_NAME = 'session-generations'
const IMAGE_STUDIO_RECORD_VERSION = 2
const IMAGE_STUDIO_IMAGE_RECORD_VERSION = 1

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
    typeof candidate.imageSrc === 'string' &&
    (candidate.thumbnailSrc == null || typeof candidate.thumbnailSrc === 'string')
  )
}

function isStoredStudioSessionMetadata(value: unknown): value is StoredStudioSessionMetadata {
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
    typeof candidate.imageStoreKey === 'string' &&
    (candidate.thumbnailSrc == null || typeof candidate.thumbnailSrc === 'string')
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

function imageStorageKey(ownerKey: string, id: string): string {
  return `${ownerKey}:image:${id}`
}

function cardToStoredMetadata(ownerKey: string, card: StudioSessionGeneration): StoredStudioSessionMetadata {
  return {
    id: card.id,
    created: card.created,
    prompt: card.prompt,
    revisedPrompt: card.revisedPrompt,
    model: card.model,
    size: card.size,
    keyName: card.keyName,
    thumbnailSrc: card.thumbnailSrc,
    imageStoreKey: imageStorageKey(ownerKey, card.id)
  }
}

function collectImageStoreKeys(value: unknown): string[] {
  if (!value || typeof value !== 'object') {
    return []
  }

  const record = value as Partial<PersistedStudioSessionRecord>
  if (!Array.isArray(record.items)) {
    return []
  }

  return record.items
    .map((item) => (isStoredStudioSessionMetadata(item) ? item.imageStoreKey : null))
    .filter((item): item is string => typeof item === 'string' && item.trim() !== '')
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

function dataURLToPersistedImage(imageSrc: string): Blob | string {
  if (typeof window === 'undefined' || typeof window.atob !== 'function') {
    return imageSrc
  }

  const match = /^data:([^;,]+)?(;base64)?,(.*)$/s.exec(imageSrc)
  if (!match || match[2] !== ';base64') {
    return imageSrc
  }

  try {
    const binary = window.atob(match[3])
    const bytes = new Uint8Array(binary.length)
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index)
    }
    return new Blob([bytes], { type: match[1] || 'image/png' })
  } catch {
    return imageSrc
  }
}

function blobToDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      resolve(typeof reader.result === 'string' ? reader.result : '')
    }
    reader.onerror = () => {
      reject(reader.error ?? new Error('Failed to read stored image'))
    }
    reader.readAsDataURL(blob)
  })
}

async function resolvePersistedImagePayload(value: unknown): Promise<string | null> {
  const payload = value as Partial<PersistedStudioImagePayload> | Blob | string | null
  if (typeof payload === 'string') {
    return payload
  }
  if (typeof Blob !== 'undefined' && payload instanceof Blob) {
    return blobToDataURL(payload)
  }
  if (payload && typeof payload === 'object') {
    const record = payload as Partial<PersistedStudioImagePayload>
    if (typeof record.image === 'string') {
      return record.image
    }
    if (typeof Blob !== 'undefined' && record.image instanceof Blob) {
      return blobToDataURL(record.image)
    }
  }
  return null
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

  try {
    const persisted = await readStoreValue(database, ownerKey)
    const record = persisted as Partial<PersistedStudioSessionRecord> | null

    if (record?.version === IMAGE_STUDIO_RECORD_VERSION && Array.isArray(record.items)) {
      const restored: StudioSessionGeneration[] = []
      for (const item of record.items) {
        if (!isStoredStudioSessionMetadata(item)) {
          continue
        }
        const imageSrc = await resolvePersistedImagePayload(await readStoreValue(database, item.imageStoreKey))
        if (!imageSrc) {
          continue
        }
        restored.push({
          id: item.id,
          created: item.created,
          prompt: item.prompt,
          revisedPrompt: item.revisedPrompt,
          model: item.model,
          size: item.size,
          keyName: item.keyName,
          imageSrc,
          thumbnailSrc: item.thumbnailSrc
        })
      }
      return mergeGenerations(restored, [])
    }

    return normalizePersistedGenerations(persisted)
  } finally {
    database.close()
  }
}

async function writePersistedGenerations(ownerKey: string, items: StudioSessionGeneration[]): Promise<void> {
  const database = await openImageStudioDatabase()
  if (!database) {
    return
  }

  try {
    const previous = await readStoreValue(database, ownerKey)
    const previousImageKeys = collectImageStoreKeys(previous)
    await writeStoreValues(database, ownerKey, previousImageKeys, items)
  } finally {
    database.close()
  }
}

function readStoreValue(database: IDBDatabase, key: string): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(IMAGE_STUDIO_STORE_NAME, 'readonly')
    const store = transaction.objectStore(IMAGE_STUDIO_STORE_NAME)
    const request = store.get(key)

    request.onsuccess = () => {
      resolve(request.result)
    }

    request.onerror = () => {
      reject(request.error ?? new Error('Failed to read stored image session'))
    }

    transaction.onabort = () => {
      reject(transaction.error ?? new Error('Failed to load stored image session'))
    }
  })
}

function writeStoreValues(
  database: IDBDatabase,
  ownerKey: string,
  previousImageKeys: string[],
  items: StudioSessionGeneration[]
): Promise<void> {
  const metadataItems = items.map((item) => cardToStoredMetadata(ownerKey, item))
  const nextImageKeys = new Set(metadataItems.map((item) => item.imageStoreKey))
  const imagePayloads = items.map((item, index) => ({
    key: metadataItems[index].imageStoreKey,
    value: {
      version: IMAGE_STUDIO_IMAGE_RECORD_VERSION,
      image: dataURLToPersistedImage(item.imageSrc)
    } satisfies PersistedStudioImagePayload
  }))

  return new Promise((resolve, reject) => {
    const transaction = database.transaction(IMAGE_STUDIO_STORE_NAME, 'readwrite')
    const store = transaction.objectStore(IMAGE_STUDIO_STORE_NAME)

    if (items.length === 0) {
      store.delete(ownerKey)
    } else {
      store.put(
        {
          version: IMAGE_STUDIO_RECORD_VERSION,
          items: metadataItems
        } satisfies PersistedStudioSessionRecord,
        ownerKey
      )
    }

    for (const payload of imagePayloads) {
      store.put(payload.value, payload.key)
    }

    for (const key of previousImageKeys) {
      if (!nextImageKeys.has(key)) {
        store.delete(key)
      }
    }

    transaction.oncomplete = () => {
      resolve()
    }

    transaction.onabort = () => {
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
    persistQueue = persistQueue.catch(() => undefined).then(async () => {
      const limits = Array.from(
        new Set(PERSIST_GENERATION_LIMITS.map((limit) => Math.min(limit, snapshot.length)).filter((limit) => limit >= 0))
      )

      let lastError: unknown
      for (const limit of limits) {
        try {
          await writePersistedGenerations(ownerKey, snapshot.slice(0, limit))
          return
        } catch (error) {
          lastError = error
        }
      }

      console.error('Failed to persist image studio session:', lastError)
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
