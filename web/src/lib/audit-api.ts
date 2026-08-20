export type AuditAction = 'create' | 'update' | 'status_change' | 'assign' | 'revoke' | 'view_key' | 'archive'

export type AuditEntityType = 'user' | 'department' | 'software_product' | 'license' | 'device' | 'license_assignment'

export interface AuditLogItem {
  id: string
  actor_id?: string
  actor_name?: string
  actor_email?: string
  action: AuditAction | string
  entity_type: AuditEntityType | string
  entity_id?: string
  metadata: Record<string, unknown>
  ip_address?: string
  created_at: string
}

interface AuditLogResult {
  items: AuditLogItem[]
  total: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class AuditAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'AuditAPIError'
    this.status = status
  }
}

export async function getAuditLogs(accessToken: string, limit = 200): Promise<AuditLogResult> {
  let response: Response
  try {
    const query = new URLSearchParams({ limit: String(limit) })
    response = await fetch(`${API_BASE_URL}/audit-logs?${query}`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
  } catch {
    throw new AuditAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể tải nhật ký hoạt động.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new AuditAPIError(message, response.status)
  }

  return (await response.json()) as AuditLogResult
}
