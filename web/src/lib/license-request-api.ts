import type { SoftwareProduct } from './software-api'

export type LicenseRequestPriority = 'normal' | 'high' | 'urgent'
export type LicenseRequestStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'
export type LicenseRequestDecisionReason = 'out_of_stock' | 'not_approved' | 'other'
export type NotificationType = 'license_request_approved' | 'license_request_rejected'

export interface LicenseRequestItem {
  id: string
  requester_id: string
  requester_name: string
  software_product_id: string
  software_product_name: string
  priority: LicenseRequestPriority
  reason: string
  status: LicenseRequestStatus
  selected_license_id?: string
  selected_license_name?: string
  assignment_id?: string
  reviewed_by?: string
  reviewed_by_name?: string
  decision_reason?: LicenseRequestDecisionReason
  response_note?: string
  created_at: string
  updated_at: string
  reviewed_at?: string
  cancelled_at?: string
}

export interface LicenseRequestCreateInput {
  software_product_id: string
  priority: LicenseRequestPriority
  reason: string
}

export interface LicenseRequestApproveInput {
  license_id: string
  response_note: string
}

export interface LicenseRequestRejectInput {
  decision_reason: LicenseRequestDecisionReason
  response_note: string
}

export interface WebsiteNotification {
  id: string
  user_id: string
  type: NotificationType
  title: string
  message: string
  entity_type: 'license_request'
  entity_id: string
  created_at: string
  read_at?: string
}

export interface ListResult<T> {
  items: T[]
  total: number
}

export interface NotificationListResult extends ListResult<WebsiteNotification> {
  unread_count: number
}

export interface LicenseRequestFilters {
  status?: LicenseRequestStatus | ''
  priority?: LicenseRequestPriority | ''
  search?: string
}

interface APIErrorPayload {
  error?: string
  code?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class LicenseRequestAPIError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'LicenseRequestAPIError'
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
    throw new LicenseRequestAPIError('Không thể kết nối tới backend.', 0)
  }
  if (!response.ok) {
    let message = 'Không thể xử lý yêu cầu license.'
    let code: string | undefined
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
      code = payload.code
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new LicenseRequestAPIError(message, response.status, code)
  }
  return (await response.json()) as T
}

export function listRequestableSoftware(accessToken: string): Promise<ListResult<SoftwareProduct>> {
  return request('/me/requestable-software', accessToken)
}

export function listMyLicenseRequests(accessToken: string): Promise<ListResult<LicenseRequestItem>> {
  return request('/me/license-requests', accessToken)
}

export function createLicenseRequest(accessToken: string, input: LicenseRequestCreateInput): Promise<LicenseRequestItem> {
  return request('/me/license-requests', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function cancelLicenseRequest(accessToken: string, requestID: string): Promise<LicenseRequestItem> {
  return request(`/me/license-requests/${requestID}/cancel`, accessToken, { method: 'PATCH' })
}

export function listNotifications(accessToken: string): Promise<NotificationListResult> {
  return request('/me/notifications', accessToken)
}

export function markNotificationRead(accessToken: string, notificationID: string): Promise<WebsiteNotification> {
  return request(`/me/notifications/${notificationID}/read`, accessToken, { method: 'PATCH' })
}

export function markAllNotificationsRead(accessToken: string): Promise<{ updated: number }> {
  return request('/me/notifications/read-all', accessToken, { method: 'PATCH' })
}

export function listLicenseRequests(accessToken: string, filters: LicenseRequestFilters = {}): Promise<ListResult<LicenseRequestItem>> {
  const query = new URLSearchParams()
  if (filters.status) query.set('status', filters.status)
  if (filters.priority) query.set('priority', filters.priority)
  if (filters.search?.trim()) query.set('search', filters.search.trim())
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  return request(`/license-requests${suffix}`, accessToken)
}

export function approveLicenseRequest(accessToken: string, requestID: string, input: LicenseRequestApproveInput): Promise<LicenseRequestItem> {
  return request(`/license-requests/${requestID}/approve`, accessToken, { method: 'PATCH', body: JSON.stringify(input) })
}

export function rejectLicenseRequest(accessToken: string, requestID: string, input: LicenseRequestRejectInput): Promise<LicenseRequestItem> {
  return request(`/license-requests/${requestID}/reject`, accessToken, { method: 'PATCH', body: JSON.stringify(input) })
}
