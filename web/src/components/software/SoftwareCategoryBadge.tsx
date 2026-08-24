import { softwareCategory, type SoftwareCategoryKey } from './software-category'
import './SoftwareCategoryBadge.css'

interface SoftwareCategoryBadgeProps {
  name: string
  publisher?: string
  size?: 'compact' | 'default' | 'large'
  className?: string
}

export function SoftwareCategoryBadge({ name, publisher, size = 'default', className = '' }: SoftwareCategoryBadgeProps) {
  const category = softwareCategory(name, publisher)
  return <span className={`software-category-badge category-${category.key} category-size-${size} ${className}`.trim()} title={category.label} aria-hidden="true">
    <CategoryGlyph category={category.key} />
  </span>
}

function CategoryGlyph({ category }: { category: SoftwareCategoryKey }) {
  if (category === 'office') {
    return <svg viewBox="0 0 24 24"><rect x="4" y="4" width="6" height="6" rx="1" /><rect x="14" y="4" width="6" height="6" rx="1" /><rect x="4" y="14" width="6" height="6" rx="1" /><rect x="14" y="14" width="6" height="6" rx="1" /></svg>
  }
  if (category === 'design') {
    return <svg viewBox="0 0 24 24"><path d="m4 20 4.8-1.1L19 8.7a2.1 2.1 0 0 0-3-3L5.8 15.9 4 20Z" /><path d="m13.9 7.8 2.3 2.3M5.8 15.9l2.3 2.3" /></svg>
  }
  if (category === 'development') {
    return <svg viewBox="0 0 24 24"><path d="m8.5 7-5 5 5 5M15.5 7l5 5-5 5M14 4l-4 16" /></svg>
  }
  if (category === 'collaboration') {
    return <svg viewBox="0 0 24 24"><rect x="3" y="6" width="12" height="12" rx="3" /><path d="m15 10 6-3v10l-6-3v-4Z" /></svg>
  }
  if (category === 'operating-system') {
    return <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="13" rx="2" /><path d="M8 21h8M12 17v4" /></svg>
  }
  return <svg viewBox="0 0 24 24"><rect x="4" y="4" width="16" height="16" rx="3" /><path d="M8 8h8M8 12h8M8 16h5" /></svg>
}
