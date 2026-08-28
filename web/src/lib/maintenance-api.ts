export type MaintenanceCategory = 'hardware' | 'software' | 'network' | 'accessory' | 'other'
export type MaintenancePriority = 'normal' | 'high' | 'urgent'
export type MaintenanceStatus = 'pending' | 'in_progress' | 'completed' | 'rejected' | 'cancelled'

export interface MaintenanceRequestItem {
  id: string
  requester_id: string
  requester_name: string
  device_id: string
  device_asset_code: string
  device_serial_number?: string
  device_name: string
  device_type: string
  device_manufacturer?: string
  device_model?: string
  device_purchased_at?: string
  device_warranty_expires_at?: string
  category: MaintenanceCategory
  priority: MaintenancePriority
  title: string
  description: string
  status: MaintenanceStatus
  assigned_to?: string
  assigned_to_name?: string
  last_actor_id: string
  last_actor_name: string
  response_note?: string
  created_at: string
  updated_at: string
  accepted_at?: string
  completed_at?: string
  rejected_at?: string
  cancelled_at?: string
}

export interface MaintenanceCreateInput {
  device_id: string
  category: MaintenanceCategory
  priority: MaintenancePriority
  title: string
  description: string
}

export interface MaintenanceListResult {
  items: MaintenanceRequestItem[]
  total: number
  open_count?: number
}

export interface MaintenanceFilters {
  status?: MaintenanceStatus | ''
  priority?: MaintenancePriority | ''
  category?: MaintenanceCategory | ''
  search?: string
}

interface APIErrorPayload { error?: string; code?: string }

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class MaintenanceAPIError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'MaintenanceAPIError'
    this.status = status
    this.code = code
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
    throw new MaintenanceAPIError('Không thể kết nối tới backend.', 0)
  }
  if (!response.ok) {
    let message = 'Không thể xử lý yêu cầu bảo trì.'
    let code: string | undefined
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
      code = payload.code
    } catch {
      // Keep fallback for non-JSON responses.
    }
    throw new MaintenanceAPIError(message, response.status, code)
  }
  return (await response.json()) as T
}

export function listMyMaintenanceRequests(accessToken: string): Promise<MaintenanceListResult> {
  return request('/me/maintenance-requests', accessToken)
}

export function createMaintenanceRequest(accessToken: string, input: MaintenanceCreateInput): Promise<MaintenanceRequestItem> {
  return request('/me/maintenance-requests', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function cancelMaintenanceRequest(accessToken: string, requestID: string): Promise<MaintenanceRequestItem> {
  return request(`/me/maintenance-requests/${requestID}/cancel`, accessToken, { method: 'POST' })
}

export function listMaintenanceRequests(accessToken: string, filters: MaintenanceFilters = {}): Promise<MaintenanceListResult> {
  const query = new URLSearchParams()
  if (filters.status) query.set('status', filters.status)
  if (filters.priority) query.set('priority', filters.priority)
  if (filters.category) query.set('category', filters.category)
  if (filters.search?.trim()) query.set('search', filters.search.trim())
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  return request(`/maintenance-requests${suffix}`, accessToken)
}

export function acceptMaintenanceRequest(accessToken: string, requestID: string): Promise<MaintenanceRequestItem> {
  return request(`/maintenance-requests/${requestID}/accept`, accessToken, { method: 'POST' })
}

export function completeMaintenanceRequest(accessToken: string, requestID: string, responseNote: string): Promise<MaintenanceRequestItem> {
  return request(`/maintenance-requests/${requestID}/complete`, accessToken, { method: 'POST', body: JSON.stringify({ response_note: responseNote }) })
}

export function rejectMaintenanceRequest(accessToken: string, requestID: string, responseNote: string): Promise<MaintenanceRequestItem> {
  return request(`/maintenance-requests/${requestID}/reject`, accessToken, { method: 'POST', body: JSON.stringify({ response_note: responseNote }) })
}
