import { useEffect, useMemo, useState, type ReactNode } from 'react'
import type { AuthSession, UserRole } from '../../lib/auth-api'
import {
  DashboardAPIError,
  getDashboardSummary,
  getLicenseAlerts,
  type DashboardSummary,
  type LicenseAlert,
} from '../../lib/dashboard-api'
import './DashboardScreen.css'

interface DashboardScreenProps {
  session: AuthSession
  onLogout: () => Promise<void>
}

type ExpiryWindow = 30 | 60 | 90
type IconName = 'grid' | 'software' | 'key' | 'assignment' | 'device' | 'users' |
  'department' | 'audit' | 'bell' | 'refresh' | 'search' | 'chevron' | 'trend' |
  'calendar' | 'alert' | 'check' | 'menu'

const roleLabels: Record<UserRole, string> = {
  admin: 'Quản trị viên',
  it_manager: 'Quản lý IT',
  employee: 'Nhân viên',
}

const navigation: Array<{ label: string; icon: IconName; active?: boolean }> = [
  { label: 'Tổng quan', icon: 'grid', active: true },
  { label: 'Phần mềm', icon: 'software' },
  { label: 'License', icon: 'key' },
  { label: 'Cấp phát', icon: 'assignment' },
  { label: 'Thiết bị', icon: 'device' },
  { label: 'Người dùng', icon: 'users' },
  { label: 'Phòng ban', icon: 'department' },
  { label: 'Nhật ký', icon: 'audit' },
]

const deviceStatusLabels: Record<string, string> = {
  available: 'Sẵn sàng',
  assigned: 'Đang sử dụng',
  maintenance: 'Bảo trì',
  retired: 'Ngừng sử dụng',
}

const alertTypeLabels: Record<string, string> = {
  expired: 'Đã hết hạn',
  expiring: 'Sắp hết hạn',
  exhausted: 'Hết seat',
  high_usage: 'Sử dụng cao',
}

const severityLabels = {
  critical: 'Nghiêm trọng',
  warning: 'Cảnh báo',
  info: 'Thông tin',
}

export function DashboardScreen({ session, onLogout }: DashboardScreenProps) {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [alerts, setAlerts] = useState<LicenseAlert[]>([])
  const [expiryWindow, setExpiryWindow] = useState<ExpiryWindow>(90)
  const [reloadKey, setReloadKey] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<DashboardAPIError | null>(null)
  const [isNavOpen, setIsNavOpen] = useState(false)

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    Promise.all([
      getDashboardSummary(session.tokens.access_token),
      getLicenseAlerts(session.tokens.access_token, expiryWindow),
    ])
      .then(([nextSummary, nextAlerts]) => {
        if (cancelled) return
        setSummary(nextSummary)
        setAlerts(nextAlerts.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        setError(
          caughtError instanceof DashboardAPIError
            ? caughtError
            : new DashboardAPIError('Đã xảy ra lỗi không mong muốn.', 0),
        )
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [expiryWindow, reloadKey, session.tokens.access_token])

  const initials = useMemo(() => session.user.full_name
    .split(' ')
    .filter(Boolean)
    .slice(-2)
    .map((part) => part[0])
    .join('')
    .toUpperCase(), [session.user.full_name])

  const criticalAlerts = alerts.filter((item) => item.severity === 'critical').length

  return (
    <div className="dashboard-shell">
      <aside className={isNavOpen ? 'dashboard-sidebar open' : 'dashboard-sidebar'}>
        <div className="dashboard-brand">
          <span className="dashboard-brand-mark" aria-hidden="true">LM</span>
          <div><strong>License Manager</strong><span>Enterprise</span></div>
        </div>

        <nav aria-label="Điều hướng chính">
          <span className="nav-section-label">Quản lý</span>
          {navigation.slice(0, 5).map((item) => <NavItem key={item.label} {...item} />)}
          <span className="nav-section-label second">Hệ thống</span>
          {navigation.slice(5).map((item) => <NavItem key={item.label} {...item} />)}
        </nav>

        <div className="sidebar-account">
          <span className="sidebar-avatar">{initials || 'U'}</span>
          <div><strong>{session.user.full_name}</strong><span>{roleLabels[session.user.role]}</span></div>
          <button type="button" onClick={onLogout} aria-label="Đăng xuất" title="Đăng xuất">
            <Icon name="chevron" />
          </button>
        </div>
      </aside>

      {isNavOpen && <button className="nav-overlay" onClick={() => setIsNavOpen(false)} aria-label="Đóng menu" />}

      <main className="dashboard-main">
        <header className="dashboard-topbar">
          <button className="mobile-menu-button" type="button" onClick={() => setIsNavOpen(true)} aria-label="Mở menu">
            <Icon name="menu" />
          </button>
          <div className="dashboard-title">
            <span>Không gian quản trị</span>
            <h1>Tổng quan</h1>
          </div>
          <div className="topbar-actions">
            <button
              className="refresh-button"
              type="button"
              onClick={() => setReloadKey((key) => key + 1)}
              disabled={isLoading}
            >
              <Icon name="refresh" />
              <span>Làm mới</span>
            </button>
            <button className="notification-button" type="button" aria-label={`${criticalAlerts} cảnh báo nghiêm trọng`}>
              <Icon name="bell" />
              {criticalAlerts > 0 && <span>{criticalAlerts}</span>}
            </button>
          </div>
        </header>

        <div className="dashboard-content">
          <section className="dashboard-welcome">
            <div>
              <p>Xin chào, {session.user.full_name.split(' ').slice(-1)[0]} 👋</p>
              <span>Đây là tình hình license và thiết bị trong doanh nghiệp của bạn.</span>
            </div>
            {summary && <time dateTime={summary.generated_at}>Cập nhật {formatDateTime(summary.generated_at)}</time>}
          </section>

          {error ? (
            <DashboardError error={error} onRetry={() => setReloadKey((key) => key + 1)} onLogout={onLogout} />
          ) : (
            <>
              <SummaryCards summary={summary} isLoading={isLoading} />
              <section className="dashboard-grid">
                <SeatUsageCard summary={summary} isLoading={isLoading} />
                <ExpiryOverview summary={summary} isLoading={isLoading} />
                <DeviceStatusCard summary={summary} isLoading={isLoading} />
              </section>
              <AlertsTable
                alerts={alerts}
                expiryWindow={expiryWindow}
                onWindowChange={setExpiryWindow}
                isLoading={isLoading}
              />
            </>
          )}
        </div>
      </main>
    </div>
  )
}

function NavItem({ label, icon, active }: { label: string; icon: IconName; active?: boolean }) {
  return (
    <span className={active ? 'nav-item active' : 'nav-item'} aria-current={active ? 'page' : undefined}>
      <Icon name={icon} />
      {label}
      {!active && <small>Sắp có</small>}
    </span>
  )
}

function SummaryCards({ summary, isLoading }: { summary: DashboardSummary | null; isLoading: boolean }) {
  const cards = [
    { label: 'Tổng license', value: summary?.total_licenses ?? 0, detail: `${summary?.total_software_products ?? 0} sản phẩm`, icon: 'key' as IconName, tone: 'blue' },
    { label: 'Tổng thiết bị', value: summary?.total_devices ?? 0, detail: `${summary?.devices_by_status.assigned ?? 0} đang sử dụng`, icon: 'device' as IconName, tone: 'violet' },
    { label: 'Seat còn trống', value: summary?.available_seats ?? 0, detail: `trên ${summary?.total_seats ?? 0} seat`, icon: 'check' as IconName, tone: 'green' },
    { label: 'Tổng chi phí', value: formatCosts(summary?.costs_by_currency ?? []), detail: 'theo dữ liệu license', icon: 'trend' as IconName, tone: 'amber' },
  ]

  return (
    <section className="summary-cards" aria-label="Số liệu tổng quan">
      {cards.map((card) => (
        <article className="summary-card" key={card.label}>
          <span className={`summary-icon ${card.tone}`}><Icon name={card.icon} /></span>
          <div>
            <span>{card.label}</span>
            {isLoading ? <span className="skeleton value" /> : <strong>{card.value}</strong>}
            <small>{card.detail}</small>
          </div>
        </article>
      ))}
    </section>
  )
}

function SeatUsageCard({ summary, isLoading }: { summary: DashboardSummary | null; isLoading: boolean }) {
  const percent = summary && summary.total_seats > 0
    ? Math.round((summary.used_seats / summary.total_seats) * 100)
    : 0

  return (
    <article className="panel seat-panel">
      <PanelHeading title="Mức sử dụng seat" subtitle="Trên toàn bộ license" />
      {isLoading ? <div className="chart-skeleton skeleton" /> : (
        <div className="seat-chart-wrap">
          <div className="seat-chart" style={{ '--usage': `${percent * 3.6}deg` } as React.CSSProperties}>
            <div><strong>{percent}%</strong><span>đã dùng</span></div>
          </div>
          <div className="seat-legend">
            <span><i className="used" /> Đã sử dụng <strong>{summary?.used_seats ?? 0}</strong></span>
            <span><i className="available" /> Còn trống <strong>{summary?.available_seats ?? 0}</strong></span>
            <span><i className="total" /> Tổng cộng <strong>{summary?.total_seats ?? 0}</strong></span>
          </div>
        </div>
      )}
    </article>
  )
}

function ExpiryOverview({ summary, isLoading }: { summary: DashboardSummary | null; isLoading: boolean }) {
  const items = [
    { label: 'Đã hết hạn', value: summary?.expired_licenses ?? 0, className: 'critical' },
    { label: 'Trong 30 ngày', value: summary?.expiring_in_30_days ?? 0, className: 'warning' },
    { label: 'Trong 60 ngày', value: summary?.expiring_in_60_days ?? 0, className: 'notice' },
    { label: 'Trong 90 ngày', value: summary?.expiring_in_90_days ?? 0, className: 'safe' },
  ]

  return (
    <article className="panel expiry-panel">
      <PanelHeading title="Thời hạn license" subtitle="Cần chú ý theo thời gian" />
      <div className="expiry-list">
        {items.map((item) => (
          <div key={item.label}>
            <span className={`expiry-dot ${item.className}`} />
            <span>{item.label}</span>
            {isLoading ? <span className="skeleton tiny" /> : <strong>{item.value}</strong>}
          </div>
        ))}
      </div>
      <div className="usage-alerts">
        <span><Icon name="alert" /> Hết seat <strong>{summary?.exhausted_licenses ?? 0}</strong></span>
        <span><Icon name="trend" /> Sử dụng cao <strong>{summary?.high_usage_licenses ?? 0}</strong></span>
      </div>
    </article>
  )
}

function DeviceStatusCard({ summary, isLoading }: { summary: DashboardSummary | null; isLoading: boolean }) {
  const statuses = Object.entries(summary?.devices_by_status ?? {})
  const total = summary?.total_devices ?? 0

  return (
    <article className="panel device-panel">
      <PanelHeading title="Trạng thái thiết bị" subtitle="Phân bổ thiết bị hiện tại" />
      {isLoading ? <div className="list-skeleton skeleton" /> : statuses.length === 0 ? (
        <EmptyCompact icon="device">Chưa có thiết bị</EmptyCompact>
      ) : (
        <div className="device-status-list">
          {statuses.map(([status, count]) => (
            <div key={status}>
              <div><span>{deviceStatusLabels[status] ?? status}</span><strong>{count}</strong></div>
              <span className="status-track"><i style={{ width: `${total ? (count / total) * 100 : 0}%` }} /></span>
            </div>
          ))}
        </div>
      )}
    </article>
  )
}

function AlertsTable({ alerts, expiryWindow, onWindowChange, isLoading }: {
  alerts: LicenseAlert[]
  expiryWindow: ExpiryWindow
  onWindowChange: (days: ExpiryWindow) => void
  isLoading: boolean
}) {
  return (
    <section className="panel alerts-panel">
      <div className="alerts-heading">
        <PanelHeading title="License cần chú ý" subtitle="Sắp xếp theo mức độ ưu tiên" />
        <div className="window-filter" aria-label="Khoảng thời gian hết hạn">
          {([30, 60, 90] as ExpiryWindow[]).map((days) => (
            <button
              type="button"
              className={expiryWindow === days ? 'active' : ''}
              onClick={() => onWindowChange(days)}
              key={days}
            >
              {days} ngày
            </button>
          ))}
        </div>
      </div>

      {isLoading ? <div className="table-skeleton skeleton" /> : alerts.length === 0 ? (
        <div className="alerts-empty">
          <span><Icon name="check" /></span>
          <strong>Không có cảnh báo</strong>
          <p>Không có license hết hạn hoặc sử dụng seat cao trong {expiryWindow} ngày tới.</p>
        </div>
      ) : (
        <div className="table-scroll">
          <table>
            <thead><tr><th>License</th><th>Cảnh báo</th><th>Thời hạn</th><th>Seat</th><th>Mức độ</th></tr></thead>
            <tbody>
              {alerts.map((alert) => (
                <tr key={alert.license_id}>
                  <td><strong>{alert.license_name}</strong><span>{formatLicenseType(alert.license_type)}</span></td>
                  <td><div className="alert-tags">{alert.alert_types.map((type) => <span key={type}>{alertTypeLabels[type] ?? type}</span>)}</div></td>
                  <td>{formatExpiry(alert)}</td>
                  <td>
                    <span className="seat-count">{alert.used_seats}/{alert.seat_count}</span>
                    <span className="mini-track"><i style={{ width: `${Math.min(alert.utilization_percent, 100)}%` }} /></span>
                  </td>
                  <td><span className={`severity ${alert.severity}`}>{severityLabels[alert.severity]}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

function DashboardError({ error, onRetry, onLogout }: {
  error: DashboardAPIError
  onRetry: () => void
  onLogout: () => Promise<void>
}) {
  const sessionExpired = error.status === 401
  const forbidden = error.status === 403
  const title = sessionExpired ? 'Phiên đăng nhập đã hết hạn' : forbidden ? 'Không có quyền truy cập' : 'Không thể tải dashboard'
  const detail = sessionExpired
    ? 'Vui lòng đăng nhập lại để tiếp tục.'
    : forbidden
      ? 'Dashboard chỉ dành cho Quản trị viên và Quản lý IT.'
      : error.status === 0
        ? 'Hãy kiểm tra backend đang chạy ở cổng 8081 rồi thử lại.'
        : error.message

  return (
    <section className="dashboard-error" role="alert">
      <span><Icon name="alert" /></span>
      <h2>{title}</h2>
      <p>{detail}</p>
      <button type="button" onClick={sessionExpired || forbidden ? onLogout : onRetry}>
        {sessionExpired || forbidden ? 'Quay lại đăng nhập' : 'Thử lại'}
      </button>
    </section>
  )
}

function PanelHeading({ title, subtitle }: { title: string; subtitle: string }) {
  return <div className="panel-heading"><div><h2>{title}</h2><p>{subtitle}</p></div></div>
}

function EmptyCompact({ icon, children }: { icon: IconName; children: ReactNode }) {
  return <div className="empty-compact"><Icon name={icon} /><span>{children}</span></div>
}

function formatCosts(costs: DashboardSummary['costs_by_currency']): string {
  if (costs.length === 0) return '0'
  return costs.map(({ currency, amount }) => {
    try {
      return new Intl.NumberFormat('vi-VN', { style: 'currency', currency, maximumFractionDigits: 0 }).format(amount)
    } catch {
      return `${new Intl.NumberFormat('vi-VN').format(amount)} ${currency}`
    }
  }).join(' · ')
}

function formatDateTime(value: string): string {
  return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function formatLicenseType(value: string): string {
  return value === 'subscription' ? 'Thuê bao' : value === 'perpetual' ? 'Vĩnh viễn' : value
}

function formatExpiry(alert: LicenseAlert): string {
  if (!alert.expires_at) return 'Không thời hạn'
  if (alert.days_until_expiry === undefined) return alert.expires_at
  if (alert.days_until_expiry < 0) return `Quá hạn ${Math.abs(alert.days_until_expiry)} ngày`
  if (alert.days_until_expiry === 0) return 'Hết hạn hôm nay'
  return `Còn ${alert.days_until_expiry} ngày`
}

function Icon({ name }: { name: IconName }) {
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
  }

  return <svg viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>
}
