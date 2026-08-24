import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import type { AuthSession, AuthUser, UserRole } from '../../lib/auth-api'
import { DepartmentAPIError, getDepartments, type DepartmentItem } from '../../lib/department-api'
import {
  createUser,
  getUsers,
  updateUserStatus,
  UserAPIError,
  type CreateUserInput,
  type UserStatus,
} from '../../lib/user-api'
import './UserManagementScreen.css'

interface UserManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type RoleFilter = 'all' | UserRole
type StatusFilter = 'all' | UserStatus

const roleLabels: Record<UserRole, string> = {
  admin: 'Quản trị viên',
  it_manager: 'Quản lý IT',
  employee: 'Nhân viên',
}

export function UserManagementScreen({ session, onNavigate, onLogout }: UserManagementScreenProps) {
  const [users, setUsers] = useState<AuthUser[]>([])
  const [departments, setDepartments] = useState<DepartmentItem[]>([])
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<UserAPIError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [statusUser, setStatusUser] = useState<AuthUser | null>(null)
  const [successMessage, setSuccessMessage] = useState('')
  const canManage = session.user.role === 'admin'

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    Promise.all([
      getUsers(session.tokens.access_token),
      getDepartments(session.tokens.access_token),
    ])
      .then(([userResult, departmentResult]) => {
        if (cancelled) return
        setUsers(userResult.items)
        setDepartments(departmentResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        if (caughtError instanceof UserAPIError) {
          setError(caughtError)
        } else if (caughtError instanceof DepartmentAPIError) {
          setError(new UserAPIError(caughtError.message, caughtError.status))
        } else {
          setError(new UserAPIError('Đã xảy ra lỗi không mong muốn.', 0))
        }
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })

    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const filteredUsers = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return users.filter((user) => {
      const matchesSearch = !query || [
        user.full_name,
        user.email,
        user.employee_code,
        user.department_name,
      ].some((value) => value?.toLocaleLowerCase('vi').includes(query))
      const matchesRole = roleFilter === 'all' || user.role === roleFilter
      const matchesStatus = statusFilter === 'all' || user.status === statusFilter
      return matchesSearch && matchesRole && matchesStatus
    })
  }, [roleFilter, search, statusFilter, users])

  const counts = useMemo(() => ({
    total: users.length,
    active: users.filter((user) => user.status === 'active').length,
    managers: users.filter((user) => user.role === 'admin' || user.role === 'it_manager').length,
    locked: users.filter((user) => user.status === 'locked').length,
  }), [users])

  async function saveUser(input: CreateUserInput) {
    const created = await createUser(session.tokens.access_token, input)
    setShowCreateDialog(false)
    setSuccessMessage(`Đã tạo tài khoản cho ${created.full_name}.`)
    setReloadKey((value) => value + 1)
  }

  async function saveStatus(user: AuthUser) {
    const nextStatus: UserStatus = user.status === 'active' ? 'locked' : 'active'
    const updated = await updateUserStatus(session.tokens.access_token, user.id, nextStatus)
    setStatusUser(null)
    setSuccessMessage(updated.status === 'locked'
      ? `Đã khóa tài khoản ${updated.email}.`
      : `Đã mở khóa tài khoản ${updated.email}.`)
    setReloadKey((value) => value + 1)
  }

  const headerActions = (
    <button className="user-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}>
      <Icon name="refresh" /><span>Làm mới</span>
    </button>
  )

  return (
    <AdminShell session={session} activePage="users" title="Quản lý người dùng" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="user-page">
        <section className="user-page-heading">
          <div>
            <h2>Danh sách người dùng</h2>
            <p>Quản lý tài khoản, vai trò, phòng ban và trạng thái truy cập hệ thống.</p>
          </div>
          {canManage && <button className="add-user-button" type="button" onClick={() => setShowCreateDialog(true)}><Icon name="plus" />Thêm người dùng</button>}
        </section>

        {!canManage && !error && <div className="user-readonly-note"><Icon name="eye" /><span>Bạn đang ở chế độ chỉ xem. Chỉ Quản trị viên mới có thể tạo hoặc khóa tài khoản.</span></div>}
        {successMessage && <div className="user-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="user-stats" aria-label="Thống kê người dùng">
          <UserStat label="Tổng người dùng" value={counts.total} detail="tài khoản trong hệ thống" tone="blue" icon="users" loading={isLoading} />
          <UserStat label="Đang hoạt động" value={counts.active} detail="có thể đăng nhập" tone="green" icon="check" loading={isLoading} />
          <UserStat label="Nhóm quản lý" value={counts.managers} detail="Admin và Quản lý IT" tone="violet" icon="settings" loading={isLoading} />
          <UserStat label="Đã khóa" value={counts.locked} detail="không thể đăng nhập" tone="red" icon="archive" loading={isLoading} />
        </section>

        <section className="user-list-card">
          <div className="user-toolbar">
            <div className="user-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm tên, email, mã nhân viên, phòng ban..." aria-label="Tìm kiếm người dùng" /></div>
            <div className="user-filter"><Icon name="users" /><select value={roleFilter} onChange={(event) => setRoleFilter(event.target.value as RoleFilter)} aria-label="Lọc vai trò"><option value="all">Tất cả vai trò</option><option value="admin">Quản trị viên</option><option value="it_manager">Quản lý IT</option><option value="employee">Nhân viên</option></select></div>
            <div className="user-filter"><Icon name="filter" /><select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value as StatusFilter)} aria-label="Lọc trạng thái"><option value="all">Tất cả trạng thái</option><option value="active">Đang hoạt động</option><option value="locked">Đã khóa</option></select></div>
            <span className="user-result-count">{filteredUsers.length} kết quả</span>
          </div>

          {error ? <UserError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="user-loading" aria-label="Đang tải người dùng">{Array.from({ length: 6 }, (_, index) => <span key={index} />)}</div>
          ) : filteredUsers.length === 0 ? (
            <div className="user-empty"><span><Icon name="users" /></span><strong>Không tìm thấy người dùng</strong><p>Thử thay đổi từ khóa, bộ lọc hoặc thêm tài khoản mới.</p></div>
          ) : (
            <div className="user-table-scroll"><table className="user-table">
              <thead><tr><th>Người dùng</th><th>Mã nhân viên</th><th>Phòng ban</th><th>Vai trò</th><th>Ngày tạo</th><th>Trạng thái</th><th /></tr></thead>
              <tbody>{filteredUsers.map((user) => <UserRow user={user} currentUserID={session.user.id} canManage={canManage} onStatus={setStatusUser} key={user.id} />)}</tbody>
            </table></div>
          )}
        </section>
      </div>

      {showCreateDialog && <CreateUserDialog departments={departments} onClose={() => setShowCreateDialog(false)} onSubmit={saveUser} />}
      {statusUser && <UserStatusDialog user={statusUser} onClose={() => setStatusUser(null)} onSubmit={() => saveStatus(statusUser)} />}
    </AdminShell>
  )
}

function UserRow({ user, currentUserID, canManage, onStatus }: { user: AuthUser; currentUserID: string; canManage: boolean; onStatus: (user: AuthUser) => void }) {
  const isCurrentUser = user.id === currentUserID
  return <tr>
    <td><div className="user-identity"><span className={`user-avatar ${user.role}`}><Icon name="users" /></span><div><strong>{user.full_name}{isCurrentUser && <small>Bạn</small>}</strong><span>{user.email}</span></div></div></td>
    <td><span className="employee-code">{user.employee_code}</span></td>
    <td><span className={user.department_name ? 'department-name' : 'department-name muted'}>{user.department_name || 'Chưa phân phòng ban'}</span></td>
    <td><span className={`user-role ${user.role}`}>{roleLabels[user.role]}</span></td>
    <td><span className="user-created-date">{formatDate(user.created_at)}</span></td>
    <td><span className={`user-status ${user.status}`}><i />{user.status === 'active' ? 'Hoạt động' : 'Đã khóa'}</span></td>
    <td>{canManage && <div className="user-row-actions"><button type="button" onClick={() => onStatus(user)} disabled={isCurrentUser && user.status === 'active'} aria-label={user.status === 'active' ? `Khóa ${user.full_name}` : `Mở khóa ${user.full_name}`} title={isCurrentUser && user.status === 'active' ? 'Bạn không thể tự khóa tài khoản của mình' : user.status === 'active' ? 'Khóa tài khoản' : 'Mở khóa tài khoản'}><Icon name={user.status === 'active' ? 'archive' : 'undo'} /></button></div>}</td>
  </tr>
}

function CreateUserDialog({ departments, onClose, onSubmit }: { departments: DepartmentItem[]; onClose: () => void; onSubmit: (input: CreateUserInput) => Promise<void> }) {
  const [input, setInput] = useState<CreateUserInput>({ email: '', password: '', full_name: '', employee_code: '', department_id: '', role: 'employee' })
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function update<K extends keyof CreateUserInput>(key: K, value: CreateUserInput[K]) {
    setInput((current) => ({ ...current, [key]: value }))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!input.full_name.trim() || !input.email.trim() || !input.employee_code.trim() || !input.password) {
      setError('Vui lòng nhập đầy đủ họ tên, email, mã nhân viên và mật khẩu.')
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.email.trim())) {
      setError('Email không đúng định dạng.')
      return
    }
    if (input.password.length < 10 || !/[A-Z]/.test(input.password) || !/[a-z]/.test(input.password) || !/\d/.test(input.password)) {
      setError('Mật khẩu phải có ít nhất 10 ký tự, gồm chữ hoa, chữ thường và chữ số.')
      return
    }

    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit({ ...input, email: input.email.trim(), full_name: input.full_name.trim(), employee_code: input.employee_code.trim() })
    } catch (caughtError) {
      setError(translateUserError(caughtError instanceof Error ? caughtError.message : 'Không thể tạo tài khoản.'))
      setIsSubmitting(false)
    }
  }

  return <div className="user-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="user-form-dialog" role="dialog" aria-modal="true" aria-labelledby="user-form-title">
    <header><div><span><Icon name="plus" /></span><div><h2 id="user-form-title">Thêm người dùng</h2><p>Tạo tài khoản và gán quyền truy cập ban đầu.</p></div></div><button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button></header>
    <form onSubmit={submit}><div className="user-form-body">
      <div className="user-form-section"><span>01</span><div><strong>Thông tin nhân sự</strong><small>Danh tính và đơn vị làm việc</small></div></div>
      <div className="user-form-grid"><label className="full">Họ và tên<input value={input.full_name} onChange={(event) => update('full_name', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: Nguyễn Văn An" disabled={isSubmitting} autoFocus required /></label><label>Email công việc<input type="email" value={input.email} onChange={(event) => update('email', event.target.value)} placeholder="Không được bỏ trống — ten@congty.vn" disabled={isSubmitting} required /></label><label>Mã nhân viên<input value={input.employee_code} onChange={(event) => update('employee_code', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: EMP-006" disabled={isSubmitting} required /></label><label>Phòng ban<select value={input.department_id} onChange={(event) => update('department_id', event.target.value)} disabled={isSubmitting}><option value="">Chưa phân phòng ban</option>{departments.map((department) => <option value={department.id} key={department.id}>{department.name} ({department.code})</option>)}</select></label><label>Vai trò<select value={input.role} onChange={(event) => update('role', event.target.value as UserRole)} disabled={isSubmitting}><option value="employee">Nhân viên</option><option value="it_manager">Quản lý IT</option><option value="admin">Quản trị viên</option></select></label></div>
      <div className="user-form-section"><span>02</span><div><strong>Bảo mật tài khoản</strong><small>Mật khẩu dùng cho lần đăng nhập đầu tiên</small></div></div>
      <div className="user-password-field"><label>Mật khẩu<div><input type={showPassword ? 'text' : 'password'} value={input.password} onChange={(event) => update('password', event.target.value)} placeholder="Không được bỏ trống — tối thiểu 10 ký tự" disabled={isSubmitting} autoComplete="new-password" required /><button type="button" onClick={() => setShowPassword((value) => !value)} disabled={isSubmitting}>{showPassword ? 'Ẩn' : 'Hiện'}</button></div><small>Phải có chữ hoa, chữ thường và ít nhất một chữ số.</small></label></div>
      {error && <div className="user-dialog-error" role="alert"><Icon name="alert" />{error}</div>}
    </div><footer><button className="user-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="user-submit" type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang tạo...' : 'Tạo tài khoản'}</button></footer></form>
  </section></div>
}

function UserStatusDialog({ user, onClose, onSubmit }: { user: AuthUser; onClose: () => void; onSubmit: () => Promise<void> }) {
  const locking = user.status === 'active'
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try { await onSubmit() } catch (caughtError) {
      setError(translateUserError(caughtError instanceof Error ? caughtError.message : 'Không thể cập nhật trạng thái.'))
      setIsSubmitting(false)
    }
  }

  return <div className="user-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="user-status-dialog" role="dialog" aria-modal="true" aria-labelledby="user-status-title">
    <span className={locking ? 'status-dialog-icon lock' : 'status-dialog-icon unlock'}><Icon name={locking ? 'archive' : 'undo'} /></span>
    <h2 id="user-status-title">{locking ? 'Khóa tài khoản?' : 'Mở khóa tài khoản?'}</h2>
    <p><strong>{user.full_name}</strong><span>{user.email}</span></p>
    <div className={locking ? 'user-status-note warning' : 'user-status-note success'}><Icon name={locking ? 'alert' : 'check'} /><span>{locking ? 'Người dùng sẽ không thể đăng nhập cho đến khi được mở khóa.' : 'Người dùng có thể đăng nhập lại ngay sau khi mở khóa.'}</span></div>
    <form onSubmit={submit}>{error && <div className="user-dialog-error" role="alert"><Icon name="alert" />{error}</div>}<footer><button className="user-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className={locking ? 'user-lock-confirm' : 'user-submit'} type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang cập nhật...' : locking ? 'Xác nhận khóa' : 'Xác nhận mở khóa'}</button></footer></form>
  </section></div>
}

function UserStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'users' | 'check' | 'settings' | 'archive'; loading: boolean }) {
  return <article className={`user-stat ${tone}`}><span><Icon name={icon} /></span><div><small>{label}</small><strong>{loading ? '—' : value}</strong><p>{detail}</p></div></article>
}

function UserError({ error, onRetry, onLogout }: { error: UserAPIError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  const forbidden = error.status === 403
  return <div className="user-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : forbidden ? 'Bạn không có quyền xem người dùng' : 'Không thể tải danh sách người dùng'}</strong><p>{expired ? 'Đăng nhập lại để tiếp tục.' : forbidden ? 'Chỉ Admin và Quản lý IT được truy cập module này.' : translateUserError(error.message)}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : new Intl.DateTimeFormat('vi-VN').format(date)
}

function translateUserError(message: string) {
  const translations: Record<string, string> = {
    'email already exists': 'Email này đã được sử dụng.',
    'employee code already exists': 'Mã nhân viên này đã được sử dụng.',
    'password must contain at least 10 characters, including uppercase, lowercase and a number': 'Mật khẩu phải có ít nhất 10 ký tự, gồm chữ hoa, chữ thường và chữ số.',
    'department not found': 'Phòng ban đã chọn không còn tồn tại.',
    'an administrator cannot lock their own account': 'Bạn không thể tự khóa tài khoản của mình.',
    'invalid user role': 'Vai trò người dùng không hợp lệ.',
    'invalid user status': 'Trạng thái người dùng không hợp lệ.',
  }
  return translations[message.toLocaleLowerCase('en')] ?? message
}
