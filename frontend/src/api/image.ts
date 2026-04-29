import { apiClient } from './client'

export const IMAGE_GENERATION_ENDPOINT = '/studio/images/generations'
export const IMAGE_EDIT_ENDPOINT = '/studio/images/edits'
export const IMAGE_HISTORY_ENDPOINT = '/studio/images/history'

export interface ImageGenerationPayload {
  model: string
  prompt: string
  size?: string
  response_format?: 'b64_json'
}

export interface ImageGenerationResultItem {
  b64_json?: string
  revised_prompt?: string
}

export interface ImageGenerationResult {
  created: number
  data: ImageGenerationResultItem[]
}

export interface SaveImageHistoryItem {
  client_id?: string
  b64_json: string
  revised_prompt?: string
}

export interface SaveImageHistoryRequest {
  request_id: string
  group_id?: number
  model: string
  size: string
  prompt: string
  created: number
  images: SaveImageHistoryItem[]
}

export interface StudioImageHistoryAsset {
  id: number
  request_id: string
  client_id?: string
  image_index: number
  model: string
  size: string
  prompt: string
  revised_prompt?: string
  image_url: string
  content_type: string
  byte_size: number
  created_at: string
  group_id?: number
}

export interface StudioImageHistoryResponse {
  items: StudioImageHistoryAsset[]
}

export interface GenerateImageRequest extends ImageGenerationPayload {
  groupId: number
  signal?: AbortSignal
}

export interface EditImageRequest extends ImageGenerationPayload {
  groupId: number
  image: Blob
  signal?: AbortSignal
}

function extractErrorMessage(payload: unknown, fallback: string): string {
  if (!payload || typeof payload !== 'object') {
    return fallback
  }

  const data = payload as Record<string, unknown>
  const error = data.error
  if (error && typeof error === 'object') {
    const message = (error as Record<string, unknown>).message
    if (typeof message === 'string' && message.trim() !== '') {
      return message.trim()
    }
  }

  const message = data.message
  if (typeof message === 'string' && message.trim() !== '') {
    return message.trim()
  }

  const detail = data.detail
  if (typeof detail === 'string' && detail.trim() !== '') {
    return detail.trim()
  }

  return fallback
}

export async function generateImage(request: GenerateImageRequest): Promise<ImageGenerationResult> {
  try {
    const { data } = await apiClient.post<ImageGenerationResult>(
      IMAGE_GENERATION_ENDPOINT,
      {
        group_id: request.groupId,
        model: request.model,
        prompt: request.prompt,
        size: request.size,
        response_format: request.response_format ?? 'b64_json'
      },
      {
        signal: request.signal,
        timeout: 300000
      }
    )
    return data
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && (error as { code?: string }).code === 'ERR_CANCELED') {
      throw error
    }
    throw new Error(extractErrorMessage(error, 'Image generation request failed'))
  }
}

export async function editImage(request: EditImageRequest): Promise<ImageGenerationResult> {
  const formData = new FormData()
  formData.append('group_id', String(request.groupId))
  formData.append('model', request.model)
  formData.append('prompt', request.prompt)
  formData.append('response_format', request.response_format ?? 'b64_json')
  if (request.size) {
    formData.append('size', request.size)
  }
  formData.append('image', request.image, 'studio-reference.png')

  try {
    const { data } = await apiClient.post<ImageGenerationResult>(IMAGE_EDIT_ENDPOINT, formData, {
      signal: request.signal,
      timeout: 300000
    })
    return data
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && (error as { code?: string }).code === 'ERR_CANCELED') {
      throw error
    }
    throw new Error(extractErrorMessage(error, 'Image edit request failed'))
  }
}

export async function saveImageHistory(request: SaveImageHistoryRequest): Promise<StudioImageHistoryResponse> {
  try {
    const { data } = await apiClient.post<StudioImageHistoryResponse>(IMAGE_HISTORY_ENDPOINT, request, {
      timeout: 120000
    })
    return data
  } catch (error) {
    throw new Error(extractErrorMessage(error, 'Image history save failed'))
  }
}

export async function listImageHistory(limit = 60): Promise<StudioImageHistoryResponse> {
  try {
    const { data } = await apiClient.get<StudioImageHistoryResponse>(IMAGE_HISTORY_ENDPOINT, {
      params: { limit },
      timeout: 60000
    })
    return data
  } catch (error) {
    throw new Error(extractErrorMessage(error, 'Image history load failed'))
  }
}
