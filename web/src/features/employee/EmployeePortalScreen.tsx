import { useEffect, useMemo, useState } from 'react'
import { Icon } from '../../components/layout/AdminShell'
import type { AuthSession } from '../../lib/auth-api'
import type { DeviceItem } from '../../lib/device-api'
import {
  getMyDevices,
  getMyLicenses,
  SelfServiceAPIError,
  type MyAssignedLicense,
} from '../../lib/self-service-api'
import './EmployeePortalScreen.css'

interface EmployeePortalScreenProps {
  session: AuthSession
  onLogout: () => Promise<void>
}

interface PortalError {
  message: string
  status: number
}

const deviceStatusLabels: Record<string, string> = {
  assigned: 'Đang sử dụng',
  available: 'Sẵn sàng',
  maintenance: 'Bảo trì',
  retired: 'Ngừng sử dụng',
  lost: 'Thất lạc',
}

const deviceTypeLabels: Record<string, string> = {
  laptop: 'Laptop',
  desktop: 'Máy tính để bàn',
  workstation: 'Máy trạm',
  server: 'Máy chủ',
  mobile: 'Điện thoại',
  tablet: 'Máy tính bảng',
}

export function EmployeePortalScreen({ session, onLogout }: EmployeePortalScreenProps) {
  const [devices, setDevices] = useState<DeviceItem[]>([])
  const [licenses, setLicenses] = useState<MyAssignedLicense[]>([])
  const [sourceFilter, setSourceFilter] = useState<'all' | 'user' | 'device'>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<PortalError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    Promise.all([
      getMyDevices(session.tokens.access_token),
      getMyLicenses(session.tokens.access_token),
    ])
      .then(([deviceResult, licenseResult]) => {
        if (cancelled) return
        setDevices(deviceResult.items)
        setLicenses(licenseResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        if (caughtError instanceof SelfServiceAPIError) {
          setError({ message: caughtError.message, status: caughtError.status })
        } else {
          setError({ message: 'Đã xảy ra lỗi không mong muốn.', status: 0 })
        }
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })

    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const filteredLicenses = useMemo(() => sourceFilter === 'all'
    ? licenses
    : licenses.filter((license) => license.assignment_source === sourceFilter), [licenses, sourceFilter])

  const stats = useMemo(() => {
    const direct = licenses.filter((license) => license.assignment_source === 'user').length
    const viaDevice = licenses.filter((license) => license.assignment_source === 'device').length
    const attention = licenses.filter((license) => licenseNeedsAttention(license)).length
    return { devices: devices.length, licenses: licenses.length, direct, viaDevice, attention }
  }, [devices.length, licenses])

  const initials = userInitials(session.user.full_name)

  return <div className="employee-portal">
    <header className="employee-header">
      <a className="employee-brand" href="#/portal" aria-label="Trang chủ License Manager"><span>LM</span><div><strong>License Manager</strong><small>Cổng thông tin nhân viên</small></div></a>
      <nav aria-label="Điều hướng cổng nhân viên"><a href="#my-devices">Thiết bị của tôi</a><a href="#my-licenses">License của tôi</a></nav>
      <div className="employee-account"><span>{initials}</span><div><strong>{session.user.full_name}</strong><small>{session.user.employee_code}</small></div><button type="button" onClick={onLogout}>Đăng xuất</button></div>
    </header>

    <main className="employee-main">
      <section className="employee-hero">
        <div className="employee-hero-copy"><span className="employee-eyebrow">Không gian cá nhân</span><h1>Chào {firstName(session.user.full_name)},<br />mọi tài sản của bạn ở đây.</h1><p>Theo dõi thiết bị công ty và quyền sử dụng phần mềm đang được cấp cho bạn trong một nơi duy nhất.</p><div className="employee-profile-meta"><span><Icon name="users" />{session.user.department_name || 'Chưa có phòng ban'}</span><span><Icon name="check" />Tài khoản đang hoạt động</span></div></div>
        <div className="employee-hero-card"><span className="employee-hero-avatar">{initials}</span><div><small>Hồ sơ nhân viên</small><strong>{session.user.full_name}</strong><p>{session.user.email}</p></div><dl><div><dt>Mã nhân viên</dt><dd>{session.user.employee_code}</dd></div><div><dt>Phòng ban</dt><dd>{session.user.department_name || 'Chưa phân phòng'}</dd></div></dl></div>
      </section>

      <section className="employee-overview" aria-label="Tổng quan tài sản cá nhân">
        <PortalStat icon="device" label="Thiết bị đang giữ" value={stats.devices} detail="tài sản công ty giao cho bạn" tone="blue" loading={isLoading} />
        <PortalStat icon="key" label="License được cấp" value={stats.licenses} detail={`${stats.direct} trực tiếp · ${stats.viaDevice} theo thiết bị`} tone="violet" loading={isLoading} />
        <PortalStat icon="alert" label="Cần chú ý" value={stats.attention} detail="license đã hoặc sắp hết hạn" tone={stats.attention ? 'amber' : 'green'} loading={isLoading} />
      </section>

      {error ? <PortalErrorState error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : <>
        {stats.attention > 0 && !isLoading && <section className="employee-alert"><span><Icon name="alert" /></span><div><strong>Bạn có {stats.attention} license cần chú ý</strong><p>Hãy liên hệ bộ phận IT nếu phần mềm sắp hết hạn vẫn cần cho công việc.</p></div><a href="#my-licenses">Xem license</a></section>}

        <div className="employee-content-grid">
          <section className="employee-panel employee-devices-panel" id="my-devices">
            <header><div><span><Icon name="device" /></span><div><h2>Thiết bị của tôi</h2><p>Tài sản công ty đang được bàn giao</p></div></div><button type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading} aria-label="Làm mới dữ liệu"><Icon name="refresh" /></button></header>
            {isLoading ? <PortalLoading count={2} /> : devices.length ? <div className="employee-device-list">{devices.map((device) => <DeviceCard device={device} key={device.id} />)}</div> : <PortalEmpty icon="device" title="Bạn chưa được giao thiết bị" detail="Thiết bị được IT bàn giao sẽ xuất hiện tại đây." />}
          </section>

          <section className="employee-panel employee-licenses-panel" id="my-licenses">
            <header><div><span><Icon name="key" /></span><div><h2>License của tôi</h2><p>Quyền sử dụng phần mềm đang có hiệu lực</p></div></div><div className="employee-license-tabs" role="group" aria-label="Lọc nguồn cấp license"><button className={sourceFilter === 'all' ? 'active' : ''} type="button" onClick={() => setSourceFilter('all')}>Tất cả <span>{licenses.length}</span></button><button className={sourceFilter === 'user' ? 'active' : ''} type="button" onClick={() => setSourceFilter('user')}>Trực tiếp <span>{stats.direct}</span></button><button className={sourceFilter === 'device' ? 'active' : ''} type="button" onClick={() => setSourceFilter('device')}>Theo thiết bị <span>{stats.viaDevice}</span></button></div></header>
            {isLoading ? <PortalLoading count={3} /> : filteredLicenses.length ? <div className="employee-license-list">{filteredLicenses.map((license) => <LicenseCard license={license} key={license.assignment_id} />)}</div> : <PortalEmpty icon="key" title={licenses.length ? 'Không có license phù hợp' : 'Bạn chưa được cấp license'} detail={licenses.length ? 'Hãy chọn một nhóm license khác.' : 'License được cấp trực tiếp hoặc qua thiết bị sẽ xuất hiện tại đây.'} />}
          </section>
        </div>
      </>}

      <section className="employee-privacy"><Icon name="check" /><div><strong>Dữ liệu cá nhân được bảo vệ</strong><p>Portal chỉ hiển thị tài sản thuộc tài khoản đang đăng nhập. License key và dữ liệu của nhân viên khác không bao giờ được trả về.</p></div></section>
    </main>
    <footer className="employee-footer"><span>License Manager · Enterprise</span><span>Cần hỗ trợ? Liên hệ bộ phận IT nội bộ.</span></footer>
  </div>
}

function DeviceCard({ device }: { device: DeviceItem }) {
  return <article className="employee-device-card">
    <div className="employee-device-top"><span className="employee-device-icon"><Icon name="device" /></span><span className={`employee-status status-${device.status}`}>{deviceStatusLabels[device.status] ?? device.status}</span></div>
    <div className="employee-device-name"><strong>{device.name}</strong><span>{device.asset_code}</span></div>
    <p>{[device.manufacturer, device.model].filter(Boolean).join(' · ') || deviceTypeLabels[device.device_type] || device.device_type}</p>
    <dl><div><dt>Loại thiết bị</dt><dd>{deviceTypeLabels[device.device_type] ?? device.device_type}</dd></div><div><dt>Serial number</dt><dd>{device.serial_number || 'Chưa cập nhật'}</dd></div><div><dt>Hạn bảo hành</dt><dd className={dateIsPast(device.warranty_expires_at) ? 'warning' : ''}>{formatDate(device.warranty_expires_at, 'Không xác định')}</dd></div></dl>
  </article>
}

function LicenseCard({ license }: { license: MyAssignedLicense }) {
  const days = daysUntil(license.expires_at)
  const attention = licenseNeedsAttention(license)
  return <article className="employee-license-card">
    <span className="employee-license-mark">{licenseInitials(license.license_name)}</span>
    <div className="employee-license-info"><div><strong>{license.license_name}</strong><span className={`employee-license-status ${attention ? 'attention' : ''}`}>{licenseStatusLabel(license, days)}</span></div><p>{license.notes || 'License phục vụ công việc của bạn.'}</p><div className="employee-license-meta"><span><Icon name="assignment" />{license.assignment_source === 'device' ? `Theo thiết bị ${license.device_asset_code || ''}` : 'Cấp trực tiếp cho bạn'}</span><span><Icon name="calendar" />{license.license_type === 'perpetual' && !license.expires_at ? 'Vĩnh viễn' : `Hết hạn ${formatDate(license.expires_at, 'chưa xác định')}`}</span></div></div>
    <span className="employee-license-type">{license.license_type === 'perpetual' ? 'Vĩnh viễn' : 'Thuê bao'}</span>
  </article>
}

function PortalStat({ icon, label, value, detail, tone, loading }: { icon: 'device' | 'key' | 'alert'; label: string; value: number; detail: string; tone: string; loading: boolean }) {
  return <article className={`employee-stat ${tone}`}><span><Icon name={icon} /></span><div><small>{label}</small><strong>{loading ? '—' : value}</strong><p>{detail}</p></div></article>
}

function PortalLoading({ count }: { count: number }) {
  return <div className="employee-loading" aria-label="Đang tải dữ liệu">{Array.from({ length: count }, (_, index) => <span key={index} />)}</div>
}

function PortalEmpty({ icon, title, detail }: { icon: 'device' | 'key'; title: string; detail: string }) {
  return <div className="employee-empty"><span><Icon name={icon} /></span><strong>{title}</strong><p>{detail}</p></div>
}

function PortalErrorState({ error, onRetry, onLogout }: { error: PortalError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  return <section className="employee-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : 'Không thể tải dữ liệu cá nhân'}</strong><p>{expired ? 'Đăng nhập lại để tiếp tục.' : error.message}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></section>
}

function licenseNeedsAttention(license: MyAssignedLicense) {
  const days = daysUntil(license.expires_at)
  return license.lifecycle_status === 'expired' || (days !== null && days <= 30)
}

function licenseStatusLabel(license: MyAssignedLicense, days: number | null) {
  if (license.lifecycle_status === 'expired' || (days !== null && days < 0)) return 'Đã hết hạn'
  if (days !== null && days === 0) return 'Hết hạn hôm nay'
  if (days !== null && days <= 30) return `Còn ${days} ngày`
  return 'Đang hoạt động'
}

function daysUntil(value?: string) {
  if (!value) return null
  const target = new Date(`${value}T00:00:00`)
  if (Number.isNaN(target.getTime())) return null
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Math.ceil((target.getTime() - today.getTime()) / 86_400_000)
}

function dateIsPast(value?: string) {
  const days = daysUntil(value)
  return days !== null && days < 0
}

function formatDate(value: string | undefined, fallback: string) {
  if (!value) return fallback
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? fallback : new Intl.DateTimeFormat('vi-VN').format(date)
}

function firstName(fullName: string) {
  return fullName.trim().split(/\s+/).slice(-1)[0] || 'bạn'
}

function userInitials(fullName: string) {
  return fullName.split(/\s+/).filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || 'NV'
}

function licenseInitials(name: string) {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join('').toUpperCase() || 'SW'
}
