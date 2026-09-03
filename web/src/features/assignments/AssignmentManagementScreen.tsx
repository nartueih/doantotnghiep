import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import {
  AssignmentAPIError,
  createAssignment,
  getAssignmentDevices,
  getAssignmentLicenses,
  getAssignments,
  getAssignmentUsers,
  revokeAssignment,
  type AssignmentInput,
  type AssignmentItem,
  type DeviceItem,
} from '../../lib/assignment-api'
import type { AuthSession, AuthUser } from '../../lib/auth-api'
import type { LicenseItem } from '../../lib/license-api'
import './AssignmentManagementScreen.css'

interface AssignmentManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type AssignmentFilter = 'all' | 'active' | 'revoked' | 'user' | 'device'

const filterLabels: Record<AssignmentFilter, string> = {
  all: 'Tất cả cấp phát',
  active: 'Đang hoạt động',
  revoked: 'Đã thu hồi',
  user: 'Theo người dùng',
  device: 'Theo thiết bị',
}

export function AssignmentManagementScreen({ session, onNavigate, onLogout }: AssignmentManagementScreenProps) {
  const [assignments, setAssignments] = useState<AssignmentItem[]>([])
  const [licenses, setLicenses] = useState<LicenseItem[]>([])
  const [users, setUsers] = useState<AuthUser[]>([])
  const [devices, setDevices] = useState<DeviceItem[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<AssignmentFilter>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<AssignmentAPIError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [showCreate, setShowCreate] = useState(false)
  const [revokeDialog, setRevokeDialog] = useState<{ assignment: AssignmentItem; loading: boolean; error?: string } | null>(null)
  const [successMessage, setSuccessMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)
    Promise.all([
      getAssignments(session.tokens.access_token),
      getAssignmentLicenses(session.tokens.access_token),
      getAssignmentUsers(session.tokens.access_token),
      getAssignmentDevices(session.tokens.access_token),
    ])
      .then(([assignmentResult, licenseResult, userResult, deviceResult]) => {
        if (cancelled) return
        setAssignments(assignmentResult.items)
        setLicenses(licenseResult.items)
        setUsers(userResult.items)
        setDevices(deviceResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        setError(caughtError instanceof AssignmentAPIError
          ? caughtError
          : new AssignmentAPIError('Đã xảy ra lỗi không mong muốn.', 0))
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })
    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const filteredAssignments = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return assignments.filter((item) => {
      const matchesSearch = !query || [item.license_name, item.target_name, item.assigned_by_name, item.notes]
        .some((value) => value?.toLocaleLowerCase('vi').includes(query))
      return matchesSearch && matchesFilter(item, filter)
    }).sort((left, right) => new Date(right.assigned_at).getTime() - new Date(left.assigned_at).getTime())
  }, [assignments, filter, search])

  const counts = useMemo(() => {
    const active = assignments.filter((item) => item.status === 'active')
    return {
      active: active.length,
      revoked: assignments.length - active.length,
      users: active.filter((item) => Boolean(item.user_id)).length,
      devices: active.filter((item) => Boolean(item.device_id)).length,
    }
  }, [assignments])

  async function saveAssignment(input: AssignmentInput) {
    const created = await createAssignment(session.tokens.access_token, input)
    setShowCreate(false)
    setSuccessMessage(`Đã cấp ${created.license_name} cho ${created.target_name}.`)
    setReloadKey((value) => value + 1)
  }

  async function confirmRevoke() {
    if (!revokeDialog) return
    const assignment = revokeDialog.assignment
    setRevokeDialog({ assignment, loading: true })
    try {
      await revokeAssignment(session.tokens.access_token, assignment.id)
      setRevokeDialog(null)
      setSuccessMessage(`Đã thu hồi ${assignment.license_name} khỏi ${assignment.target_name}.`)
      setReloadKey((value) => value + 1)
    } catch (caughtError) {
      setRevokeDialog({
        assignment,
        loading: false,
        error: translateAssignmentError(caughtError instanceof Error ? caughtError.message : 'Không thể thu hồi cấp phát.'),
      })
    }
  }

  const headerActions = (
    <button className="assignment-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}>
      <Icon name="refresh" /><span>Làm mới</span>
    </button>
  )

  return (
    <AdminShell session={session} activePage="assignments" title="Quản lý cấp phát" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="assignment-page">
        <section className="assignment-page-heading">
          <div><h2>Lịch sử cấp phát</h2><p>Cấp và thu hồi quyền sử dụng license cho người dùng hoặc thiết bị.</p></div>
          <button className="add-assignment-button" type="button" onClick={() => setShowCreate(true)}><Icon name="plus" />Cấp license</button>
        </section>

        {successMessage && <div className="assignment-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="assignment-stats" aria-label="Thống kê cấp phát">
          <AssignmentStat label="Đang hoạt động" value={counts.active} detail="seat đang sử dụng" tone="blue" icon="assignment" loading={isLoading} />
          <AssignmentStat label="Theo người dùng" value={counts.users} detail="cấp phát trực tiếp" tone="green" icon="users" loading={isLoading} />
          <AssignmentStat label="Theo thiết bị" value={counts.devices} detail="gắn với tài sản" tone="violet" icon="device" loading={isLoading} />
          <AssignmentStat label="Đã thu hồi" value={counts.revoked} detail="lịch sử được giữ lại" tone="gray" icon="undo" loading={isLoading} />
        </section>

        <section className="assignment-list-card">
          <div className="assignment-toolbar">
            <div className="assignment-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm license, người dùng, thiết bị..." aria-label="Tìm kiếm cấp phát" /></div>
            <div className="assignment-filter"><Icon name="filter" /><select value={filter} onChange={(event) => setFilter(event.target.value as AssignmentFilter)} aria-label="Lọc cấp phát">{(Object.keys(filterLabels) as AssignmentFilter[]).map((value) => <option value={value} key={value}>{filterLabels[value]}</option>)}</select></div>
            <span className="assignment-result-count">{filteredAssignments.length} kết quả</span>
          </div>

          {error ? <AssignmentError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="assignment-loading" aria-label="Đang tải cấp phát">{Array.from({ length: 6 }, (_, index) => <span key={index} />)}</div>
          ) : filteredAssignments.length === 0 ? (
            <div className="assignment-empty"><span><Icon name="assignment" /></span><strong>Không tìm thấy cấp phát</strong><p>Thử thay đổi từ khóa, bộ lọc hoặc tạo một cấp phát mới.</p></div>
          ) : (
            <div className="assignment-table-scroll"><table className="assignment-table">
              <thead><tr><th>License</th><th>Đối tượng nhận</th><th>Hình thức</th><th>Người cấp</th><th>Thời gian</th><th>Trạng thái</th><th /></tr></thead>
              <tbody>{filteredAssignments.map((item) => <AssignmentRow assignment={item} onRevoke={(assignment) => setRevokeDialog({ assignment, loading: false })} key={item.id} />)}</tbody>
            </table></div>
          )}
        </section>
      </div>

      {showCreate && <CreateAssignmentDialog licenses={licenses} users={users} devices={devices} assignments={assignments} onClose={() => setShowCreate(false)} onSubmit={saveAssignment} />}
      {revokeDialog && (
        <div className="assignment-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!revokeDialog.loading && event.target === event.currentTarget) setRevokeDialog(null) }}>
          <section className="revoke-dialog" role="dialog" aria-modal="true" aria-labelledby="revoke-dialog-title">
            <span className="revoke-dialog-icon"><Icon name="undo" /></span>
            <h2 id="revoke-dialog-title">Thu hồi cấp phát?</h2>
            <p><strong>{revokeDialog.assignment.license_name}</strong> sẽ được thu hồi khỏi <strong>{revokeDialog.assignment.target_name}</strong>.</p>
            <div className="revoke-dialog-note"><Icon name="check" /><span>Seat sẽ được trả lại ngay; bản ghi lịch sử và Audit Log vẫn được giữ.</span></div>
            {revokeDialog.error && <div className="assignment-dialog-error" role="alert"><Icon name="alert" />{revokeDialog.error}</div>}
            <footer><button className="assignment-cancel" type="button" onClick={() => setRevokeDialog(null)} disabled={revokeDialog.loading}>Hủy</button><button className="revoke-confirm" type="button" onClick={confirmRevoke} disabled={revokeDialog.loading}>{revokeDialog.loading ? 'Đang thu hồi...' : 'Xác nhận thu hồi'}</button></footer>
          </section>
        </div>
      )}
    </AdminShell>
  )
}

function CreateAssignmentDialog({ licenses, users, devices, assignments, onClose, onSubmit }: {
  licenses: LicenseItem[]
  users: AuthUser[]
  devices: DeviceItem[]
  assignments: AssignmentItem[]
  onClose: () => void
  onSubmit: (input: AssignmentInput) => Promise<void>
}) {
  const availableLicenses = licenses.filter((item) => item.lifecycle_status === 'active' && item.available_seats > 0)
  const [licenseID, setLicenseID] = useState(availableLicenses[0]?.id ?? '')
  const [targetType, setTargetType] = useState<'user' | 'device'>(() => availableLicenses[0]?.assignment_type === 'device' ? 'device' : 'user')
  const [targetID, setTargetID] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const selectedLicense = availableLicenses.find((item) => item.id === licenseID)

  const assignedTargets = useMemo(() => new Set(assignments
    .filter((item) => item.status === 'active' && item.license_id === licenseID)
    .map((item) => targetType === 'user' ? item.user_id : item.device_id)
    .filter(Boolean)), [assignments, licenseID, targetType])
  const targetUsers = users.filter((item) => item.status === 'active' && !assignedTargets.has(item.id))
  const targetDevices = devices.filter((item) => item.status !== 'retired' && item.status !== 'lost' && !assignedTargets.has(item.id))

  function changeLicense(nextID: string) {
    const license = availableLicenses.find((item) => item.id === nextID)
    setLicenseID(nextID)
    setTargetType(license?.assignment_type === 'device' ? 'device' : 'user')
    setTargetID('')
    setError('')
  }

  function changeTargetType(nextType: 'user' | 'device') {
    setTargetType(nextType)
    setTargetID('')
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!licenseID || !targetID) {
      setError('Vui lòng chọn license và đối tượng nhận cấp phát.')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit({ license_id: licenseID, notes: notes.trim(), ...(targetType === 'user' ? { user_id: targetID } : { device_id: targetID }) })
    } catch (caughtError) {
      setError(translateAssignmentError(caughtError instanceof Error ? caughtError.message : 'Không thể tạo cấp phát.'))
      setIsSubmitting(false)
    }
  }

  return (
    <div className="assignment-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}>
      <section className="create-assignment-dialog" role="dialog" aria-modal="true" aria-labelledby="create-assignment-title">
        <header><div><span><Icon name="assignment" /></span><div><h2 id="create-assignment-title">Cấp license</h2><p>Chọn license và đúng đối tượng sử dụng.</p></div></div><button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button></header>
        <form onSubmit={submit}>
          <div className="create-assignment-body">
            {availableLicenses.length === 0 ? <div className="no-assignable-license"><Icon name="alert" /><div><strong>Không có license khả dụng</strong><p>License phải đang hiệu lực và còn ít nhất một seat trống.</p></div></div> : <>
              <label>License
                <select value={licenseID} onChange={(event) => changeLicense(event.target.value)} disabled={isSubmitting} required>
                  {availableLicenses.map((item) => <option value={item.id} key={item.id}>{item.name} — còn {item.available_seats}/{item.seat_count} seat</option>)}
                </select>
              </label>

              {selectedLicense?.assignment_type === 'mixed' && <fieldset><legend>Đối tượng cấp phát</legend><div className="target-type-switch"><button type="button" className={targetType === 'user' ? 'active' : ''} onClick={() => changeTargetType('user')} disabled={isSubmitting}><Icon name="users" />Người dùng</button><button type="button" className={targetType === 'device' ? 'active' : ''} onClick={() => changeTargetType('device')} disabled={isSubmitting}><Icon name="device" />Thiết bị</button></div></fieldset>}

              <label>{targetType === 'user' ? 'Người dùng nhận license' : 'Thiết bị nhận license'}
                <select value={targetID} onChange={(event) => setTargetID(event.target.value)} disabled={isSubmitting} required>
                  <option value="">{targetType === 'user' ? 'Chọn người dùng — không được bỏ trống' : 'Chọn thiết bị — không được bỏ trống'}</option>
                  {targetType === 'user'
                    ? targetUsers.map((item) => <option value={item.id} key={item.id}>{item.full_name} — {item.employee_code}{item.department_name ? ` · ${item.department_name}` : ''}</option>)
                    : targetDevices.map((item) => <option value={item.id} key={item.id}>{item.asset_code} — {item.name} · {deviceStatusLabel(item.status)}</option>)}
                </select>
                <small>{targetType === 'user' ? `${targetUsers.length} người dùng có thể chọn` : `${targetDevices.length} thiết bị có thể chọn`}</small>
              </label>

              <label>Ghi chú<textarea rows={3} value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Mục đích sử dụng hoặc thông tin bàn giao..." disabled={isSubmitting} /></label>
            </>}
            {error && <div className="assignment-dialog-error" role="alert"><Icon name="alert" />{error}</div>}
          </div>
          <footer><button className="assignment-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="assignment-submit" type="submit" disabled={isSubmitting || availableLicenses.length === 0}>{isSubmitting ? 'Đang cấp phát...' : 'Xác nhận cấp phát'}</button></footer>
        </form>
      </section>
    </div>
  )
}

function AssignmentStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'assignment' | 'users' | 'device' | 'undo'; loading: boolean }) {
  return <article className="assignment-stat"><span className={tone}><Icon name={icon} /></span><div><p>{label}</p>{loading ? <i /> : <strong>{value}</strong>}<small>{detail}</small></div></article>
}

function AssignmentRow({ assignment, onRevoke }: { assignment: AssignmentItem; onRevoke: (assignment: AssignmentItem) => void }) {
  const isUser = Boolean(assignment.user_id)
  return <tr className={assignment.status === 'revoked' ? 'revoked' : undefined}>
    <td><div className="assignment-license"><span><Icon name="key" /></span><div><strong>{assignment.license_name}</strong><small>{assignment.notes || 'Không có ghi chú'}</small></div></div></td>
    <td><div className="assignment-target"><span>{targetInitials(assignment.target_name)}</span><div><strong>{assignment.target_name}</strong><small>{isUser ? 'Người dùng' : 'Thiết bị'}</small></div></div></td>
    <td><span className={`assignment-source ${isUser ? 'user' : 'device'}`}><Icon name={isUser ? 'users' : 'device'} />{isUser ? 'Người dùng' : 'Thiết bị'}</span></td>
    <td><strong className="assigned-by">{assignment.assigned_by_name || 'Quản trị viên'}</strong></td>
    <td><strong className="assigned-date">{formatDateTime(assignment.assigned_at)}</strong>{assignment.revoked_at && <small className="revoked-date">Thu hồi {formatDateTime(assignment.revoked_at)}</small>}</td>
    <td><span className={`assignment-status ${assignment.status}`}><i />{assignment.status === 'active' ? 'Đang hoạt động' : 'Đã thu hồi'}</span></td>
    <td>{assignment.status === 'active' && <button className="revoke-button" type="button" onClick={() => onRevoke(assignment)} title="Thu hồi cấp phát" aria-label={`Thu hồi ${assignment.license_name} khỏi ${assignment.target_name}`}><Icon name="undo" /></button>}</td>
  </tr>
}

function AssignmentError({ error, onRetry, onLogout }: { error: AssignmentAPIError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const authError = error.status === 401 || error.status === 403
  return <div className="assignment-error"><Icon name="alert" /><strong>{authError ? 'Không thể truy cập' : 'Không thể tải cấp phát'}</strong><p>{error.status === 401 ? 'Phiên đăng nhập đã hết hạn.' : error.status === 403 ? 'Tài khoản không có quyền quản lý cấp phát.' : error.status === 0 ? 'Hãy kiểm tra backend đang chạy ở cổng 8080.' : error.message}</p><button type="button" onClick={authError ? onLogout : onRetry}>{authError ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function matchesFilter(item: AssignmentItem, filter: AssignmentFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'active' || filter === 'revoked') return item.status === filter
  return filter === 'user' ? Boolean(item.user_id) : Boolean(item.device_id)
}

function translateAssignmentError(message: string): string {
  if (message.includes('already assigned')) return 'License này đang được cấp cho đối tượng đã chọn.'
  if (message.includes('no available seats')) return 'License đã hết seat. Hãy thu hồi cấp phát cũ hoặc tăng số seat.'
  if (message.includes('not currently active')) return 'License không còn hiệu lực hoặc đã được lưu trữ.'
  if (message.includes('assignment type')) return 'Đối tượng không phù hợp với hình thức cấp phát của license.'
  if (message.includes('unavailable')) return 'Người dùng hoặc thiết bị đã chọn không còn khả dụng.'
  if (message.includes('already revoked')) return 'Cấp phát này đã được thu hồi trước đó.'
  return message
}

function targetInitials(name: string): string { return name.split(/\s+/).filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || '—' }
function formatDateTime(value: string): string { return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value)) }
function deviceStatusLabel(value: DeviceItem['status']): string { return value === 'assigned' ? 'Đã bàn giao' : value === 'maintenance' ? 'Bảo trì' : 'Khả dụng' }
