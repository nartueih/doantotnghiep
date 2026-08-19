import { useMemo, useState, type ReactNode } from 'react'
import type { AuthSession, UserRole } from '../../lib/auth-api'
import './AdminShell.css'

export type AdminPage = 'dashboard' | 'licenses'
export type IconName = 'grid' | 'software' | 'key' | 'assignment' | 'device' | 'users' |
  'department' | 'audit' | 'bell' | 'refresh' | 'search' | 'chevron' | 'trend' |
  'calendar' | 'alert' | 'check' | 'menu' | 'plus' | 'filter' | 'eye' | 'edit' | 'archive' | 'close'

interface AdminShellProps {
  session: AuthSession
  activePage: AdminPage
  title: string
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
  actions?: ReactNode
  children: ReactNode
}

const roleLabels: Record<UserRole, string> = {
  admin: 'Quản trị viên',
  it_manager: 'Quản lý IT',
  employee: 'Nhân viên',
}

const navigation: Array<{ label: string; icon: IconName; page?: AdminPage }> = [
  { label: 'Tổng quan', icon: 'grid', page: 'dashboard' },
  { label: 'Phần mềm', icon: 'software' },
  { label: 'License', icon: 'key', page: 'licenses' },
  { label: 'Cấp phát', icon: 'assignment' },
  { label: 'Thiết bị', icon: 'device' },
  { label: 'Người dùng', icon: 'users' },
  { label: 'Phòng ban', icon: 'department' },
  { label: 'Nhật ký', icon: 'audit' },
]

export function AdminShell({ session, activePage, title, onNavigate, onLogout, actions, children }: AdminShellProps) {
  const [isNavOpen, setIsNavOpen] = useState(false)
  const initials = useMemo(() => session.user.full_name
    .split(' ')
    .filter(Boolean)
    .slice(-2)
    .map((part) => part[0])
    .join('')
    .toUpperCase(), [session.user.full_name])

  function navigate(page: AdminPage) {
    onNavigate(page)
    setIsNavOpen(false)
  }

  return (
    <div className="admin-shell">
      <aside className={isNavOpen ? 'admin-sidebar open' : 'admin-sidebar'}>
        <div className="admin-brand">
          <span className="admin-brand-mark" aria-hidden="true">LM</span>
          <div><strong>License Manager</strong><span>Enterprise</span></div>
        </div>

        <nav aria-label="Điều hướng chính">
          <span className="admin-nav-label">Quản lý</span>
          {navigation.slice(0, 5).map((item) => (
            <NavItem key={item.label} item={item} activePage={activePage} onNavigate={navigate} />
          ))}
          <span className="admin-nav-label second">Hệ thống</span>
          {navigation.slice(5).map((item) => (
            <NavItem key={item.label} item={item} activePage={activePage} onNavigate={navigate} />
          ))}
        </nav>

        <div className="admin-account">
          <span className="admin-avatar">{initials || 'U'}</span>
          <div><strong>{session.user.full_name}</strong><span>{roleLabels[session.user.role]}</span></div>
          <button type="button" onClick={onLogout} aria-label="Đăng xuất" title="Đăng xuất">
            <Icon name="chevron" />
          </button>
        </div>
      </aside>

      {isNavOpen && <button className="admin-overlay" onClick={() => setIsNavOpen(false)} aria-label="Đóng menu" />}

      <main className="admin-main">
        <header className="admin-topbar">
          <button className="admin-menu-button" type="button" onClick={() => setIsNavOpen(true)} aria-label="Mở menu">
            <Icon name="menu" />
          </button>
          <div className="admin-title">
            <span>Không gian quản trị</span>
            <h1>{title}</h1>
          </div>
          {actions && <div className="admin-actions">{actions}</div>}
        </header>
        {children}
      </main>
    </div>
  )
}

function NavItem({ item, activePage, onNavigate }: {
  item: { label: string; icon: IconName; page?: AdminPage }
  activePage: AdminPage
  onNavigate: (page: AdminPage) => void
}) {
  const active = item.page === activePage
  if (!item.page) {
    return <span className="admin-nav-item disabled"><Icon name={item.icon} />{item.label}<small>Sắp có</small></span>
  }
  return (
    <button
      type="button"
      className={active ? 'admin-nav-item active' : 'admin-nav-item'}
      aria-current={active ? 'page' : undefined}
      onClick={() => onNavigate(item.page!)}
    >
      <Icon name={item.icon} />{item.label}
    </button>
  )
}

export function Icon({ name }: { name: IconName }) {
  const paths: Record<IconName, ReactNode> = {
    grid: <><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></>,
    software: <><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M3 9h18M8 4v5" /></>,
    key: <><circle cx="8" cy="15" r="4" /><path d="M11 12l8-8M16 7l3 3M14 9l2 2" /></>,
    assignment: <><path d="M9 5H5a2 2 0 00-2 2v12h16v-5" /><path d="M13 3h8v8M21 3l-10 10" /></>,
    device: <><rect x="5" y="2" width="14" height="20" rx="2" /><path d="M9 18h6" /></>,
    users: <><path d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" /></>,
    department: <><path d="M3 21h18M5 21V8l7-5 7 5v13M9 12h2M13 12h2M9 16h2M13 16h2" /></>,
    audit: <><path d="M4 4h16v16H4zM8 9h8M8 13h8M8 17h5" /></>,
    bell: <><path d="M18 8a6 6 0 00-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></>,
    refresh: <><path d="M20 6v5h-5M4 18v-5h5" /><path d="M18.5 9A7 7 0 006 6.5L4 11M5.5 15A7 7 0 0018 17.5l2-4.5" /></>,
    search: <><circle cx="11" cy="11" r="7" /><path d="M20 20l-4-4" /></>,
    chevron: <path d="M9 18l6-6-6-6" />,
    trend: <><path d="M3 17l6-6 4 4 8-9" /><path d="M15 6h6v6" /></>,
    calendar: <><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M16 3v4M8 3v4M3 10h18" /></>,
    alert: <><path d="M10.3 3.7L2.4 18a2 2 0 001.7 3h15.8a2 2 0 001.7-3L13.7 3.7a2 2 0 00-3.4 0z" /><path d="M12 9v4M12 17h.01" /></>,
    check: <><circle cx="12" cy="12" r="9" /><path d="M8 12l3 3 5-6" /></>,
    menu: <path d="M4 6h16M4 12h16M4 18h16" />,
    plus: <path d="M12 5v14M5 12h14" />,
    filter: <path d="M4 5h16l-6 7v5l-4 2v-7z" />,
    eye: <><path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z" /><circle cx="12" cy="12" r="3" /></>,
    edit: <><path d="M12 20h9M16.5 3.5a2.1 2.1 0 013 3L8 18l-4 1 1-4z" /></>,
    archive: <><path d="M4 8v12h16V8M3 4h18v4H3zM9 12h6" /></>,
    close: <path d="M6 6l12 12M18 6L6 18" />,
  }
  return <svg viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>
}
