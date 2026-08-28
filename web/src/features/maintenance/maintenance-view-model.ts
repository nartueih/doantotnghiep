import type {
  MaintenanceCategory,
  MaintenanceRequestItem,
  MaintenanceStatus,
} from '../../lib/maintenance-api.ts'

const statusLabels: Record<MaintenanceStatus, string> = {
  pending: 'Chờ tiếp nhận',
  in_progress: 'Đang xử lý',
  completed: 'Hoàn thành',
  rejected: 'Đã từ chối',
  cancelled: 'Đã hủy',
}

const categoryLabels: Record<MaintenanceCategory, string> = {
  hardware: 'Phần cứng',
  software: 'Phần mềm',
  network: 'Mạng',
  accessory: 'Phụ kiện',
  other: 'Khác',
}

export type MaintenanceAction = 'accept' | 'complete' | 'reject'

export function maintenanceStatusLabel(status: MaintenanceStatus) {
  return statusLabels[status]
}

export function maintenanceCategoryLabel(category: MaintenanceCategory) {
  return categoryLabels[category]
}

export function availableMaintenanceActions(status: MaintenanceStatus): MaintenanceAction[] {
  if (status === 'pending') return ['accept', 'reject']
  if (status === 'in_progress') return ['complete', 'reject']
  return []
}

export function hasOpenMaintenance(items: MaintenanceRequestItem[], deviceID: string) {
  return items.some((item) => item.device_id === deviceID && (item.status === 'pending' || item.status === 'in_progress'))
}
