import type { DeviceItem } from './device-api'

export type AssignmentSource = 'user' | 'device'

export interface MyAssignedLicense {
  assignment_id: string
  license_id: string
  license_name: string
  software_product_id: string
  license_type: 'subscription' | 'perpetual' | string
  assignment_source: AssignmentSource | string
  device_id?: string
  device_asset_code?: string
  assigned_at: string
  expires_at?: string
  lifecycle_status: 'active' | 'upcoming' | 'expired' | string
  notes?: string
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class SelfServiceAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'SelfServiceAPIError'
    this.status = status
  }
}

async function request<T>(path: string, accessToken: string): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
  } catch {
    throw new SelfServiceAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể tải dữ liệu cá nhân.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new SelfServiceAPIError(message, response.status)
  }

  return (await response.json()) as T
}

export function getMyDevices(accessToken: string): Promise<ListResult<DeviceItem>> {
  return request('/me/devices', accessToken)
}

export function getMyLicenses(accessToken: string): Promise<ListResult<MyAssignedLicense>> {
  return request('/me/licenses', accessToken)
}
