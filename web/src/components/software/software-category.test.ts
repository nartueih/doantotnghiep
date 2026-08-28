import assert from 'node:assert/strict'
import test from 'node:test'
import { softwareCategory } from './software-category.ts'

test('softwareCategory groups known products into neutral software categories', () => {
  assert.deepEqual(softwareCategory('Adobe Photoshop', 'Adobe'), { key: 'design', label: 'Thiết kế' })
  assert.deepEqual(softwareCategory('Figma Organization', 'Figma'), { key: 'design', label: 'Thiết kế' })
  assert.deepEqual(softwareCategory('Microsoft 365 Apps', 'Microsoft'), { key: 'office', label: 'Văn phòng' })
  assert.deepEqual(softwareCategory('Windows 11 Enterprise', 'Microsoft'), { key: 'operating-system', label: 'Hệ điều hành' })
  assert.deepEqual(softwareCategory('JetBrains All Products Pack', 'JetBrains'), { key: 'development', label: 'Phát triển' })
  assert.deepEqual(softwareCategory('Zoom Workplace Business', 'Zoom'), { key: 'collaboration', label: 'Cộng tác' })
})

test('softwareCategory uses a neutral application category for unknown software', () => {
  assert.deepEqual(softwareCategory('Notion'), { key: 'general', label: 'Ứng dụng' })
})
