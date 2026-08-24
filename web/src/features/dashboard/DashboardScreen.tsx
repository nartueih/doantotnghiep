import { useEffect, useRef, useState, type ReactNode } from 'react'
import { AdminShell, Icon, type AdminPage, type IconName } from '../../components/layout/AdminShell'
import { SoftwareCategoryBadge } from '../../components/software/SoftwareCategoryBadge'
import type { AuthSession } from '../../lib/auth-api'
import {
  DashboardAPIError,
  getDashboardSummary,
  getLicenseAlerts,
  type DashboardSummary,
  type LicenseAlert,
} from '../../lib/dashboard-api'
import { criticalLicenseAlerts } from './dashboard-view-model'
import './DashboardScreen.css'

interface DashboardScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type ExpiryWindow = 30 | 60 | 90
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

export function DashboardScreen({ session, onNavigate, onLogout }: DashboardScreenProps) {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [alerts, setAlerts] = useState<LicenseAlert[]>([])
  const [expiryWindow, setExpiryWindow] = useState<ExpiryWindow>(90)
  const [reloadKey, setReloadKey] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<DashboardAPIError | null>(null)
  const [isNotificationOpen, setIsNotificationOpen] = useState(false)
  const notificationRef = useRef<HTMLDivElement>(null)

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

  useEffect(() => {
    function closeNotification(event: MouseEvent) {
      if (notificationRef.current && !notificationRef.current.contains(event.target as Node)) setIsNotificationOpen(false)
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setIsNotificationOpen(false)
    }
    document.addEventListener('mousedown', closeNotification)
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.removeEventListener('mousedown', closeNotification)
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [])

  const criticalItems = criticalLicenseAlerts(alerts)

  function showAllAlerts() {
    setIsNotificationOpen(false)
    document.getElementById('license-alerts')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  const headerActions = <div className="topbar-actions">
    <button
      className="refresh-button"
      type="button"
      onClick={() => setReloadKey((key) => key + 1)}
      disabled={isLoading}
    >
      <Icon name="refresh" />
      <span>Làm mới</span>
    </button>
    <div className="admin-alert-notifications" ref={notificationRef}>
      <button
        className="notification-button"
        type="button"
        aria-label={`${criticalItems.length} cảnh báo license nghiêm trọng`}
        aria-expanded={isNotificationOpen}
        aria-controls="admin-license-alert-panel"
        onClick={() => setIsNotificationOpen((open) => !open)}
      >
        <Icon name="bell" />
        {criticalItems.length > 0 && <span>{criticalItems.length > 99 ? '99+' : criticalItems.length}</span>}
      </button>
      {isNotificationOpen && <section id="admin-license-alert-panel" className="admin-alert-panel" aria-label="Cảnh báo license nghiêm trọng">
        <header><div><strong>Cảnh báo license</strong><span>{criticalItems.length} mục nghiêm trọng</span></div><button type="button" onClick={() => setIsNotificationOpen(false)} aria-label="Đóng"><Icon name="close" /></button></header>
        {isLoading ? <div className="admin-alert-loading"><span /><span /></div> : criticalItems.length ? <div className="admin-alert-list">
          {criticalItems.map((alert) => <button type="button" key={alert.license_id} onClick={() => { setIsNotificationOpen(false); onNavigate('licenses') }}>
            <SoftwareCategoryBadge name={alert.license_name} size="compact" />
            <span><strong>{alert.license_name}</strong><small>{alert.alert_types.map((type) => alertTypeLabels[type] ?? type).join(' · ')}</small><p>{formatAlertNotification(alert)}</p></span>
            <Icon name="chevron" />
          </button>)}
        </div> : <div className="admin-alert-empty"><Icon name="check" /><strong>Không có cảnh báo nghiêm trọng</strong><p>Các license hiện chưa cần xử lý khẩn cấp.</p></div>}
        <footer><button type="button" onClick={showAllAlerts}>Xem tất cả cảnh báo</button></footer>
      </section>}
    </div>
  </div>

  return (
    <AdminShell session={session} activePage="dashboard" title="Tổng quan" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
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
    </AdminShell>
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
    <section className="panel alerts-panel" id="license-alerts">
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

function formatAlertNotification(alert: LicenseAlert): string {
  const expiry = formatExpiry(alert)
  const seat = `${alert.used_seats}/${alert.seat_count} seat đã dùng`
  return alert.expires_at ? `${expiry} · ${seat}` : seat
}
