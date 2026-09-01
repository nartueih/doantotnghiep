import assert from 'node:assert/strict'
import test from 'node:test'
import {
  countAdminRequestBadges,
  formatAdminRequestBadge,
} from './admin-request-badges.ts'

test('countAdminRequestBadges counts only requests waiting for first handling', () => {
  const badges = countAdminRequestBadges(
    ['pending', 'approved', 'pending', 'rejected', 'cancelled'],
    ['pending', 'in_progress', 'completed', 'pending', 'rejected', 'cancelled'],
  )

  assert.deepEqual(badges, {
    licenseRequests: 2,
    maintenanceRequests: 2,
  })
})

test('formatAdminRequestBadge caps large counts for the compact sidebar', () => {
  assert.equal(formatAdminRequestBadge(8), '8')
  assert.equal(formatAdminRequestBadge(100), '99+')
})
