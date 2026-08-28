import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Icon } from '../../components/layout/AdminShell'
import {
  cancelLicenseRequest,
  createLicenseRequest,
  LicenseRequestAPIError,
  listMyLicenseRequests,
  listRequestableSoftware,
  type LicenseRequestItem,
  type LicenseRequestPriority,
} from '../../lib/license-request-api'
import type { SoftwareProduct } from '../../lib/software-api'
import {
  rejectReasonLabel,
  requestPriorityLabel,
  requestStatusLabel,
} from '../requests/request-view-model'

interface EmployeeLicenseRequestsProps {
  accessToken: string
  onSessionExpired: () => void
}

export function EmployeeLicenseRequests({ accessToken, onSessionExpired }: EmployeeLicenseRequestsProps) {
  const [items, setItems] = useState<LicenseRequestItem[]>([])
  const [software, setSoftware] = useState<SoftwareProduct[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [softwareProductID, setSoftwareProductID] = useState('')
  const [priority, setPriority] = useState<LicenseRequestPriority>('normal')
  const [reason, setReason] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [cancellingID, setCancellingID] = useState('')

  const load = useCallback(async () => {
    setIsLoading(true)
    setError('')
    try {
      const [requestResult, softwareResult] = await Promise.all([
        listMyLicenseRequests(accessToken),
        listRequestableSoftware(accessToken),
      ])
      setItems(requestResult.items)
      setSoftware(softwareResult.items)
    } catch (caughtError) {
      handleError(caughtError, 'Không thể tải yêu cầu license.', setError, onSessionExpired)
    } finally {
      setIsLoading(false)
    }
  }, [accessToken, onSessionExpired])

  useEffect(() => { void load() }, [load])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!softwareProductID || !reason.trim()) return
    setIsSubmitting(true)
    setError('')
    try {
      await createLicenseRequest(accessToken, {
        software_product_id: softwareProductID,
        priority,
        reason: reason.trim(),
      })
      closeModal()
      await load()
    } catch (caughtError) {
      handleError(caughtError, 'Không thể tạo yêu cầu license.', setError, onSessionExpired)
    } finally {
      setIsSubmitting(false)
    }
  }

  async function cancel(item: LicenseRequestItem) {
    if (!window.confirm(`Hủy yêu cầu ${item.software_product_name}?`)) return
    setCancellingID(item.id)
    setError('')
    try {
      const cancelled = await cancelLicenseRequest(accessToken, item.id)
      setItems((current) => current.map((request) => request.id === cancelled.id ? cancelled : request))
    } catch (caughtError) {
      handleError(caughtError, 'Không thể hủy yêu cầu.', setError, onSessionExpired)
    } finally {
      setCancellingID('')
    }
  }

  function closeModal() {
    if (isSubmitting) return
    setIsModalOpen(false)
    setSoftwareProductID('')
    setPriority('normal')
    setReason('')
  }

  return <>
    <section className="employee-panel employee-request-panel" id="my-license-requests">
      <header>
        <div><span><Icon name="assignment" /></span><div><h2>Yêu cầu license</h2><p>Gửi nhu cầu phần mềm và theo dõi phản hồi từ IT</p></div></div>
        <div className="employee-request-header-actions">
          <span>{items.filter((item) => item.status === 'pending').length} đang chờ</span>
          <button type="button" onClick={() => setIsModalOpen(true)}><Icon name="plus" />Tạo yêu cầu</button>
        </div>
      </header>
      {error && <div className="employee-request-error" role="alert"><Icon name="alert" />{error}</div>}
      {isLoading ? <PortalRequestLoading /> : items.length ? <div className="employee-request-list">
        {items.map((item) => <article className="employee-request-card" key={item.id}>
          <div className="employee-request-main">
            <div className="employee-request-title"><strong>{item.software_product_name}</strong><span className={`request-status status-${item.status}`}>{requestStatusLabel(item.status)}</span></div>
            <p>{item.reason}</p>
            <div className="employee-request-meta"><span>Ưu tiên: <strong>{requestPriorityLabel(item.priority)}</strong></span><span>Gửi lúc {formatDateTime(item.created_at)}</span></div>
          </div>
          <div className="employee-request-result">
            {item.status === 'approved' && <><strong>{item.selected_license_name || 'License đã được cấp'}</strong><span>{item.response_note || 'IT đã duyệt và cấp license cho bạn.'}</span></>}
            {item.status === 'rejected' && <><strong>{item.decision_reason ? rejectReasonLabel(item.decision_reason) : 'Đã từ chối'}</strong><span>{item.response_note}</span></>}
            {item.status === 'cancelled' && <span>Bạn đã hủy yêu cầu này.</span>}
            {item.status === 'pending' && <button type="button" disabled={cancellingID === item.id} onClick={() => void cancel(item)}>{cancellingID === item.id ? 'Đang hủy...' : 'Hủy yêu cầu'}</button>}
          </div>
        </article>)}
      </div> : <div className="employee-request-empty"><Icon name="assignment" /><strong>Chưa có yêu cầu license</strong><p>Khi cần thêm phần mềm cho công việc, hãy gửi yêu cầu để bộ phận IT xử lý.</p><button type="button" onClick={() => setIsModalOpen(true)}>Tạo yêu cầu đầu tiên</button></div>}
    </section>

    {isModalOpen && <div className="employee-request-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeModal() }}>
      <section className="employee-request-dialog" role="dialog" aria-modal="true" aria-labelledby="employee-request-title">
        <button className="employee-request-close" type="button" onClick={closeModal} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button>
        <span className="employee-request-dialog-icon"><Icon name="assignment" /></span>
        <h2 id="employee-request-title">Tạo yêu cầu cấp license</h2>
        <p>Cho bộ phận IT biết phần mềm và mục đích bạn cần sử dụng.</p>
        {error && <div className="employee-request-error" role="alert"><Icon name="alert" />{error}</div>}
        <form onSubmit={submit}>
          <label htmlFor="request-software">Phần mềm</label>
          <select id="request-software" value={softwareProductID} onChange={(event) => setSoftwareProductID(event.target.value)} disabled={isSubmitting} required>
            <option value="">Chọn phần mềm — không được bỏ trống</option>
            {software.map((product) => <option value={product.id} key={product.id}>{product.name} · {product.publisher}</option>)}
          </select>
          <label htmlFor="request-priority">Mức ưu tiên</label>
          <select id="request-priority" value={priority} onChange={(event) => setPriority(event.target.value as LicenseRequestPriority)} disabled={isSubmitting}>
            <option value="normal">Bình thường</option><option value="high">Cao</option><option value="urgent">Khẩn cấp</option>
          </select>
          <label htmlFor="request-reason">Lý do sử dụng</label>
          <textarea id="request-reason" value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Nhập lý do sử dụng — không được bỏ trống" rows={4} disabled={isSubmitting} required />
          <div className="employee-request-dialog-actions"><button type="button" onClick={closeModal} disabled={isSubmitting}>Hủy</button><button type="submit" disabled={isSubmitting || !softwareProductID || !reason.trim()}>{isSubmitting ? 'Đang gửi...' : 'Gửi yêu cầu'}</button></div>
        </form>
      </section>
    </div>}
  </>
}

function PortalRequestLoading() {
  return <div className="employee-request-loading" aria-label="Đang tải yêu cầu"><span /><span /></div>
}

function handleError(error: unknown, fallback: string, setError: (value: string) => void, onSessionExpired: () => void) {
  if (error instanceof LicenseRequestAPIError) {
    if (error.status === 401) onSessionExpired()
    setError(error.message)
    return
  }
  setError(error instanceof Error ? error.message : fallback)
}

function formatDateTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(date)
}
