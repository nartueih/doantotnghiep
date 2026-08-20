import type { AuthUser, UserRole } from './auth-api'

export type UserStatus = 'active' | 'locked'

export interface DepartmentItem {
  id: string
  name: string
  code: string
  created_at: string
  updated_at: string
}

export interface CreateUserInput {
  email: string
  password: string
  full_name: string
  employee_code: string
  department_id: string
  role: UserRole
}

interface ListResult<T> {
  items: T[]
  total: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class UserAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'UserAPIError'
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
    throw new UserAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể xử lý dữ liệu người dùng.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the fallback for non-JSON responses.
    }
    throw new UserAPIError(message, response.status)
  }

  return (await response.json()) as T
}

export function getUsers(accessToken: string): Promise<ListResult<AuthUser>> {
  return request('/users', accessToken)
}

export function getDepartments(accessToken: string): Promise<ListResult<DepartmentItem>> {
  return request('/departments', accessToken)
}

export function createUser(accessToken: string, input: CreateUserInput): Promise<AuthUser> {
  return request('/users', accessToken, { method: 'POST', body: JSON.stringify(input) })
}

export function updateUserStatus(accessToken: string, userID: string, status: UserStatus): Promise<AuthUser> {
  return request(`/users/${userID}/status`, accessToken, { method: 'PATCH', body: JSON.stringify({ status }) })
}
