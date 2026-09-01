import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import { notifyAdminRequestBadgesChanged } from '../../components/layout/use-admin-request-badges'
import type { AuthSession } from '../../lib/auth-api'
import {
  acceptMaintenanceRequest,
  completeMaintenanceRequest,
  listMaintenanceRequests,
  MaintenanceAPIError,
  rejectMaintenanceRequest,
  type MaintenanceCategory,
  type MaintenancePriority,
  type MaintenanceRequestItem,
  type MaintenanceStatus,
} from '../../lib/maintenance-api'
import { availableMaintenanceActions, maintenanceCategoryLabel, maintenanceStatusLabel } from './maintenance-view-model'
import './MaintenanceManagementScreen.css'

interface Props {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type DialogAction = 'complete' | 'reject'

export function MaintenanceManagementScreen({ session, onNavigate, onLogout }: Props) {
  const [items, setItems] = useState<MaintenanceRequestItem[]>([])
  const [summaryItems, setSummaryItems] = useState<MaintenanceRequestItem[]>([])
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<MaintenanceStatus | ''>('')
  const [priority, setPriority] = useState<MaintenancePriority | ''>('')
  const [category, setCategory] = useState<MaintenanceCategory | ''>('')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<MaintenanceAPIError | null>(null)
  const [busyID, setBusyID] = useState('')
  const [dialog, setDialog] = useState<{ item: MaintenanceRequestItem; action: DialogAction; note: string; error: string } | null>(null)
  const requestSequence = useRef(0)
  const invalidatePendingRequests = useCallback(() => { requestSequence.current++ }, [])

  const load = useCallback(async () => {
    const sequence = ++requestSequence.current
    setIsLoading(true)
    setError(null)
    try {
      const [result, summary] = await Promise.all([
        listMaintenanceRequests(session.tokens.access_token, { status, priority, category, search }),
        listMaintenanceRequests(session.tokens.access_token),
      ])
      if (sequence !== requestSequence.current) return
      setItems(result.items)
      setSummaryItems(summary.items)
    } catch (caught) {
      if (sequence !== requestSequence.current) return
      setError(caught instanceof MaintenanceAPIError ? caught : new MaintenanceAPIError('Không thể tải yêu cầu bảo trì.', 0))
    } finally {
      if (sequence === requestSequence.current) setIsLoading(false)
    }
  }, [category, priority, search, session.tokens.access_token, status])

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 250)
    return () => { window.clearTimeout(timer); invalidatePendingRequests() }
  }, [invalidatePendingRequests, load])

  const counts = useMemo(() => ({
    pending: summaryItems.filter((item) => item.status === 'pending').length,
    inProgress: summaryItems.filter((item) => item.status === 'in_progress').length,
    completed: summaryItems.filter((item) => item.status === 'completed').length,
    urgent: summaryItems.filter((item) => item.priority === 'urgent' && ['pending', 'in_progress'].includes(item.status)).length,
  }), [summaryItems])

  async function accept(item: MaintenanceRequestItem) {
    setBusyID(item.id)
    try {
      await acceptMaintenanceRequest(session.tokens.access_token, item.id)
      await load()
      notifyAdminRequestBadgesChanged()
    } catch (caught) {
      setError(caught instanceof MaintenanceAPIError ? caught : new MaintenanceAPIError('Không thể tiếp nhận yêu cầu.', 0))
    } finally {
      setBusyID('')
    }
  }

  async function submitDecision() {
    if (!dialog?.note.trim()) return
    const current = dialog
    setBusyID(current.item.id)
    try {
      if (current.action === 'complete') await completeMaintenanceRequest(session.tokens.access_token, current.item.id, current.note.trim())
      else await rejectMaintenanceRequest(session.tokens.access_token, current.item.id, current.note.trim())
      setDialog(null)
      await load()
      notifyAdminRequestBadgesChanged()
    } catch (caught) {
      if (caught instanceof MaintenanceAPIError && caught.status === 401) {
        setDialog(null)
        setError(caught)
      } else {
        const message = caught instanceof Error ? caught.message : 'Không thể xử lý yêu cầu.'
        setDialog({ ...current, error: message })
      }
    } finally {
      setBusyID('')
    }
  }

  return <AdminShell session={session} activePage="maintenance" title="Yêu cầu bảo trì" onNavigate={onNavigate} onLogout={onLogout} actions={<button className="maintenance-refresh" type="button" onClick={() => void load()} disabled={isLoading}><Icon name="refresh" />Làm mới</button>}>
    <div className="maintenance-page">
      <section className="maintenance-stats" aria-label="Tổng quan yêu cầu bảo trì">
        <Stat label="Chờ tiếp nhận" value={counts.pending} icon="alert" tone="amber" />
        <Stat label="Đang xử lý" value={counts.inProgress} icon="settings" tone="violet" />
        <Stat label="Đã hoàn thành" value={counts.completed} icon="check" tone="blue" />
        <Stat label="Khẩn cấp đang mở" value={counts.urgent} icon="alert" tone="red" />
      </section>

      <section className="maintenance-card">
        <div className="maintenance-toolbar">
          <label className="maintenance-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm nhân viên, asset, serial, tiêu đề..." /></label>
          <select value={status} onChange={(event) => setStatus(event.target.value as MaintenanceStatus | '')}><option value="">Tất cả trạng thái</option><option value="pending">Chờ tiếp nhận</option><option value="in_progress">Đang xử lý</option><option value="completed">Hoàn thành</option><option value="rejected">Đã từ chối</option><option value="cancelled">Đã hủy</option></select>
          <select value={priority} onChange={(event) => setPriority(event.target.value as MaintenancePriority | '')}><option value="">Tất cả ưu tiên</option><option value="normal">Bình thường</option><option value="high">Cao</option><option value="urgent">Khẩn cấp</option></select>
          <select value={category} onChange={(event) => setCategory(event.target.value as MaintenanceCategory | '')}><option value="">Tất cả nhóm lỗi</option><option value="hardware">Phần cứng</option><option value="software">Phần mềm</option><option value="network">Mạng</option><option value="accessory">Phụ kiện</option><option value="other">Khác</option></select>
        </div>

        {error ? <div className="maintenance-error"><Icon name="alert" /><div><strong>{error.status === 401 ? 'Phiên đăng nhập đã hết hạn' : 'Không thể tải dữ liệu'}</strong><p>{error.message}</p></div><button type="button" onClick={error.status === 401 ? onLogout : () => void load()}>{error.status === 401 ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
          : isLoading ? <div className="maintenance-loading"><span /><span /><span /></div>
            : items.length ? <div className="maintenance-list">{items.map((item) => <MaintenanceCard key={item.id} item={item} busy={busyID === item.id} onAccept={() => void accept(item)} onComplete={() => setDialog({ item, action: 'complete', note: '', error: '' })} onReject={() => setDialog({ item, action: 'reject', note: '', error: '' })} />)}</div>
              : <div className="maintenance-empty"><Icon name="settings" /><strong>Không tìm thấy yêu cầu bảo trì</strong><p>Hãy thử thay đổi từ khóa hoặc bộ lọc.</p></div>}
      </section>
    </div>

    {dialog && <div className="maintenance-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !busyID) setDialog(null) }}><section className="maintenance-dialog" role="dialog" aria-modal="true" aria-labelledby="maintenance-dialog-title"><button className="maintenance-dialog-close" type="button" onClick={() => setDialog(null)} disabled={Boolean(busyID)}><Icon name="close" /></button><span className={`maintenance-dialog-icon ${dialog.action}`}><Icon name={dialog.action === 'complete' ? 'check' : 'close'} /></span><h2 id="maintenance-dialog-title">{dialog.action === 'complete' ? 'Hoàn thành bảo trì' : 'Từ chối yêu cầu'}</h2><p><strong>{dialog.item.device_asset_code}</strong> · {dialog.item.device_name}<br />Nhân viên sẽ nhận thông báo trên website.</p><label>Phản hồi cho nhân viên<textarea rows={4} value={dialog.note} onChange={(event) => setDialog({ ...dialog, note: event.target.value, error: '' })} placeholder="Nhập kết quả hoặc lý do — không được bỏ trống" disabled={Boolean(busyID)} /></label>{dialog.error && <div className="maintenance-dialog-error"><Icon name="alert" />{dialog.error}</div>}<footer><button type="button" onClick={() => setDialog(null)} disabled={Boolean(busyID)}>Hủy</button><button className={dialog.action} type="button" onClick={() => void submitDecision()} disabled={Boolean(busyID) || !dialog.note.trim()}>{busyID ? 'Đang xử lý...' : dialog.action === 'complete' ? 'Xác nhận hoàn thành' : 'Gửi phản hồi từ chối'}</button></footer></section></div>}
  </AdminShell>
}

function MaintenanceCard({ item, busy, onAccept, onComplete, onReject }: { item: MaintenanceRequestItem; busy: boolean; onAccept: () => void; onComplete: () => void; onReject: () => void }) {
  const actions = availableMaintenanceActions(item.status)
  return <article className="maintenance-item">
    <header><div className="maintenance-requester"><span>{initials(item.requester_name)}</span><div><strong>{item.requester_name}</strong><small>Gửi {formatDateTime(item.created_at)}</small></div></div><div className="maintenance-tags"><span className={`maintenance-priority ${item.priority}`}>{priorityLabel(item.priority)}</span><span className={`maintenance-status ${item.status}`}>{maintenanceStatusLabel(item.status)}</span></div></header>
    <div className="maintenance-item-body"><div className="maintenance-device"><span><Icon name="device" /></span><div><small>Thiết bị</small><strong>{item.device_asset_code} · {item.device_name}</strong><p>{[item.device_manufacturer, item.device_model].filter(Boolean).join(' · ') || item.device_type}</p><dl><div><dt>Serial</dt><dd>{item.device_serial_number || 'Chưa cập nhật'}</dd></div><div><dt>Loại</dt><dd>{item.device_type}</dd></div><div><dt>Mua ngày</dt><dd>{formatDate(item.device_purchased_at)}</dd></div><div><dt>Bảo hành</dt><dd>{formatDate(item.device_warranty_expires_at)}</dd></div></dl></div></div><div className="maintenance-problem"><small>{maintenanceCategoryLabel(item.category)}</small><strong>{item.title}</strong><p>{item.description}</p>{item.response_note && <blockquote><b>Phản hồi:</b> {item.response_note}</blockquote>}{item.assigned_to_name && <span>Phụ trách: <strong>{item.assigned_to_name}</strong></span>}</div></div>
    <footer>{actions.length ? <div>{actions.includes('accept') && <button className="accept" type="button" onClick={onAccept} disabled={busy}>{busy ? 'Đang tiếp nhận...' : 'Tiếp nhận'}</button>}{actions.includes('complete') && <button className="complete" type="button" onClick={onComplete} disabled={busy}>Hoàn thành</button>}{actions.includes('reject') && <button className="reject" type="button" onClick={onReject} disabled={busy}>Từ chối</button>}</div> : <span>Yêu cầu đã kết thúc</span>}</footer>
  </article>
}

function Stat({ label, value, icon, tone }: { label: string; value: number; icon: 'alert' | 'settings' | 'check'; tone: string }) { return <article className="maintenance-stat"><span className={tone}><Icon name={icon} /></span><div><small>{label}</small><strong>{value}</strong></div></article> }
function priorityLabel(value: MaintenancePriority) { return value === 'urgent' ? 'Khẩn cấp' : value === 'high' ? 'Cao' : 'Bình thường' }
function initials(value: string) { return value.split(/\s+/).filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || 'NV' }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(date) }
function formatDate(value?: string) { if (!value) return 'Chưa cập nhật'; const date = new Date(`${value}T00:00:00`); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN').format(date) }
