import assert from 'node:assert/strict'
import test from 'node:test'
import type { LicenseItem } from '../../lib/license-api.ts'
import { isNoAvailableSeatsConflict, normalizeAPIError } from '../../lib/api-error.ts'
import {
  eligibleLicenses,
  rejectReasonLabel,
  requestPriorityLabel,
  requestStatusLabel,
} from './request-view-model.ts'

const baseLicense: LicenseItem = {
  id: 'eligible-license',
  software_product_id: 'software-1',
  name: 'Adobe Business',
  license_type: 'subscription',
  assignment_type: 'user',
  seat_count: 5,
  used_seats: 2,
  available_seats: 3,
  allow_employee_key_view: false,
  vendor: 'Adobe',
  cost: 0,
  lifecycle_status: 'active',
  created_at: '2026-08-23T08:00:00Z',
  updated_at: '2026-08-23T08:00:00Z',
}

test('eligibleLicenses keeps active matching user licenses with available seats', () => {
  const licenses: LicenseItem[] = [
    baseLicense,
    { ...baseLicense, id: 'mixed-license', assignment_type: 'mixed' },
    { ...baseLicense, id: 'wrong-software', software_product_id: 'software-2' },
    { ...baseLicense, id: 'device-only', assignment_type: 'device' },
    { ...baseLicense, id: 'exhausted', available_seats: 0, used_seats: 5 },
    { ...baseLicense, id: 'expired', lifecycle_status: 'expired' },
  ]

  assert.deepEqual(eligibleLicenses(licenses, 'software-1').map((item) => item.id), [
    'eligible-license',
    'mixed-license',
  ])
})

test('request labels are Vietnamese and deterministic', () => {
  assert.equal(requestStatusLabel('pending'), 'Đang chờ')
  assert.equal(requestStatusLabel('approved'), 'Đã duyệt')
  assert.equal(requestStatusLabel('rejected'), 'Đã từ chối')
  assert.equal(requestStatusLabel('cancelled'), 'Đã hủy')
  assert.equal(requestPriorityLabel('normal'), 'Bình thường')
  assert.equal(requestPriorityLabel('high'), 'Cao')
  assert.equal(requestPriorityLabel('urgent'), 'Khẩn cấp')
  assert.equal(rejectReasonLabel('out_of_stock'), 'Tạm hết license')
  assert.equal(rejectReasonLabel('not_approved'), 'Không được phê duyệt')
  assert.equal(rejectReasonLabel('other'), 'Lý do khác')
})

test('normalizeAPIError preserves status and machine-readable code across API clients', () => {
  const foreignAPIError = Object.assign(new Error('Seat unavailable'), {
    status: 409,
    code: 'no_available_seats',
  })

  assert.deepEqual(normalizeAPIError(foreignAPIError, 'Fallback'), {
    message: 'Seat unavailable',
    status: 409,
    code: 'no_available_seats',
  })
  assert.deepEqual(normalizeAPIError(null, 'Fallback'), {
    message: 'Fallback',
    status: 0,
  })
})

test('only the no-available-seats conflict receives exhausted-seat guidance', () => {
  assert.equal(isNoAvailableSeatsConflict({ message: 'No seats', status: 409, code: 'no_available_seats' }), true)
  assert.equal(isNoAvailableSeatsConflict({ message: 'Already handled', status: 409, code: 'request_not_pending' }), false)
  assert.equal(isNoAvailableSeatsConflict({ message: 'No seats', status: 422, code: 'no_available_seats' }), false)
})
