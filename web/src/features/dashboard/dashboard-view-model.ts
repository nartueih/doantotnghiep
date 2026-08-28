import type { LicenseAlert } from '../../lib/dashboard-api'

export function criticalLicenseAlerts(alerts: LicenseAlert[]): LicenseAlert[] {
  return alerts.filter((item) => item.severity === 'critical')
}
