import assert from 'node:assert/strict'
import test from 'node:test'
import {
  availableMaintenanceActions,
  hasOpenMaintenance,
  maintenanceCategoryLabel,
  maintenanceStatusLabel,
} from './maintenance-view-model.ts'
import type { MaintenanceRequestItem } from '../../lib/maintenance-api.ts'

test('maintenance labels are Vietnamese and deterministic', () => {
  assert.equal(maintenanceStatusLabel('pending'), 'Chờ tiếp nhận')
  assert.equal(maintenanceStatusLabel('in_progress'), 'Đang xử lý')
  assert.equal(maintenanceCategoryLabel('hardware'), 'Phần cứng')
  assert.equal(maintenanceCategoryLabel('other'), 'Khác')
})

test('availableMaintenanceActions follows the request state machine', () => {
  assert.deepEqual(availableMaintenanceActions('pending'), ['accept', 'reject'])
  assert.deepEqual(availableMaintenanceActions('in_progress'), ['complete', 'reject'])
  assert.deepEqual(availableMaintenanceActions('completed'), [])
  assert.deepEqual(availableMaintenanceActions('cancelled'), [])
})

test('hasOpenMaintenance matches only pending or in-progress requests for the device', () => {
  const requests = [
    { id: 'one', device_id: 'device-1', status: 'completed' },
    { id: 'two', device_id: 'device-2', status: 'in_progress' },
  ] as MaintenanceRequestItem[]
  assert.equal(hasOpenMaintenance(requests, 'device-1'), false)
  assert.equal(hasOpenMaintenance(requests, 'device-2'), true)
})
