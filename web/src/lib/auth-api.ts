export type UserRole = 'admin' | 'it_manager' | 'employee'

export interface AuthUser {
  id: string
  email: string
  full_name: string
  employee_code: string
  department_id?: string
  department_name?: string
  role: UserRole
  status: 'active' | 'locked'
  created_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface AuthSession {
  tokens: TokenPair
  user: AuthUser
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class APIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'APIError'
    this.status = status
  }
}

async function readError(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as APIErrorPayload
    return payload.error ?? 'Yêu cầu không thành công.'
  } catch {
    return 'Máy chủ trả về dữ liệu không hợp lệ.'
  }
}

export async function login(email: string, password: string): Promise<AuthSession> {
  let response: Response

  try {
    response = await fetch(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
  } catch {
    throw new APIError('Không thể kết nối tới backend. Hãy kiểm tra API đang chạy ở cổng 8081.', 0)
  }

  if (!response.ok) {
    throw new APIError(await readError(response), response.status)
  }

  return (await response.json()) as AuthSession
}

export async function logout(refreshToken: string): Promise<void> {
  try {
    await fetch(`${API_BASE_URL}/auth/logout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
  } catch {
    // The local session is still cleared when the API is temporarily unavailable.
  }
}
