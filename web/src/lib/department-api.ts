export interface DepartmentItem {
  id: string
  name: string
  code: string
  created_at: string
  updated_at: string
}

export interface DepartmentInput {
  name: string
  code: string
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class DepartmentAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'DepartmentAPIError'
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
    throw new DepartmentAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể xử lý dữ liệu phòng ban.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new DepartmentAPIError(message, response.status)
  }

  return (await response.json()) as T
}

export function getDepartments(accessToken: string): Promise<ListResult<DepartmentItem>> {
  return request('/departments', accessToken)
}

export function createDepartment(accessToken: string, input: DepartmentInput): Promise<DepartmentItem> {
  return request('/departments', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function updateDepartment(accessToken: string, departmentID: string, input: DepartmentInput): Promise<DepartmentItem> {
  return request(`/departments/${departmentID}`, accessToken, { method: 'PUT', body: JSON.stringify(input) })
}
