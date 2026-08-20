export interface SoftwareProduct {
  id: string
  name: string
  publisher: string
  version: string
  description: string
  created_at: string
  updated_at: string
}

export interface SoftwareInput {
  name: string
  publisher: string
  version: string
  description: string
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class SoftwareAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'SoftwareAPIError'
    this.status = status
  }
}

async function request<T>(path: string, accessToken: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      headers: {
        Authorization: `Bearer ${accessToken}`,
        ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
        ...init?.headers,
      },
    })
  } catch {
    throw new SoftwareAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể xử lý dữ liệu phần mềm.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new SoftwareAPIError(message, response.status)
  }

  return (await response.json()) as T
}

export function getSoftwareProducts(accessToken: string): Promise<ListResult<SoftwareProduct>> {
  return request('/software', accessToken)
}

export function createSoftwareProduct(accessToken: string, input: SoftwareInput): Promise<SoftwareProduct> {
  return request('/software', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function updateSoftwareProduct(accessToken: string, productID: string, input: SoftwareInput): Promise<SoftwareProduct> {
  return request(`/software/${productID}`, accessToken, { method: 'PUT', body: JSON.stringify(input) })
}
