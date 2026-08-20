import type { AuthUser } from './auth-api'

export type DeviceStatus = 'available' | 'assigned' | 'maintenance' | 'retired' | 'lost'

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
  status: DeviceStatus
  purchased_at?: string
  warranty_expires_at?: string
  created_at: string
  updated_at: string
}

export interface DeviceInput {
  asset_code: string
  serial_number: string
  name: string
  device_type: string
  manufacturer: string
  model: string
  purchased_at: string
  warranty_expires_at: string
}

export interface DeviceLicenseAssignment {
  id: string
  license_id: string
  license_name: string
  device_id?: string
  status: 'active' | 'revoked'
}

interface ListResult<T> { items: T[]; total: number }
interface APIErrorPayload { error?: string }

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class DeviceAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'DeviceAPIError'
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
    throw new DeviceAPIError('Không thể kết nối tới backend.', 0)
  }
  if (!response.ok) {
    let message = 'Không thể xử lý dữ liệu thiết bị.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new DeviceAPIError(message, response.status)
  }
  return (await response.json()) as T
}

export function getDevices(accessToken: string): Promise<ListResult<DeviceItem>> {
  return request('/devices', accessToken)
}

export function getDeviceUsers(accessToken: string): Promise<ListResult<AuthUser>> {
  return request('/users', accessToken)
}

export function getDeviceLicenseAssignments(accessToken: string): Promise<ListResult<DeviceLicenseAssignment>> {
  return request('/license-assignments', accessToken)
}

export function createDevice(accessToken: string, input: DeviceInput): Promise<DeviceItem> {
  return request('/devices', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function updateDevice(accessToken: string, deviceID: string, input: DeviceInput): Promise<DeviceItem> {
  return request(`/devices/${deviceID}`, accessToken, { method: 'PUT', body: JSON.stringify(input) })
}

export function updateDeviceStatus(accessToken: string, deviceID: string, status: Exclude<DeviceStatus, 'assigned'>): Promise<DeviceItem> {
  return request(`/devices/${deviceID}/status`, accessToken, { method: 'PATCH', body: JSON.stringify({ status }) })
}

export function assignDevice(accessToken: string, deviceID: string, userID: string): Promise<DeviceItem> {
  return request(`/devices/${deviceID}/assignment`, accessToken, { method: 'PATCH', body: JSON.stringify({ user_id: userID }) })
}
