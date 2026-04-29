import { apiClient } from './client'

export const IMAGE_GENERATION_ENDPOINT = '/studio/images/generations'

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

export interface GenerateImageRequest extends ImageGenerationPayload {
  groupId: number
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
