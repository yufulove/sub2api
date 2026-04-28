export const IMAGE_GENERATION_ENDPOINT = '/v1/images/generations'

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
  apiKey: string
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
  const response = await fetch(IMAGE_GENERATION_ENDPOINT, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${request.apiKey}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model: request.model,
      prompt: request.prompt,
      size: request.size,
      response_format: request.response_format ?? 'b64_json'
    }),
    signal: request.signal
  })

  const rawText = await response.text()
  let payload: unknown = null
  if (rawText) {
    try {
      payload = JSON.parse(rawText) as unknown
    } catch {
      payload = null
    }
  }

  if (!response.ok) {
    throw new Error(extractErrorMessage(payload, 'Image generation request failed'))
  }

  return payload as ImageGenerationResult
}
