import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Icon } from '../../components/layout/AdminShell'
import type { DeviceItem } from '../../lib/device-api'
import { cancelMaintenanceRequest, createMaintenanceRequest, listMyMaintenanceRequests, MaintenanceAPIError, type MaintenanceCategory, type MaintenancePriority, type MaintenanceRequestItem } from '../../lib/maintenance-api'
import { maintenanceCategoryLabel, maintenanceStatusLabel } from '../maintenance/maintenance-view-model'

interface Props { accessToken: string; devices: DeviceItem[]; initialDeviceID?: string; onInitialDeviceHandled: () => void; onSessionExpired: () => void; onOpenRequestsChange: (deviceIDs: Set<string>) => void }

export function EmployeeMaintenanceRequests({ accessToken, devices, initialDeviceID, onInitialDeviceHandled, onSessionExpired, onOpenRequestsChange }: Props) {
  const [items, setItems] = useState<MaintenanceRequestItem[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [deviceID, setDeviceID] = useState('')
  const [category, setCategory] = useState<MaintenanceCategory>('hardware')
  const [priority, setPriority] = useState<MaintenancePriority>('normal')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [cancellingID, setCancellingID] = useState('')

  const load = useCallback(async () => {
    setIsLoading(true); setError('')
    try {
      const result = await listMyMaintenanceRequests(accessToken)
      setItems(result.items)
      onOpenRequestsChange(new Set(result.items.filter((item) => item.status === 'pending' || item.status === 'in_progress').map((item) => item.device_id)))
    } catch (caught) { handleError(caught, 'Không thể tải yêu cầu bảo trì.', setError, onSessionExpired) }
    finally { setIsLoading(false) }
  }, [accessToken, onOpenRequestsChange, onSessionExpired])
  useEffect(() => { void load() }, [load])
  useEffect(() => {
    if (!initialDeviceID) return
    setDeviceID(initialDeviceID)
    setIsModalOpen(true)
    onInitialDeviceHandled()
  }, [initialDeviceID, onInitialDeviceHandled])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!deviceID || !title.trim() || !description.trim()) return
    setIsSubmitting(true); setError('')
    try { await createMaintenanceRequest(accessToken, { device_id: deviceID, category, priority, title: title.trim(), description: description.trim() }); closeModal(); await load() }
    catch (caught) { handleError(caught, 'Không thể tạo yêu cầu bảo trì.', setError, onSessionExpired) }
    finally { setIsSubmitting(false) }
  }
  async function cancel(item: MaintenanceRequestItem) {
    if (!window.confirm(`Hủy yêu cầu bảo trì cho ${item.device_asset_code}?`)) return
    setCancellingID(item.id); setError('')
    try { await cancelMaintenanceRequest(accessToken, item.id); await load() }
    catch (caught) { handleError(caught, 'Không thể hủy yêu cầu bảo trì.', setError, onSessionExpired) }
    finally { setCancellingID('') }
  }
  function openModal() { setDeviceID(availableDevices[0]?.id ?? ''); setIsModalOpen(true) }
  function closeModal() { if (isSubmitting) return; setIsModalOpen(false); setDeviceID(''); setCategory('hardware'); setPriority('normal'); setTitle(''); setDescription('') }

  const selectedDevice = devices.find((device) => device.id === deviceID)
  const openCount = items.filter((item) => item.status === 'pending' || item.status === 'in_progress').length
  const openDeviceIDs = new Set(items.filter((item) => item.status === 'pending' || item.status === 'in_progress').map((item) => item.device_id))
  const availableDevices = devices.filter((device) => !openDeviceIDs.has(device.id))
  return <>
    <section className="employee-panel employee-maintenance-panel" id="my-maintenance-requests">
      <header><div><span><Icon name="settings" /></span><div><h2>Yêu cầu bảo trì</h2><p>Báo sự cố thiết bị và theo dõi quá trình xử lý</p></div></div><div className="employee-request-header-actions"><span>{openCount} đang mở</span><button type="button" onClick={openModal} disabled={!availableDevices.length}><Icon name="plus" />Báo sự cố</button></div></header>
      {error && <div className="employee-request-error" role="alert"><Icon name="alert" />{error}</div>}
      {isLoading ? <div className="employee-request-loading"><span /><span /></div> : items.length ? <div className="employee-maintenance-list">{items.map((item) => <article className="employee-maintenance-card" key={item.id}>
        <div className="employee-maintenance-device"><span><Icon name="device" /></span><div><strong>{item.device_name}</strong><small>{item.device_asset_code} · Serial: {item.device_serial_number || 'Chưa cập nhật'}</small><p>{[item.device_manufacturer, item.device_model].filter(Boolean).join(' · ') || item.device_type}</p><dl className="employee-maintenance-history-snapshot"><div><dt>Loại</dt><dd>{item.device_type}</dd></div><div><dt>Ngày mua</dt><dd>{formatDate(item.device_purchased_at)}</dd></div><div><dt>Bảo hành</dt><dd>{formatDate(item.device_warranty_expires_at)}</dd></div></dl></div></div>
        <div className="employee-maintenance-problem"><div><span className={`employee-maintenance-priority ${item.priority}`}>{priorityLabel(item.priority)}</span><span className={`request-status status-${item.status}`}>{maintenanceStatusLabel(item.status)}</span></div><strong>{maintenanceCategoryLabel(item.category)} · {item.title}</strong><p>{item.description}</p><small>Gửi lúc {formatDateTime(item.created_at)}</small></div>
        <div className="employee-maintenance-result">{item.assigned_to_name && <strong>Phụ trách: {item.assigned_to_name}</strong>}{item.response_note && <p>{item.response_note}</p>}{item.status === 'pending' && <button type="button" disabled={cancellingID === item.id} onClick={() => void cancel(item)}>{cancellingID === item.id ? 'Đang hủy...' : 'Hủy yêu cầu'}</button>}{item.status === 'in_progress' && <span>IT đang kiểm tra thiết bị của bạn.</span>}{item.status === 'completed' && <span>Yêu cầu đã được hoàn thành.</span>}{item.status === 'rejected' && <span>Yêu cầu đã bị từ chối.</span>}{item.status === 'cancelled' && <span>Bạn đã hủy yêu cầu này.</span>}<LifecycleTimes item={item} /></div>
      </article>)}</div> : <div className="employee-request-empty"><Icon name="settings" /><strong>Chưa có yêu cầu bảo trì</strong><p>Khi thiết bị gặp sự cố, bạn có thể gửi đầy đủ thông tin tới bộ phận IT tại đây.</p><button type="button" onClick={openModal} disabled={!availableDevices.length}>{availableDevices.length ? 'Báo sự cố đầu tiên' : 'Chưa có thiết bị có thể yêu cầu'}</button></div>}
    </section>
    {isModalOpen && <div className="employee-request-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeModal() }}><section className="employee-request-dialog employee-maintenance-dialog" role="dialog" aria-modal="true" aria-labelledby="employee-maintenance-title"><button className="employee-request-close" type="button" onClick={closeModal} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button><span className="employee-request-dialog-icon"><Icon name="settings" /></span><h2 id="employee-maintenance-title">Báo sự cố thiết bị</h2><p>Thông tin thiết bị được lưu tại thời điểm gửi để IT dễ kiểm tra.</p>{error && <div className="employee-request-error" role="alert"><Icon name="alert" />{error}</div>}<form onSubmit={submit}>
      <label htmlFor="maintenance-device">Thiết bị</label><select id="maintenance-device" value={deviceID} onChange={(event) => setDeviceID(event.target.value)} disabled={isSubmitting} required><option value="">Chọn thiết bị — không được bỏ trống</option>{availableDevices.map((device) => <option value={device.id} key={device.id}>{device.asset_code} · {device.name}</option>)}</select>
      {selectedDevice && <dl className="employee-maintenance-snapshot"><div><dt>Serial</dt><dd>{selectedDevice.serial_number || 'Chưa cập nhật'}</dd></div><div><dt>Loại thiết bị</dt><dd>{selectedDevice.device_type}</dd></div><div><dt>Hãng / model</dt><dd>{[selectedDevice.manufacturer, selectedDevice.model].filter(Boolean).join(' · ') || 'Chưa cập nhật'}</dd></div><div><dt>Ngày mua</dt><dd>{formatDate(selectedDevice.purchased_at)}</dd></div><div><dt>Bảo hành đến</dt><dd>{formatDate(selectedDevice.warranty_expires_at)}</dd></div></dl>}
      <div className="employee-maintenance-form-grid"><label>Nhóm sự cố<select value={category} onChange={(event) => setCategory(event.target.value as MaintenanceCategory)} disabled={isSubmitting}><option value="hardware">Phần cứng</option><option value="software">Phần mềm</option><option value="network">Mạng</option><option value="accessory">Phụ kiện</option><option value="other">Khác</option></select></label><label>Mức ưu tiên<select value={priority} onChange={(event) => setPriority(event.target.value as MaintenancePriority)} disabled={isSubmitting}><option value="normal">Bình thường</option><option value="high">Cao</option><option value="urgent">Khẩn cấp</option></select></label></div>
      <label htmlFor="maintenance-title">Tiêu đề sự cố</label><input id="maintenance-title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Nhập tiêu đề — không được bỏ trống" maxLength={200} disabled={isSubmitting} required />
      <label htmlFor="maintenance-description">Mô tả chi tiết</label><textarea id="maintenance-description" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Mô tả biểu hiện, thời điểm và ảnh hưởng — không được bỏ trống" rows={4} disabled={isSubmitting} required />
      <div className="employee-request-dialog-actions"><button type="button" onClick={closeModal} disabled={isSubmitting}>Hủy</button><button type="submit" disabled={isSubmitting || !deviceID || !title.trim() || !description.trim()}>{isSubmitting ? 'Đang gửi...' : 'Gửi yêu cầu bảo trì'}</button></div>
    </form></section></div>}
  </>
}

function LifecycleTimes({ item }: { item: MaintenanceRequestItem }) {
  const values = [
    item.accepted_at && `Tiếp nhận ${formatDateTime(item.accepted_at)}`,
    item.completed_at && `Hoàn thành ${formatDateTime(item.completed_at)}`,
    item.rejected_at && `Từ chối ${formatDateTime(item.rejected_at)}`,
    item.cancelled_at && `Hủy ${formatDateTime(item.cancelled_at)}`,
  ].filter(Boolean)
  return values.length ? <div className="employee-maintenance-times">{values.map((value) => <small key={value as string}>{value}</small>)}</div> : null
}

function handleError(error: unknown, fallback: string, setError: (value: string) => void, onSessionExpired: () => void) { if (error instanceof MaintenanceAPIError) { if (error.status === 401) onSessionExpired(); setError(error.code === 'open_maintenance_request_exists' ? 'Thiết bị này đã có một yêu cầu bảo trì đang mở.' : error.message); return } setError(error instanceof Error ? error.message : fallback) }
function priorityLabel(value: MaintenancePriority) { return value === 'urgent' ? 'Khẩn cấp' : value === 'high' ? 'Cao' : 'Bình thường' }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(date) }
function formatDate(value?: string) { if (!value) return 'Chưa cập nhật'; const date = new Date(`${value}T00:00:00`); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN').format(date) }
