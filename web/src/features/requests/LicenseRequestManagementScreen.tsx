import { useEffect, useMemo, useState } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import type { AuthSession } from '../../lib/auth-api'
import { getLicenses, type LicenseItem } from '../../lib/license-api'
import {
  approveLicenseRequest,
  LicenseRequestAPIError,
  listLicenseRequests,
  rejectLicenseRequest,
  type LicenseRequestDecisionReason,
  type LicenseRequestItem,
  type LicenseRequestPriority,
  type LicenseRequestStatus,
} from '../../lib/license-request-api'
import {
  eligibleLicenses,
  rejectReasonLabel,
  requestPriorityLabel,
  requestStatusLabel,
} from './request-view-model'
import './LicenseRequestManagementScreen.css'

interface LicenseRequestManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type StatusFilter = LicenseRequestStatus | 'all'
type PriorityFilter = LicenseRequestPriority | 'all'

interface ApproveDialogState {
  request: LicenseRequestItem
  licenseID: string
  responseNote: string
  loading: boolean
  error: string
}

interface RejectDialogState {
  request: LicenseRequestItem
  decisionReason: LicenseRequestDecisionReason
  responseNote: string
  loading: boolean
  error: string
}

export function LicenseRequestManagementScreen({ session, onNavigate, onLogout }: LicenseRequestManagementScreenProps) {
  const [requests, setRequests] = useState<LicenseRequestItem[]>([])
  const [licenses, setLicenses] = useState<LicenseItem[]>([])
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilter>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<LicenseRequestAPIError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [successMessage, setSuccessMessage] = useState('')
  const [approveDialog, setApproveDialog] = useState<ApproveDialogState | null>(null)
  const [rejectDialog, setRejectDialog] = useState<RejectDialogState | null>(null)

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)
    Promise.all([
      listLicenseRequests(session.tokens.access_token),
      getLicenses(session.tokens.access_token),
    ])
      .then(([requestResult, licenseResult]) => {
        if (cancelled) return
        setRequests(requestResult.items)
        setLicenses(licenseResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        setError(caughtError instanceof LicenseRequestAPIError
          ? caughtError
          : new LicenseRequestAPIError(caughtError instanceof Error ? caughtError.message : 'Không thể tải yêu cầu license.', 0))
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })
    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const filteredRequests = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return requests.filter((item) => {
      const matchesSearch = !query || [item.requester_name, item.software_product_name, item.reason]
        .some((value) => value.toLocaleLowerCase('vi').includes(query))
      const matchesStatus = statusFilter === 'all' || item.status === statusFilter
      const matchesPriority = priorityFilter === 'all' || item.priority === priorityFilter
      return matchesSearch && matchesStatus && matchesPriority
    })
  }, [priorityFilter, requests, search, statusFilter])

  const counts = useMemo(() => ({
    pending: requests.filter((item) => item.status === 'pending').length,
    approved: requests.filter((item) => item.status === 'approved').length,
    rejected: requests.filter((item) => item.status === 'rejected').length,
    urgent: requests.filter((item) => item.status === 'pending' && item.priority === 'urgent').length,
  }), [requests])

  function openApprove(item: LicenseRequestItem) {
    const available = eligibleLicenses(licenses, item.software_product_id)
    setApproveDialog({ request: item, licenseID: available[0]?.id ?? '', responseNote: '', loading: false, error: '' })
  }

  function openReject(item: LicenseRequestItem, reason: LicenseRequestDecisionReason = 'not_approved') {
    setRejectDialog({ request: item, decisionReason: reason, responseNote: '', loading: false, error: '' })
  }

  async function confirmApprove() {
    if (!approveDialog?.licenseID) return
    const current = approveDialog
    setApproveDialog({ ...current, loading: true, error: '' })
    try {
      const approved = await approveLicenseRequest(session.tokens.access_token, current.request.id, {
        license_id: current.licenseID,
        response_note: current.responseNote.trim(),
      })
      setApproveDialog(null)
      setSuccessMessage(`Đã duyệt ${approved.software_product_name} cho ${approved.requester_name}.`)
      setReloadKey((value) => value + 1)
    } catch (caughtError) {
      let message = caughtError instanceof Error ? caughtError.message : 'Không thể duyệt yêu cầu.'
      if (caughtError instanceof LicenseRequestAPIError && caughtError.status === 409) {
        message = 'License vừa hết seat. Hãy từ chối với lý do Tạm hết license để phản hồi nhân viên.'
      }
      setApproveDialog({ ...current, loading: false, error: message })
    }
  }

  async function confirmReject() {
    if (!rejectDialog?.responseNote.trim()) return
    const current = rejectDialog
    setRejectDialog({ ...current, loading: true, error: '' })
    try {
      const rejected = await rejectLicenseRequest(session.tokens.access_token, current.request.id, {
        decision_reason: current.decisionReason,
        response_note: current.responseNote.trim(),
      })
      setRejectDialog(null)
      setSuccessMessage(`Đã gửi phản hồi cho ${rejected.requester_name}.`)
      setReloadKey((value) => value + 1)
    } catch (caughtError) {
      setRejectDialog({ ...current, loading: false, error: caughtError instanceof Error ? caughtError.message : 'Không thể từ chối yêu cầu.' })
    }
  }

  const headerActions = <button className="request-refresh" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}><Icon name="refresh" /><span>Làm mới</span></button>

  return <AdminShell session={session} activePage="requests" title="Quản lý yêu cầu" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
    <div className="request-admin-page">
      <section className="request-admin-heading"><div><h2>Yêu cầu cấp license</h2><p>Tiếp nhận nhu cầu phần mềm, duyệt cấp phát hoặc gửi phản hồi cho nhân viên.</p></div><span>{counts.pending} yêu cầu đang chờ</span></section>

      {successMessage && <div className="request-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

      <section className="request-admin-stats">
        <RequestStat label="Đang chờ" value={counts.pending} detail="cần được xử lý" icon="request" tone="blue" loading={isLoading} />
        <RequestStat label="Khẩn cấp" value={counts.urgent} detail="trong hàng chờ" icon="alert" tone="amber" loading={isLoading} />
        <RequestStat label="Đã duyệt" value={counts.approved} detail="đã tạo cấp phát" icon="check" tone="green" loading={isLoading} />
        <RequestStat label="Đã từ chối" value={counts.rejected} detail="đã gửi phản hồi" icon="close" tone="red" loading={isLoading} />
      </section>

      <section className="request-admin-card">
        <div className="request-admin-toolbar">
          <div className="request-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm nhân viên, phần mềm, lý do..." aria-label="Tìm kiếm yêu cầu" /></div>
          <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value as StatusFilter)} aria-label="Lọc trạng thái"><option value="all">Tất cả trạng thái</option><option value="pending">Đang chờ</option><option value="approved">Đã duyệt</option><option value="rejected">Đã từ chối</option><option value="cancelled">Đã hủy</option></select>
          <select value={priorityFilter} onChange={(event) => setPriorityFilter(event.target.value as PriorityFilter)} aria-label="Lọc ưu tiên"><option value="all">Tất cả ưu tiên</option><option value="normal">Bình thường</option><option value="high">Cao</option><option value="urgent">Khẩn cấp</option></select>
          <span>{filteredRequests.length} kết quả</span>
        </div>

        {error ? <RequestAdminError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? <div className="request-admin-loading" aria-label="Đang tải yêu cầu">{Array.from({ length: 5 }, (_, index) => <span key={index} />)}</div> : filteredRequests.length ? <>
          <div className="request-table-wrap"><table className="request-table"><thead><tr><th>Nhân viên</th><th>Phần mềm và lý do</th><th>Ưu tiên</th><th>Thời gian</th><th>Trạng thái</th><th>Thao tác</th></tr></thead><tbody>{filteredRequests.map((item) => <RequestRow item={item} onApprove={openApprove} onReject={openReject} key={item.id} />)}</tbody></table></div>
          <div className="request-mobile-list">{filteredRequests.map((item) => <RequestMobileCard item={item} onApprove={openApprove} onReject={openReject} key={item.id} />)}</div>
        </> : <div className="request-admin-empty"><Icon name="request" /><strong>Không tìm thấy yêu cầu</strong><p>Thử thay đổi từ khóa hoặc bộ lọc hiện tại.</p></div>}
      </section>
    </div>

    {approveDialog && <ApproveDialog state={approveDialog} licenses={eligibleLicenses(licenses, approveDialog.request.software_product_id)} onChange={setApproveDialog} onClose={() => { if (!approveDialog.loading) setApproveDialog(null) }} onConfirm={() => void confirmApprove()} />}
    {rejectDialog && <RejectDialog state={rejectDialog} onChange={setRejectDialog} onClose={() => { if (!rejectDialog.loading) setRejectDialog(null) }} onConfirm={() => void confirmReject()} />}
  </AdminShell>
}

function RequestRow({ item, onApprove, onReject }: { item: LicenseRequestItem; onApprove: (item: LicenseRequestItem) => void; onReject: (item: LicenseRequestItem, reason?: LicenseRequestDecisionReason) => void }) {
  return <tr><td><div className="request-user"><span>{initials(item.requester_name)}</span><div><strong>{item.requester_name}</strong><small>{item.requester_id.slice(0, 8)}</small></div></div></td><td><div className="request-software"><strong>{item.software_product_name}</strong><small>{item.reason}</small>{item.response_note && <em>Phản hồi: {item.response_note}</em>}</div></td><td><span className={`request-priority priority-${item.priority}`}>{requestPriorityLabel(item.priority)}</span></td><td>{formatDateTime(item.created_at)}</td><td><span className={`request-admin-status status-${item.status}`}>{requestStatusLabel(item.status)}</span></td><td><RequestActions item={item} onApprove={onApprove} onReject={onReject} /></td></tr>
}

function RequestMobileCard({ item, onApprove, onReject }: { item: LicenseRequestItem; onApprove: (item: LicenseRequestItem) => void; onReject: (item: LicenseRequestItem, reason?: LicenseRequestDecisionReason) => void }) {
  return <article><header><div><strong>{item.software_product_name}</strong><span>{item.requester_name}</span></div><span className={`request-admin-status status-${item.status}`}>{requestStatusLabel(item.status)}</span></header><p>{item.reason}</p><div className="request-mobile-meta"><span>{requestPriorityLabel(item.priority)}</span><span>{formatDateTime(item.created_at)}</span></div>{item.response_note && <small>Phản hồi: {item.response_note}</small>}<RequestActions item={item} onApprove={onApprove} onReject={onReject} /></article>
}

function RequestActions({ item, onApprove, onReject }: { item: LicenseRequestItem; onApprove: (item: LicenseRequestItem) => void; onReject: (item: LicenseRequestItem, reason?: LicenseRequestDecisionReason) => void }) {
  if (item.status !== 'pending') return <span className="request-no-action">Đã xử lý</span>
  return <div className="request-row-actions"><button className="approve" type="button" onClick={() => onApprove(item)}>Duyệt</button><button type="button" onClick={() => onReject(item)}>Từ chối</button><button className="out-of-stock" type="button" onClick={() => onReject(item, 'out_of_stock')}>Tạm hết</button></div>
}

function ApproveDialog({ state, licenses, onChange, onClose, onConfirm }: { state: ApproveDialogState; licenses: LicenseItem[]; onChange: (state: ApproveDialogState) => void; onClose: () => void; onConfirm: () => void }) {
  return <div className="request-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose() }}><section className="request-decision-dialog" role="dialog" aria-modal="true" aria-labelledby="approve-request-title"><button className="request-dialog-close" type="button" onClick={onClose} disabled={state.loading} aria-label="Đóng"><Icon name="close" /></button><span className="decision-icon approve"><Icon name="check" /></span><h2 id="approve-request-title">Duyệt yêu cầu license</h2><p><strong>{state.request.requester_name}</strong> đang yêu cầu <strong>{state.request.software_product_name}</strong>.</p>{licenses.length ? <><label>License cấp cho nhân viên<select value={state.licenseID} onChange={(event) => onChange({ ...state, licenseID: event.target.value, error: '' })} disabled={state.loading}>{licenses.map((license) => <option value={license.id} key={license.id}>{license.name} — còn {license.available_seats}/{license.seat_count} seat</option>)}</select></label><label>Phản hồi<textarea rows={3} value={state.responseNote} onChange={(event) => onChange({ ...state, responseNote: event.target.value })} placeholder="Ghi chú cho nhân viên (không bắt buộc)" disabled={state.loading} /></label></> : <div className="request-no-license"><Icon name="alert" /><div><strong>Không có license phù hợp còn seat</strong><p>Hãy đóng hộp thoại và dùng “Tạm hết” để gửi phản hồi cho nhân viên.</p></div></div>}{state.error && <div className="request-dialog-error" role="alert"><Icon name="alert" />{state.error}</div>}<footer><button type="button" onClick={onClose} disabled={state.loading}>Hủy</button><button className="confirm-approve" type="button" onClick={onConfirm} disabled={state.loading || !state.licenseID}>{state.loading ? 'Đang duyệt...' : 'Duyệt và cấp license'}</button></footer></section></div>
}

function RejectDialog({ state, onChange, onClose, onConfirm }: { state: RejectDialogState; onChange: (state: RejectDialogState) => void; onClose: () => void; onConfirm: () => void }) {
  return <div className="request-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose() }}><section className="request-decision-dialog" role="dialog" aria-modal="true" aria-labelledby="reject-request-title"><button className="request-dialog-close" type="button" onClick={onClose} disabled={state.loading} aria-label="Đóng"><Icon name="close" /></button><span className="decision-icon reject"><Icon name="close" /></span><h2 id="reject-request-title">Gửi phản hồi từ chối</h2><p><strong>{state.request.requester_name}</strong> sẽ nhận thông báo ngay trong website.</p><label>Lý do<select value={state.decisionReason} onChange={(event) => onChange({ ...state, decisionReason: event.target.value as LicenseRequestDecisionReason, error: '' })} disabled={state.loading}><option value="out_of_stock">Tạm hết license</option><option value="not_approved">Không được phê duyệt</option><option value="other">Lý do khác</option></select></label><label>Phản hồi<textarea rows={4} value={state.responseNote} onChange={(event) => onChange({ ...state, responseNote: event.target.value, error: '' })} placeholder="Nhập phản hồi — không được bỏ trống" disabled={state.loading} /></label>{state.error && <div className="request-dialog-error" role="alert"><Icon name="alert" />{state.error}</div>}<footer><button type="button" onClick={onClose} disabled={state.loading}>Hủy</button><button className="confirm-reject" type="button" onClick={onConfirm} disabled={state.loading || !state.responseNote.trim()}>{state.loading ? 'Đang gửi...' : `Từ chối · ${rejectReasonLabel(state.decisionReason)}`}</button></footer></section></div>
}

function RequestStat({ label, value, detail, icon, tone, loading }: { label: string; value: number; detail: string; icon: 'request' | 'alert' | 'check' | 'close'; tone: string; loading: boolean }) {
  return <article className="request-admin-stat"><span className={tone}><Icon name={icon} /></span><div><p>{label}</p>{loading ? <i /> : <strong>{value}</strong>}<small>{detail}</small></div></article>
}

function RequestAdminError({ error, onRetry, onLogout }: { error: LicenseRequestAPIError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  return <div className="request-admin-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : 'Không thể tải yêu cầu'}</strong><p>{error.message}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function initials(name: string) { return name.split(/\s+/).filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || 'NV' }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(date) }
