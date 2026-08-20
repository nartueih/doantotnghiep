export type LicenseType = 'subscription' | 'perpetual'
export type AssignmentType = 'user' | 'device' | 'mixed'
export type LicenseLifecycle = 'active' | 'expired' | 'upcoming' | 'archived'

export { getSoftwareProducts } from './software-api'
export type { SoftwareProduct } from './software-api'

export interface LicenseItem {
  id: string
  software_product_id: string
  name: string
  license_type: LicenseType
  assignment_type: AssignmentType
  seat_count: number
  used_seats: number
  available_seats: number
  key_hint?: string
  allow_employee_key_view: boolean
  vendor: string
  purchased_at?: string
  starts_at?: string
  expires_at?: string
  cost: number
  currency?: string
  notes?: string
  lifecycle_status: LicenseLifecycle
  created_at: string
  updated_at: string
  archived_at?: string
}

export interface LicenseInput {
  software_product_id: string
  name: string
  license_type: LicenseType
  assignment_type: AssignmentType
  seat_count: number
  license_key: string
  allow_employee_key_view: boolean
  vendor: string
  purchased_at: string
  starts_at: string
  expires_at: string
  cost: number
  currency: string
  notes: string
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload { error?: string }

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class LicenseAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'LicenseAPIError'
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
    throw new LicenseAPIError('Không thể kết nối tới backend.', 0)
  }
  if (!response.ok) {
    let message = 'Không thể tải dữ liệu license.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback message for non-JSON responses.
    }
    throw new LicenseAPIError(message, response.status)
  }
  return (await response.json()) as T
}

export function getLicenses(accessToken: string): Promise<ListResult<LicenseItem>> {
  return request('/licenses', accessToken)
}

export async function revealLicenseKey(accessToken: string, licenseID: string): Promise<string> {
  const result = await request<{ license_key: string }>(`/licenses/${licenseID}/key`, accessToken)
  return result.license_key
}

export function createLicense(accessToken: string, input: LicenseInput): Promise<LicenseItem> {
  return request('/licenses', accessToken, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateLicense(accessToken: string, licenseID: string, input: LicenseInput): Promise<LicenseItem> {
  return request(`/licenses/${licenseID}`, accessToken, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function archiveLicense(accessToken: string, licenseID: string): Promise<LicenseItem> {
  return request(`/licenses/${licenseID}/archive`, accessToken, { method: 'PATCH' })
}
