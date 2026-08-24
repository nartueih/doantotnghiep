export type SoftwareCategoryKey = 'office' | 'design' | 'development' | 'collaboration' | 'operating-system' | 'general'

export interface SoftwareCategory {
  key: SoftwareCategoryKey
  label: string
}

export function softwareCategory(name: string, publisher = ''): SoftwareCategory {
  const source = `${name} ${publisher}`.trim().toLocaleLowerCase('en')
  if (includesAny(source, ['windows', 'macos', 'ubuntu', 'linux'])) return { key: 'operating-system', label: 'Hệ điều hành' }
  if (includesAny(source, ['zoom', 'google meet', 'microsoft teams', 'webex', 'slack'])) return { key: 'collaboration', label: 'Cộng tác' }
  if (includesAny(source, ['jetbrains', 'visual studio', 'github', 'gitlab', 'intellij'])) return { key: 'development', label: 'Phát triển' }
  if (includesAny(source, ['adobe', 'photoshop', 'illustrator', 'figma', 'canva'])) return { key: 'design', label: 'Thiết kế' }
  if (includesAny(source, ['microsoft 365', 'office 365', 'm365', 'google workspace', 'libreoffice'])) return { key: 'office', label: 'Văn phòng' }
  return { key: 'general', label: 'Ứng dụng' }
}

function includesAny(source: string, keywords: string[]): boolean {
  return keywords.some((keyword) => source.includes(keyword))
}
