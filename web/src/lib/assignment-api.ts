import type { AuthUser } from './auth-api'
import type { LicenseItem } from './license-api'

export type AssignmentStatus = 'active' | 'revoked'

export interface AssignmentItem {
  id: string
  license_id: string
  license_name: string
  user_id?: string
  device_id?: string
  target_name: string
  assigned_by: string
  assigned_by_name: string
  assigned_at: string
  revoked_at?: string
  revoked_by?: string
  revoked_by_name?: string
  status: AssignmentStatus
  notes?: string
}

export interface DeviceItem {
  id: string
  assigned_user_id?: string
  assigned_user_name?: string
  asset_code: string
  serial_number?: string
  name: string
  device_type: string
  manufacturer?: string
  model?: string
  status: 'available' | 'assigned' | 'maintenance' | 'retired' | 'lost'
}

export interface AssignmentInput {
  license_id: string
  user_id?: string
  device_id?: string
  notes: string
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload { error?: string }

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class AssignmentAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'AssignmentAPIError'
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
    throw new AssignmentAPIError('Không thể kết nối tới backend.', 0)
  }
  if (!response.ok) {
    let message = 'Không thể xử lý dữ liệu cấp phát.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new AssignmentAPIError(message, response.status)
  }
  return (await response.json()) as T
}

export function getAssignments(accessToken: string): Promise<ListResult<AssignmentItem>> {
  return request('/license-assignments', accessToken)
}

export function getAssignmentLicenses(accessToken: string): Promise<ListResult<LicenseItem>> {
  return request('/licenses', accessToken)
}

export function getAssignmentUsers(accessToken: string): Promise<ListResult<AuthUser>> {
  return request('/users', accessToken)
}

export function getAssignmentDevices(accessToken: string): Promise<ListResult<DeviceItem>> {
  return request('/devices', accessToken)
}

export function createAssignment(accessToken: string, input: AssignmentInput): Promise<AssignmentItem> {
  return request('/license-assignments', accessToken, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function revokeAssignment(accessToken: string, assignmentID: string): Promise<AssignmentItem> {
  return request(`/license-assignments/${assignmentID}/revoke`, accessToken, { method: 'PATCH' })
}
