import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import type { AuthSession, AuthUser } from '../../lib/auth-api'
import {
  assignDevice,
  createDevice,
  DeviceAPIError,
  getDeviceLicenseAssignments,
  getDevices,
  getDeviceUsers,
  updateDevice,
  updateDeviceStatus,
  type DeviceInput,
  type DeviceItem,
  type DeviceLicenseAssignment,
  type DeviceStatus,
} from '../../lib/device-api'
import './DeviceManagementScreen.css'

interface DeviceManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type DeviceFilter = 'all' | DeviceStatus

const filterLabels: Record<DeviceFilter, string> = {
  all: 'Tất cả trạng thái',
  available: 'Khả dụng',
  assigned: 'Đã bàn giao',
  maintenance: 'Đang bảo trì',
  retired: 'Đã thanh lý',
  lost: 'Thất lạc',
}

export function DeviceManagementScreen({ session, onNavigate, onLogout }: DeviceManagementScreenProps) {
  const [devices, setDevices] = useState<DeviceItem[]>([])
  const [users, setUsers] = useState<AuthUser[]>([])
  const [licenseAssignments, setLicenseAssignments] = useState<DeviceLicenseAssignment[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<DeviceFilter>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<DeviceAPIError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [formDevice, setFormDevice] = useState<DeviceItem | 'new' | null>(null)
  const [ownerDialog, setOwnerDialog] = useState<{ device: DeviceItem; mode: 'assign' | 'unassign' } | null>(null)
  const [statusDevice, setStatusDevice] = useState<DeviceItem | null>(null)
  const [successMessage, setSuccessMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)
    Promise.all([
      getDevices(session.tokens.access_token),
      getDeviceUsers(session.tokens.access_token),
      getDeviceLicenseAssignments(session.tokens.access_token),
    ])
      .then(([deviceResult, userResult, assignmentResult]) => {
        if (cancelled) return
        setDevices(deviceResult.items)
        setUsers(userResult.items)
        setLicenseAssignments(assignmentResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        setError(caughtError instanceof DeviceAPIError
          ? caughtError
          : new DeviceAPIError('Đã xảy ra lỗi không mong muốn.', 0))
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })
    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const licensesByDevice = useMemo(() => {
    const result = new Map<string, DeviceLicenseAssignment[]>()
    for (const assignment of licenseAssignments) {
      if (assignment.status !== 'active' || !assignment.device_id) continue
      result.set(assignment.device_id, [...(result.get(assignment.device_id) ?? []), assignment])
    }
    return result
  }, [licenseAssignments])

  const filteredDevices = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return devices.filter((device) => {
      const licenses = licensesByDevice.get(device.id) ?? []
      const matchesSearch = !query || [device.asset_code, device.name, device.serial_number, device.manufacturer, device.model, device.assigned_user_name, ...licenses.map((item) => item.license_name)]
        .some((value) => value?.toLocaleLowerCase('vi').includes(query))
      return matchesSearch && (filter === 'all' || device.status === filter)
    })
  }, [devices, filter, licensesByDevice, search])

  const counts = useMemo(() => ({
    total: devices.length,
    available: devices.filter((item) => item.status === 'available').length,
    assigned: devices.filter((item) => item.status === 'assigned').length,
    attention: devices.filter((item) => item.status === 'maintenance' || item.status === 'retired' || item.status === 'lost').length,
  }), [devices])

  async function saveDevice(input: DeviceInput) {
    if (formDevice === 'new') {
      const created = await createDevice(session.tokens.access_token, input)
      setSuccessMessage(`Đã thêm thiết bị ${created.asset_code}.`)
    } else if (formDevice) {
      const updated = await updateDevice(session.tokens.access_token, formDevice.id, input)
      setSuccessMessage(`Đã cập nhật thiết bị ${updated.asset_code}.`)
    }
    setFormDevice(null)
    setReloadKey((value) => value + 1)
  }

  async function saveOwner(device: DeviceItem, userID: string) {
    const updated = await assignDevice(session.tokens.access_token, device.id, userID)
    setOwnerDialog(null)
    setSuccessMessage(userID ? `Đã bàn giao ${updated.asset_code} cho ${updated.assigned_user_name}.` : `Đã thu hồi ${updated.asset_code} về kho.`)
    setReloadKey((value) => value + 1)
  }

  async function saveStatus(device: DeviceItem, status: Exclude<DeviceStatus, 'assigned'>) {
    const updated = await updateDeviceStatus(session.tokens.access_token, device.id, status)
    setStatusDevice(null)
    setSuccessMessage(`Đã chuyển ${updated.asset_code} sang trạng thái ${deviceStatusLabel(updated.status)}.`)
    setReloadKey((value) => value + 1)
  }

  const headerActions = <button className="device-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}><Icon name="refresh" /><span>Làm mới</span></button>

  return (
    <AdminShell session={session} activePage="devices" title="Quản lý thiết bị" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="device-page">
        <section className="device-page-heading">
          <div><h2>Danh sách thiết bị</h2><p>Theo dõi tài sản, người đang sử dụng, bảo hành và license gắn với thiết bị.</p></div>
          <button className="add-device-button" type="button" onClick={() => setFormDevice('new')}><Icon name="plus" />Thêm thiết bị</button>
        </section>

        {successMessage && <div className="device-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="device-stats" aria-label="Thống kê thiết bị">
          <DeviceStat label="Tổng thiết bị" value={counts.total} detail="tài sản trong hệ thống" tone="blue" icon="device" loading={isLoading} />
          <DeviceStat label="Khả dụng" value={counts.available} detail="sẵn sàng bàn giao" tone="green" icon="check" loading={isLoading} />
          <DeviceStat label="Đã bàn giao" value={counts.assigned} detail="đang có người dùng" tone="violet" icon="users" loading={isLoading} />
          <DeviceStat label="Cần chú ý" value={counts.attention} detail="bảo trì, thanh lý, thất lạc" tone="amber" icon="alert" loading={isLoading} />
        </section>

        <section className="device-list-card">
          <div className="device-toolbar">
            <div className="device-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm mã tài sản, serial, người dùng, license..." aria-label="Tìm kiếm thiết bị" /></div>
            <div className="device-filter"><Icon name="filter" /><select value={filter} onChange={(event) => setFilter(event.target.value as DeviceFilter)} aria-label="Lọc trạng thái thiết bị">{(Object.keys(filterLabels) as DeviceFilter[]).map((value) => <option value={value} key={value}>{filterLabels[value]}</option>)}</select></div>
            <span className="device-result-count">{filteredDevices.length} kết quả</span>
          </div>

          {error ? <DeviceError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="device-loading" aria-label="Đang tải thiết bị">{Array.from({ length: 6 }, (_, index) => <span key={index} />)}</div>
          ) : filteredDevices.length === 0 ? (
            <div className="device-empty"><span><Icon name="device" /></span><strong>Không tìm thấy thiết bị</strong><p>Thử thay đổi từ khóa, bộ lọc hoặc thêm thiết bị mới.</p></div>
          ) : (
            <div className="device-table-scroll"><table className="device-table">
              <thead><tr><th>Thiết bị</th><th>Thông tin phần cứng</th><th>Người sử dụng</th><th>License thiết bị</th><th>Bảo hành</th><th>Trạng thái</th><th /></tr></thead>
              <tbody>{filteredDevices.map((device) => <DeviceRow device={device} licenses={licensesByDevice.get(device.id) ?? []} onEdit={setFormDevice} onOwner={(item) => setOwnerDialog({ device: item, mode: item.assigned_user_id ? 'unassign' : 'assign' })} onStatus={setStatusDevice} key={device.id} />)}</tbody>
            </table></div>
          )}
        </section>
      </div>

      {formDevice && <DeviceFormDialog device={formDevice === 'new' ? undefined : formDevice} onClose={() => setFormDevice(null)} onSubmit={saveDevice} />}
      {ownerDialog && <DeviceOwnerDialog device={ownerDialog.device} mode={ownerDialog.mode} users={users} onClose={() => setOwnerDialog(null)} onSubmit={(userID) => saveOwner(ownerDialog.device, userID)} />}
      {statusDevice && <DeviceStatusDialog device={statusDevice} activeLicenseCount={(licensesByDevice.get(statusDevice.id) ?? []).length} onClose={() => setStatusDevice(null)} onSubmit={(status) => saveStatus(statusDevice, status)} />}
    </AdminShell>
  )
}

function DeviceFormDialog({ device, onClose, onSubmit }: { device?: DeviceItem; onClose: () => void; onSubmit: (input: DeviceInput) => Promise<void> }) {
  const [input, setInput] = useState<DeviceInput>(() => ({
    asset_code: device?.asset_code ?? '', serial_number: device?.serial_number ?? '', name: device?.name ?? '', device_type: device?.device_type ?? 'laptop',
    manufacturer: device?.manufacturer ?? '', model: device?.model ?? '', purchased_at: device?.purchased_at ?? '', warranty_expires_at: device?.warranty_expires_at ?? '',
  }))
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function update<K extends keyof DeviceInput>(key: K, value: DeviceInput[K]) { setInput((current) => ({ ...current, [key]: value })) }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!input.asset_code.trim() || !input.name.trim() || !input.device_type.trim()) { setError('Vui lòng nhập mã tài sản, tên và loại thiết bị.'); return }
    if (input.purchased_at && input.warranty_expires_at && input.warranty_expires_at < input.purchased_at) { setError('Ngày hết hạn bảo hành không được trước ngày mua.'); return }
    setError(''); setIsSubmitting(true)
    try { await onSubmit(input) } catch (caughtError) { setError(translateDeviceError(caughtError instanceof Error ? caughtError.message : 'Không thể lưu thiết bị.')); setIsSubmitting(false) }
  }

  return <div className="device-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="device-form-dialog" role="dialog" aria-modal="true" aria-labelledby="device-form-title">
    <header><div><span><Icon name={device ? 'edit' : 'plus'} /></span><div><h2 id="device-form-title">{device ? 'Chỉnh sửa thiết bị' : 'Thêm thiết bị'}</h2><p>{device ? 'Cập nhật thông tin tài sản và bảo hành.' : 'Tạo tài sản mới trong hệ thống.'}</p></div></div><button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button></header>
    <form onSubmit={submit}><div className="device-form-body">
      <div className="device-form-section"><span>01</span><div><strong>Nhận diện tài sản</strong><small>Mã quản lý và loại thiết bị</small></div></div>
      <div className="device-form-grid"><label>Mã tài sản<input value={input.asset_code} onChange={(event) => update('asset_code', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: LT-007" disabled={isSubmitting} required /></label><label>Serial number<input value={input.serial_number} onChange={(event) => update('serial_number', event.target.value)} placeholder="Số serial của nhà sản xuất" disabled={isSubmitting} /></label><label className="full">Tên thiết bị<input value={input.name} onChange={(event) => update('name', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: Laptop Dell Latitude" disabled={isSubmitting} required /></label><label>Loại thiết bị<select value={input.device_type} onChange={(event) => update('device_type', event.target.value)} disabled={isSubmitting}><option value="laptop">Laptop</option><option value="desktop">Máy bàn</option><option value="workstation">Workstation</option><option value="server">Máy chủ</option><option value="tablet">Máy tính bảng</option><option value="phone">Điện thoại</option><option value="other">Khác</option></select></label><label>Hãng sản xuất<input value={input.manufacturer} onChange={(event) => update('manufacturer', event.target.value)} placeholder="Dell, HP, Apple..." disabled={isSubmitting} /></label></div>
      <div className="device-form-section"><span>02</span><div><strong>Cấu hình và bảo hành</strong><small>Model, ngày mua và thời hạn hỗ trợ</small></div></div>
      <div className="device-form-grid"><label className="full">Model<input value={input.model} onChange={(event) => update('model', event.target.value)} placeholder="Tên hoặc mã model" disabled={isSubmitting} /></label><label>Ngày mua<input type="date" value={input.purchased_at} onChange={(event) => update('purchased_at', event.target.value)} disabled={isSubmitting} /></label><label>Hết hạn bảo hành<input type="date" value={input.warranty_expires_at} onChange={(event) => update('warranty_expires_at', event.target.value)} disabled={isSubmitting} /></label></div>
      {error && <div className="device-dialog-error" role="alert"><Icon name="alert" />{error}</div>}
    </div><footer><button className="device-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="device-submit" type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang lưu...' : device ? 'Lưu thay đổi' : 'Thêm thiết bị'}</button></footer></form>
  </section></div>
}

function DeviceOwnerDialog({ device, mode, users, onClose, onSubmit }: { device: DeviceItem; mode: 'assign' | 'unassign'; users: AuthUser[]; onClose: () => void; onSubmit: (userID: string) => Promise<void> }) {
  const activeUsers = users.filter((item) => item.status === 'active')
  const [userID, setUserID] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  async function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if (mode === 'assign' && !userID) { setError('Vui lòng chọn người nhận thiết bị.'); return } setError(''); setIsSubmitting(true); try { await onSubmit(mode === 'assign' ? userID : '') } catch (caughtError) { setError(translateDeviceError(caughtError instanceof Error ? caughtError.message : 'Không thể cập nhật bàn giao.')); setIsSubmitting(false) } }
  return <div className="device-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="device-owner-dialog" role="dialog" aria-modal="true" aria-labelledby="device-owner-title"><span className={mode === 'assign' ? 'owner-dialog-icon assign' : 'owner-dialog-icon unassign'}><Icon name={mode === 'assign' ? 'users' : 'undo'} /></span><h2 id="device-owner-title">{mode === 'assign' ? 'Bàn giao thiết bị' : 'Thu hồi thiết bị'}</h2><p><strong>{device.asset_code}</strong> · {device.name}</p><form onSubmit={submit}>{mode === 'assign' ? <label>Người nhận<select value={userID} onChange={(event) => setUserID(event.target.value)} disabled={isSubmitting} required><option value="">Chọn người dùng — không được bỏ trống</option>{activeUsers.map((user) => <option value={user.id} key={user.id}>{user.full_name} — {user.employee_code}{user.department_name ? ` · ${user.department_name}` : ''}</option>)}</select><small>Chỉ hiển thị tài khoản đang hoạt động.</small></label> : <div className="device-owner-note"><Icon name="check" /><span>Thiết bị sẽ trở lại trạng thái khả dụng. Các license cấp trực tiếp theo thiết bị không bị thu hồi.</span></div>}{error && <div className="device-dialog-error" role="alert"><Icon name="alert" />{error}</div>}<footer><button className="device-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className={mode === 'assign' ? 'device-submit' : 'device-unassign-confirm'} type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang cập nhật...' : mode === 'assign' ? 'Xác nhận bàn giao' : 'Xác nhận thu hồi'}</button></footer></form></section></div>
}

function DeviceStatusDialog({ device, activeLicenseCount, onClose, onSubmit }: { device: DeviceItem; activeLicenseCount: number; onClose: () => void; onSubmit: (status: Exclude<DeviceStatus, 'assigned'>) => Promise<void> }) {
  const initialStatus = device.status === 'assigned' ? 'available' : device.status
  const [status, setStatus] = useState<Exclude<DeviceStatus, 'assigned'>>(initialStatus)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  async function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); setError(''); setIsSubmitting(true); try { await onSubmit(status) } catch (caughtError) { setError(translateDeviceError(caughtError instanceof Error ? caughtError.message : 'Không thể đổi trạng thái.')); setIsSubmitting(false) } }
  return <div className="device-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="device-status-dialog" role="dialog" aria-modal="true" aria-labelledby="device-status-title"><span className="status-dialog-icon"><Icon name="settings" /></span><h2 id="device-status-title">Đổi trạng thái thiết bị</h2><p><strong>{device.asset_code}</strong> · {device.name}</p><form onSubmit={submit}><label>Trạng thái mới<select value={status} onChange={(event) => setStatus(event.target.value as Exclude<DeviceStatus, 'assigned'>)} disabled={isSubmitting}><option value="available">Khả dụng</option><option value="maintenance">Đang bảo trì</option><option value="retired">Đã thanh lý</option><option value="lost">Thất lạc</option></select></label>{activeLicenseCount > 0 && (status === 'retired' || status === 'lost') && <div className="device-status-warning"><Icon name="alert" /><span>Thiết bị còn {activeLicenseCount} license đang hoạt động. Hãy thu hồi chúng tại trang Cấp phát nếu không còn sử dụng.</span></div>}{error && <div className="device-dialog-error" role="alert"><Icon name="alert" />{error}</div>}<footer><button className="device-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="device-submit" type="submit" disabled={isSubmitting || status === device.status}>{isSubmitting ? 'Đang cập nhật...' : 'Cập nhật trạng thái'}</button></footer></form></section></div>
}

function DeviceStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'device' | 'check' | 'users' | 'alert'; loading: boolean }) { return <article className="device-stat"><span className={tone}><Icon name={icon} /></span><div><p>{label}</p>{loading ? <i /> : <strong>{value}</strong>}<small>{detail}</small></div></article> }

function DeviceRow({ device, licenses, onEdit, onOwner, onStatus }: { device: DeviceItem; licenses: DeviceLicenseAssignment[]; onEdit: (device: DeviceItem) => void; onOwner: (device: DeviceItem) => void; onStatus: (device: DeviceItem) => void }) {
  const warranty = warrantyInfo(device.warranty_expires_at)
  return <tr className={device.status === 'retired' || device.status === 'lost' ? 'inactive' : undefined}><td><div className="device-identity"><span><Icon name="device" /></span><div><strong>{device.asset_code}</strong><small>{device.name}</small></div></div></td><td><strong className="device-model">{[device.manufacturer, device.model].filter(Boolean).join(' ') || 'Chưa cập nhật'}</strong><small className="device-serial">{device.serial_number || 'Chưa có serial'} · {deviceTypeLabel(device.device_type)}</small></td><td>{device.assigned_user_name ? <div className="device-owner"><span>{targetInitials(device.assigned_user_name)}</span><div><strong>{device.assigned_user_name}</strong><small>Đang sử dụng</small></div></div> : <span className="device-unassigned">Chưa bàn giao</span>}</td><td>{licenses.length ? <div className="device-licenses"><strong>{licenses[0].license_name}</strong>{licenses.length > 1 && <small>+{licenses.length - 1} license khác</small>}</div> : <span className="device-no-license">Chưa có license</span>}</td><td><strong className={`warranty-value ${warranty.tone}`}>{warranty.label}</strong><small className="warranty-date">{device.warranty_expires_at ? formatDate(device.warranty_expires_at) : 'Chưa cập nhật'}</small></td><td><span className={`device-status ${device.status}`}><i />{deviceStatusLabel(device.status)}</span></td><td><div className="device-row-actions"><button type="button" onClick={() => onEdit(device)} title="Chỉnh sửa" aria-label={`Chỉnh sửa ${device.asset_code}`}><Icon name="edit" /></button><button type="button" onClick={() => onOwner(device)} disabled={device.status !== 'available' && device.status !== 'assigned'} title={device.status === 'assigned' ? 'Thu hồi thiết bị' : device.status === 'available' ? 'Bàn giao thiết bị' : 'Thiết bị không khả dụng để bàn giao'} aria-label={`${device.assigned_user_id ? 'Thu hồi' : 'Bàn giao'} ${device.asset_code}`}><Icon name={device.assigned_user_id ? 'undo' : 'users'} /></button><button type="button" onClick={() => onStatus(device)} disabled={device.status === 'assigned'} title={device.status === 'assigned' ? 'Thu hồi thiết bị trước khi đổi trạng thái' : 'Đổi trạng thái'} aria-label={`Đổi trạng thái ${device.asset_code}`}><Icon name="settings" /></button></div></td></tr>
}

function DeviceError({ error, onRetry, onLogout }: { error: DeviceAPIError; onRetry: () => void; onLogout: () => Promise<void> }) { const authError = error.status === 401 || error.status === 403; return <div className="device-error"><Icon name="alert" /><strong>{authError ? 'Không thể truy cập' : 'Không thể tải thiết bị'}</strong><p>{error.status === 401 ? 'Phiên đăng nhập đã hết hạn.' : error.status === 403 ? 'Tài khoản không có quyền quản lý thiết bị.' : error.status === 0 ? 'Hãy kiểm tra backend đang chạy ở cổng 8081.' : error.message}</p><button type="button" onClick={authError ? onLogout : onRetry}>{authError ? 'Đăng nhập lại' : 'Thử lại'}</button></div> }

function translateDeviceError(message: string): string { if (message.includes('asset code already exists')) return 'Mã tài sản đã tồn tại.'; if (message.includes('serial number already exists')) return 'Serial number đã tồn tại.'; if (message.includes('must be unassigned')) return 'Hãy thu hồi thiết bị khỏi người dùng trước khi đổi trạng thái.'; if (message.includes('not available for assignment')) return 'Thiết bị không ở trạng thái khả dụng để bàn giao.'; if (message.includes('user does not exist')) return 'Người dùng không tồn tại hoặc đang bị khóa.'; if (message.includes('warranty expiration')) return 'Ngày hết hạn bảo hành không hợp lệ.'; return message }
function deviceStatusLabel(value: DeviceStatus): string { return value === 'available' ? 'Khả dụng' : value === 'assigned' ? 'Đã bàn giao' : value === 'maintenance' ? 'Đang bảo trì' : value === 'retired' ? 'Đã thanh lý' : 'Thất lạc' }
function deviceTypeLabel(value: string): string { const labels: Record<string,string> = { laptop: 'Laptop', desktop: 'Máy bàn', workstation: 'Workstation', server: 'Máy chủ', tablet: 'Máy tính bảng', phone: 'Điện thoại', other: 'Khác' }; return labels[value] ?? value }
function targetInitials(name: string): string { return name.split(/\s+/).filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || '—' }
function formatDate(value: string): string { return new Intl.DateTimeFormat('vi-VN').format(new Date(`${value}T00:00:00`)) }
function warrantyInfo(value?: string): { label: string; tone: string } { if (!value) return { label: 'Chưa cập nhật', tone: 'muted' }; const today = new Date(); today.setHours(0,0,0,0); const expiry = new Date(`${value}T00:00:00`); const days = Math.round((expiry.getTime() - today.getTime()) / 86400000); if (days < 0) return { label: 'Hết bảo hành', tone: 'expired' }; if (days <= 90) return { label: `Còn ${days} ngày`, tone: 'warning' }; return { label: 'Còn bảo hành', tone: 'active' } }
