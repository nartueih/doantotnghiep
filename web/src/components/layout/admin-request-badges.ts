import type { LicenseRequestStatus } from '../../lib/license-request-api'
import type { MaintenanceStatus } from '../../lib/maintenance-api'

export interface AdminRequestBadges {
  licenseRequests: number
  maintenanceRequests: number
}

export function countAdminRequestBadges(
  licenseStatuses: LicenseRequestStatus[],
  maintenanceStatuses: MaintenanceStatus[],
): AdminRequestBadges {
  return {
    licenseRequests: licenseStatuses.filter((status) => status === 'pending').length,
    maintenanceRequests: maintenanceStatuses.filter((status) => status === 'pending').length,
  }
}

export function formatAdminRequestBadge(count: number): string {
  return count > 99 ? '99+' : String(count)
}
