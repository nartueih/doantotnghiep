import assert from 'node:assert/strict'
import test from 'node:test'
import type { LicenseAlert } from '../../lib/dashboard-api.ts'
import { criticalLicenseAlerts } from './dashboard-view-model.ts'

const baseAlert: LicenseAlert = {
  license_id: 'license-1',
  license_name: 'Adobe Business',
  license_type: 'subscription',
  seat_count: 10,
  used_seats: 10,
  available_seats: 0,
  utilization_percent: 100,
  severity: 'critical',
  alert_types: ['exhausted'],
}

test('criticalLicenseAlerts keeps only critical alerts in API order', () => {
  const result = criticalLicenseAlerts([
    baseAlert,
    { ...baseAlert, license_id: 'warning', severity: 'warning' },
    { ...baseAlert, license_id: 'critical-2', license_name: 'Office Business' },
  ])

  assert.deepEqual(result.map((item) => item.license_id), ['license-1', 'critical-2'])
})
