export interface CostByCurrency {
  currency: string
  amount: number
}

export interface DashboardSummary {
  total_devices: number
  devices_by_status: Record<string, number>
  total_software_products: number
  total_licenses: number
  total_seats: number
  used_seats: number
  available_seats: number
  expired_licenses: number
  expiring_in_30_days: number
  expiring_in_60_days: number
  expiring_in_90_days: number
  exhausted_licenses: number
  high_usage_licenses: number
  costs_by_currency: CostByCurrency[]
  generated_at: string
}

export type AlertSeverity = 'critical' | 'warning' | 'info'

export interface LicenseAlert {
  license_id: string
  license_name: string
  license_type: string
  expires_at?: string
  days_until_expiry?: number
  seat_count: number
  used_seats: number
  available_seats: number
  utilization_percent: number
  severity: AlertSeverity
  alert_types: string[]
}

export interface LicenseAlertsResult {
  items: LicenseAlert[]
  total: number
  expiry_window_days: number
}

interface APIErrorPayload {
  error?: string
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export class DashboardAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'DashboardAPIError'
    this.status = status
  }
}

async function authorizedGet<T>(path: string, accessToken: string): Promise<T> {
  let response: Response

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
  } catch {
    throw new DashboardAPIError('Không thể kết nối tới backend.', 0)
  }

  if (!response.ok) {
    let message = 'Không thể tải dữ liệu dashboard.'
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error ?? message
    } catch {
      // Keep the default message if the response is not JSON.
    }
    throw new DashboardAPIError(message, response.status)
  }

  return (await response.json()) as T
}

export function getDashboardSummary(accessToken: string): Promise<DashboardSummary> {
  return authorizedGet('/dashboard/summary', accessToken)
}

export function getLicenseAlerts(
  accessToken: string,
  days: 30 | 60 | 90,
): Promise<LicenseAlertsResult> {
  return authorizedGet(`/dashboard/license-alerts?days=${days}`, accessToken)
}
