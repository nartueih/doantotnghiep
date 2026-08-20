import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import type { AuthSession, AuthUser } from '../../lib/auth-api'
import {
  createDepartment,
  DepartmentAPIError,
  getDepartments,
  updateDepartment,
  type DepartmentInput,
  type DepartmentItem,
} from '../../lib/department-api'
import { getUsers, UserAPIError } from '../../lib/user-api'
import './DepartmentManagementScreen.css'

interface DepartmentManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

interface PageError {
  message: string
  status: number
}

export function DepartmentManagementScreen({ session, onNavigate, onLogout }: DepartmentManagementScreenProps) {
  const [departments, setDepartments] = useState<DepartmentItem[]>([])
  const [users, setUsers] = useState<AuthUser[]>([])
  const [search, setSearch] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<PageError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [formDepartment, setFormDepartment] = useState<DepartmentItem | 'new' | null>(null)
  const [successMessage, setSuccessMessage] = useState('')
  const canManage = session.user.role === 'admin'

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    Promise.all([
      getDepartments(session.tokens.access_token),
      getUsers(session.tokens.access_token),
    ])
      .then(([departmentResult, userResult]) => {
        if (cancelled) return
        setDepartments(departmentResult.items)
        setUsers(userResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        if (caughtError instanceof DepartmentAPIError || caughtError instanceof UserAPIError) {
          setError({ message: caughtError.message, status: caughtError.status })
        } else {
          setError({ message: 'Đã xảy ra lỗi không mong muốn.', status: 0 })
        }
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })

    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const usersByDepartment = useMemo(() => {
    const result = new Map<string, AuthUser[]>()
    for (const user of users) {
      if (!user.department_id) continue
      result.set(user.department_id, [...(result.get(user.department_id) ?? []), user])
    }
    return result
  }, [users])

  const filteredDepartments = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return departments.filter((department) => {
      const members = usersByDepartment.get(department.id) ?? []
      return !query || [department.name, department.code, ...members.flatMap((member) => [member.full_name, member.email, member.employee_code])]
        .some((value) => value.toLocaleLowerCase('vi').includes(query))
    })
  }, [departments, search, usersByDepartment])

  const counts = useMemo(() => {
    const assigned = users.filter((user) => user.department_id).length
    let largestName = 'Chưa có dữ liệu'
    let largestCount = 0
    for (const department of departments) {
      const count = usersByDepartment.get(department.id)?.length ?? 0
      if (count > largestCount) {
        largestCount = count
        largestName = department.name
      }
    }
    return { total: departments.length, assigned, largestName, largestCount, unassigned: users.length - assigned }
  }, [departments, users, usersByDepartment])

  async function saveDepartment(input: DepartmentInput) {
    if (formDepartment === 'new') {
      const created = await createDepartment(session.tokens.access_token, input)
      setSuccessMessage(`Đã tạo phòng ban ${created.name}.`)
    } else if (formDepartment) {
      const updated = await updateDepartment(session.tokens.access_token, formDepartment.id, input)
      setSuccessMessage(`Đã cập nhật phòng ban ${updated.name}.`)
    }
    setFormDepartment(null)
    setReloadKey((value) => value + 1)
  }

  const headerActions = <button className="department-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}><Icon name="refresh" /><span>Làm mới</span></button>

  return (
    <AdminShell session={session} activePage="departments" title="Quản lý phòng ban" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="department-page">
        <section className="department-page-heading">
          <div><h2>Danh sách phòng ban</h2><p>Tổ chức đơn vị làm việc và theo dõi nhân sự đang thuộc từng phòng ban.</p></div>
          {canManage && <button className="add-department-button" type="button" onClick={() => setFormDepartment('new')}><Icon name="plus" />Thêm phòng ban</button>}
        </section>

        {!canManage && !error && <div className="department-readonly-note"><Icon name="eye" /><span>Bạn đang ở chế độ chỉ xem. Chỉ Quản trị viên mới có thể thêm hoặc chỉnh sửa phòng ban.</span></div>}
        {successMessage && <div className="department-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="department-stats" aria-label="Thống kê phòng ban">
          <DepartmentStat label="Tổng phòng ban" value={counts.total} detail="đơn vị trong hệ thống" tone="blue" icon="department" loading={isLoading} />
          <DepartmentStat label="Đã phân phòng" value={counts.assigned} detail="người dùng có phòng ban" tone="green" icon="users" loading={isLoading} />
          <DepartmentStat label="Phòng đông nhất" value={counts.largestCount} detail={counts.largestName} tone="violet" icon="trend" loading={isLoading} />
          <DepartmentStat label="Chưa phân phòng" value={counts.unassigned} detail="người dùng cần cập nhật" tone="amber" icon="alert" loading={isLoading} />
        </section>

        <section className="department-list-card">
          <div className="department-toolbar">
            <div className="department-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm tên, mã phòng ban hoặc nhân viên..." aria-label="Tìm kiếm phòng ban" /></div>
            <span className="department-result-count">{filteredDepartments.length} kết quả</span>
          </div>

          {error ? <DepartmentError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="department-loading" aria-label="Đang tải phòng ban">{Array.from({ length: 4 }, (_, index) => <span key={index} />)}</div>
          ) : filteredDepartments.length === 0 ? (
            <div className="department-empty"><span><Icon name="department" /></span><strong>Không tìm thấy phòng ban</strong><p>Thử thay đổi từ khóa hoặc thêm phòng ban mới.</p></div>
          ) : (
            <div className="department-table-scroll"><table className="department-table">
              <thead><tr><th>Phòng ban</th><th>Mã phòng ban</th><th>Nhân sự</th><th>Thành viên tiêu biểu</th><th>Cập nhật gần nhất</th><th /></tr></thead>
              <tbody>{filteredDepartments.map((department) => <DepartmentRow department={department} members={usersByDepartment.get(department.id) ?? []} canManage={canManage} onEdit={setFormDepartment} key={department.id} />)}</tbody>
            </table></div>
          )}
        </section>
      </div>

      {formDepartment && <DepartmentFormDialog department={formDepartment === 'new' ? undefined : formDepartment} onClose={() => setFormDepartment(null)} onSubmit={saveDepartment} />}
    </AdminShell>
  )
}

function DepartmentRow({ department, members, canManage, onEdit }: { department: DepartmentItem; members: AuthUser[]; canManage: boolean; onEdit: (department: DepartmentItem) => void }) {
  const visibleMembers = members.slice(0, 3)
  return <tr>
    <td><div className="department-identity"><span><Icon name="department" /></span><div><strong>{department.name}</strong><small>Tạo ngày {formatDate(department.created_at)}</small></div></div></td>
    <td><span className="department-code">{department.code}</span></td>
    <td><div className="department-member-count"><strong>{members.length}</strong><span>người dùng</span></div></td>
    <td>{members.length ? <div className="department-members"><div>{visibleMembers.map((member) => <span title={`${member.full_name} · ${member.employee_code}`} key={member.id}>{initials(member.full_name)}</span>)}{members.length > visibleMembers.length && <span className="more">+{members.length - visibleMembers.length}</span>}</div><small>{members.slice(0, 2).map((member) => member.full_name).join(', ')}{members.length > 2 ? '…' : ''}</small></div> : <span className="department-no-members">Chưa có thành viên</span>}</td>
    <td><span className="department-updated">{formatDate(department.updated_at)}</span></td>
    <td>{canManage && <div className="department-row-actions"><button type="button" onClick={() => onEdit(department)} aria-label={`Chỉnh sửa ${department.name}`} title="Chỉnh sửa phòng ban"><Icon name="edit" /></button></div>}</td>
  </tr>
}

function DepartmentFormDialog({ department, onClose, onSubmit }: { department?: DepartmentItem; onClose: () => void; onSubmit: (input: DepartmentInput) => Promise<void> }) {
  const [input, setInput] = useState<DepartmentInput>({ name: department?.name ?? '', code: department?.code ?? '' })
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!input.name.trim() || !input.code.trim()) {
      setError('Vui lòng nhập đầy đủ tên và mã phòng ban.')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit({ name: input.name.trim(), code: input.code.trim().toUpperCase() })
    } catch (caughtError) {
      setError(translateDepartmentError(caughtError instanceof Error ? caughtError.message : 'Không thể lưu phòng ban.'))
      setIsSubmitting(false)
    }
  }

  return <div className="department-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="department-form-dialog" role="dialog" aria-modal="true" aria-labelledby="department-form-title">
    <header><div><span><Icon name={department ? 'edit' : 'plus'} /></span><div><h2 id="department-form-title">{department ? 'Chỉnh sửa phòng ban' : 'Thêm phòng ban'}</h2><p>{department ? 'Cập nhật tên và mã nhận diện của đơn vị.' : 'Tạo đơn vị mới trong cơ cấu doanh nghiệp.'}</p></div></div><button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button></header>
    <form onSubmit={submit}><div className="department-form-body">
      <label>Tên phòng ban<input value={input.name} onChange={(event) => setInput((current) => ({ ...current, name: event.target.value }))} placeholder="Không được bỏ trống — Ví dụ: Kinh doanh" disabled={isSubmitting} autoFocus required /></label>
      <label>Mã phòng ban<input value={input.code} onChange={(event) => setInput((current) => ({ ...current, code: event.target.value.toUpperCase() }))} placeholder="Không được bỏ trống — Ví dụ: SALES" disabled={isSubmitting} required /><small>Mã được chuẩn hóa thành chữ hoa và phải là duy nhất.</small></label>
      {error && <div className="department-dialog-error" role="alert"><Icon name="alert" />{error}</div>}
    </div><footer><button className="department-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="department-submit" type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang lưu...' : department ? 'Lưu thay đổi' : 'Tạo phòng ban'}</button></footer></form>
  </section></div>
}

function DepartmentStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'department' | 'users' | 'trend' | 'alert'; loading: boolean }) {
  return <article className={`department-stat ${tone}`}><span><Icon name={icon} /></span><div><small>{label}</small><strong>{loading ? '—' : value}</strong><p>{detail}</p></div></article>
}

function DepartmentError({ error, onRetry, onLogout }: { error: PageError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  const forbidden = error.status === 403
  return <div className="department-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : forbidden ? 'Bạn không có quyền xem phòng ban' : 'Không thể tải danh sách phòng ban'}</strong><p>{expired ? 'Đăng nhập lại để tiếp tục.' : forbidden ? 'Chỉ Admin và Quản lý IT được truy cập module này.' : translateDepartmentError(error.message)}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function initials(name: string) {
  return name.split(' ').filter(Boolean).slice(-2).map((part) => part[0]).join('').toUpperCase() || 'U'
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : new Intl.DateTimeFormat('vi-VN').format(date)
}

function translateDepartmentError(message: string) {
  const translations: Record<string, string> = {
    'department name already exists': 'Tên phòng ban này đã được sử dụng.',
    'department code already exists': 'Mã phòng ban này đã được sử dụng.',
    'department name and code are required': 'Tên và mã phòng ban không được bỏ trống.',
    'department not found': 'Phòng ban không còn tồn tại.',
  }
  return translations[message.toLocaleLowerCase('en')] ?? message
}
