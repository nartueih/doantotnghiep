import type { LicenseItem } from '../../lib/license-api.ts'
import type {
  LicenseRequestDecisionReason,
  LicenseRequestPriority,
  LicenseRequestStatus,
} from '../../lib/license-request-api.ts'

const statusLabels: Record<LicenseRequestStatus, string> = {
  pending: 'Đang chờ',
  approved: 'Đã duyệt',
  rejected: 'Đã từ chối',
  cancelled: 'Đã hủy',
}

const priorityLabels: Record<LicenseRequestPriority, string> = {
  normal: 'Bình thường',
  high: 'Cao',
  urgent: 'Khẩn cấp',
}

const rejectionLabels: Record<LicenseRequestDecisionReason, string> = {
  out_of_stock: 'Tạm hết license',
  not_approved: 'Không được phê duyệt',
  other: 'Lý do khác',
}

export function requestStatusLabel(status: LicenseRequestStatus) {
  return statusLabels[status]
}

export function requestPriorityLabel(priority: LicenseRequestPriority) {
  return priorityLabels[priority]
}

export function rejectReasonLabel(reason: LicenseRequestDecisionReason) {
  return rejectionLabels[reason]
}

export function eligibleLicenses(items: LicenseItem[], softwareProductID: string) {
  return items.filter((item) =>
    item.software_product_id === softwareProductID
    && item.lifecycle_status === 'active'
    && item.available_seats > 0
    && (item.assignment_type === 'user' || item.assignment_type === 'mixed'))
}
