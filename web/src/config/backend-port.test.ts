import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    return entry.isDirectory() ? sourceFiles(path) : [path]
  })
}

test('development frontend consistently targets backend port 8080', () => {
  const legacyPort = String(8080 + 1)
  const legacyPortPattern = new RegExp(`\\b${legacyPort}\\b`)
  const checkedFiles = [
    '.env.example',
    'README.md',
    'vite.config.ts',
    ...sourceFiles('src').filter((path) => /\.(ts|tsx)$/.test(path)),
  ]

  for (const path of checkedFiles) {
    assert.doesNotMatch(
      readFileSync(path, 'utf8'),
      legacyPortPattern,
      `${path} still references legacy backend port ${legacyPort}`,
    )
  }

  assert.match(readFileSync('vite.config.ts', 'utf8'), /http:\/\/localhost:8080/)
  assert.match(readFileSync('.env.example', 'utf8'), /VITE_API_PROXY_TARGET=http:\/\/localhost:8080/)
})
