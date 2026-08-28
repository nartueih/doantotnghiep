import { Fragment, useEffect, useMemo, useState } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import { type AuthSession } from '../../lib/auth-api'
import { AuditAPIError, getAuditLogs, type AuditLogItem } from '../../lib/audit-api'
import './AuditLogScreen.css'

interface AuditLogScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

interface PageError {
  message: string
  status: number
}

const actionLabels: Record<string, string> = {
  create: 'Tạo mới',
  update: 'Cập nhật',
  status_change: 'Đổi trạng thái',
  assign: 'Cấp phát',
  revoke: 'Thu hồi',
  view_key: 'Xem khóa',
  archive: 'Lưu trữ',
  request: 'Gửi yêu cầu',
  cancel: 'Hủy yêu cầu',
  approve: 'Phê duyệt',
  reject: 'Từ chối',
  accept: 'Tiếp nhận',
  complete: 'Hoàn thành',
}

const entityLabels: Record<string, string> = {
  user: 'Người dùng',
  department: 'Phòng ban',
  software_product: 'Phần mềm',
  license: 'License',
  device: 'Thiết bị',
  license_assignment: 'Cấp phát license',
  license_request: 'Yêu cầu license',
  maintenance_request: 'Yêu cầu bảo trì',
}

const metadataLabels: Record<string, string> = {
  name: 'Tên',
  email: 'Email',
  role: 'Vai trò',
  status: 'Trạng thái',
  code: 'Mã',
  publisher: 'Nhà phát hành',
  version: 'Phiên bản',
  asset_code: 'Mã tài sản',
  device_type: 'Loại thiết bị',
  seat_count: 'Số seat',
  key_configured: 'Đã cấu hình khóa',
  key_changed: 'Đã đổi khóa',
  department_id: 'ID phòng ban',
  software_product_id: 'ID phần mềm',
  assigned_user_id: 'ID người nhận',
  license_id: 'ID license',
  user_id: 'ID người dùng',
  device_id: 'ID thiết bị',
  category: 'Nhóm sự cố',
  priority: 'Mức ưu tiên',
  assigned_to: 'ID người phụ trách',
}

export function AuditLogScreen({ session, onNavigate, onLogout }: AuditLogScreenProps) {
  const [logs, setLogs] = useState<AuditLogItem[]>([])
  const [search, setSearch] = useState('')
  const [actionFilter, setActionFilter] = useState('all')
  const [entityFilter, setEntityFilter] = useState('all')
  const [actorFilter, setActorFilter] = useState('all')
  const [expandedID, setExpandedID] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<PageError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    getAuditLogs(session.tokens.access_token)
      .then((result) => {
        if (!cancelled) setLogs(result.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        if (caughtError instanceof AuditAPIError) {
          setError({ message: caughtError.message, status: caughtError.status })
        } else {
          setError({ message: 'Đã xảy ra lỗi không mong muốn.', status: 0 })
        }
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })

    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const actions = useMemo(() => [...new Set(logs.map((item) => item.action))].sort(), [logs])
  const entities = useMemo(() => [...new Set(logs.map((item) => item.entity_type))].sort(), [logs])
  const actors = useMemo(() => {
    const result = new Map<string, { id: string; name: string; email: string }>()
    for (const item of logs) {
      if (!item.actor_id) continue
      result.set(item.actor_id, {
        id: item.actor_id,
        name: item.actor_name || item.actor_email || 'Người dùng không xác định',
        email: item.actor_email || '',
      })
    }
    return [...result.values()].sort((a, b) => a.name.localeCompare(b.name, 'vi'))
  }, [logs])

  const filteredLogs = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return logs.filter((item) => {
      const searchable = [
        item.actor_name,
        item.actor_email,
        actionLabels[item.action],
        entityLabels[item.entity_type],
        item.entity_id,
        item.ip_address,
        JSON.stringify(item.metadata),
      ].filter(Boolean).join(' ').toLocaleLowerCase('vi')
      return (!query || searchable.includes(query))
        && (actionFilter === 'all' || item.action === actionFilter)
        && (entityFilter === 'all' || item.entity_type === entityFilter)
        && (actorFilter === 'all' || item.actor_id === actorFilter)
    })
  }, [actionFilter, actorFilter, entityFilter, logs, search])

  const stats = useMemo(() => {
    const today = new Date()
    const todayCount = logs.filter((item) => sameLocalDay(new Date(item.created_at), today)).length
    const securityCount = logs.filter((item) => ['view_key', 'status_change', 'archive'].includes(item.action)).length
    return { total: logs.length, today: todayCount, actors: actors.length, security: securityCount }
  }, [actors.length, logs])

  const hasFilters = Boolean(search || actionFilter !== 'all' || entityFilter !== 'all' || actorFilter !== 'all')

  function clearFilters() {
    setSearch('')
    setActionFilter('all')
    setEntityFilter('all')
    setActorFilter('all')
  }

  const headerActions = <button className="audit-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}><Icon name="refresh" /><span>Làm mới</span></button>

  return (
    <AdminShell session={session} activePage="audit" title="Nhật ký hoạt động" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="audit-page">
        <section className="audit-page-heading">
          <div><h2>Lịch sử thao tác hệ thống</h2><p>Theo dõi ai đã thay đổi dữ liệu nào, vào thời điểm nào và từ địa chỉ mạng nào.</p></div>
          <span className="audit-readonly"><Icon name="eye" />Chỉ đọc · Không thể sửa hoặc xóa</span>
        </section>

        <section className="audit-stats" aria-label="Thống kê nhật ký">
          <AuditStat label="Tổng sự kiện" value={stats.total} detail="trong 200 bản ghi mới nhất" tone="blue" icon="audit" loading={isLoading} />
          <AuditStat label="Hôm nay" value={stats.today} detail="thao tác phát sinh hôm nay" tone="green" icon="calendar" loading={isLoading} />
          <AuditStat label="Người thao tác" value={stats.actors} detail="tài khoản có hoạt động" tone="violet" icon="users" loading={isLoading} />
          <AuditStat label="Sự kiện nhạy cảm" value={stats.security} detail="xem khóa, đổi trạng thái, lưu trữ" tone="amber" icon="alert" loading={isLoading} />
        </section>

        <section className="audit-list-card">
          <div className="audit-toolbar">
            <div className="audit-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm người thao tác, đối tượng, IP hoặc nội dung..." aria-label="Tìm kiếm nhật ký" /></div>
            <div className="audit-filter"><Icon name="filter" /><select value={actionFilter} onChange={(event) => setActionFilter(event.target.value)} aria-label="Lọc hành động"><option value="all">Tất cả hành động</option>{actions.map((action) => <option value={action} key={action}>{actionLabels[action] ?? action}</option>)}</select></div>
            <div className="audit-filter"><select value={entityFilter} onChange={(event) => setEntityFilter(event.target.value)} aria-label="Lọc đối tượng"><option value="all">Tất cả đối tượng</option>{entities.map((entity) => <option value={entity} key={entity}>{entityLabels[entity] ?? entity}</option>)}</select></div>
            <div className="audit-filter actor"><select value={actorFilter} onChange={(event) => setActorFilter(event.target.value)} aria-label="Lọc người thao tác"><option value="all">Tất cả người thao tác</option>{actors.map((actor) => <option value={actor.id} key={actor.id}>{actor.name}</option>)}</select></div>
            {hasFilters && <button className="audit-clear-filter" type="button" onClick={clearFilters}>Xóa lọc</button>}
            <span className="audit-result-count">{filteredLogs.length} kết quả</span>
          </div>

          {error ? <AuditError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="audit-loading" aria-label="Đang tải nhật ký">{Array.from({ length: 7 }, (_, index) => <span key={index} />)}</div>
          ) : filteredLogs.length === 0 ? (
            <div className="audit-empty"><span><Icon name="audit" /></span><strong>{logs.length ? 'Không tìm thấy sự kiện phù hợp' : 'Chưa có hoạt động nào được ghi nhận'}</strong><p>{logs.length ? 'Hãy thử thay đổi từ khóa hoặc xóa bớt bộ lọc.' : 'Các thao tác tạo, sửa, cấp phát và xem khóa sẽ xuất hiện tại đây.'}</p>{hasFilters && <button type="button" onClick={clearFilters}>Xóa bộ lọc</button>}</div>
          ) : (
            <div className="audit-table-scroll"><table className="audit-table">
              <thead><tr><th>Thời gian</th><th>Người thao tác</th><th>Hành động</th><th>Đối tượng</th><th>Nội dung chính</th><th>Địa chỉ IP</th><th /></tr></thead>
              <tbody>{filteredLogs.map((item) => {
                const expanded = expandedID === item.id
                return <Fragment key={item.id}><tr className={expanded ? 'expanded' : undefined}>
                  <td><time dateTime={item.created_at}><strong>{formatTime(item.created_at)}</strong><small>{formatDate(item.created_at)}</small></time></td>
                  <td><div className="audit-actor"><span>{initials(item.actor_name || item.actor_email || 'Hệ thống')}</span><div><strong>{item.actor_name || 'Không xác định'}</strong><small>{item.actor_email || item.actor_id || 'Hệ thống'}</small></div></div></td>
                  <td><span className={`audit-action action-${item.action}`}>{actionLabels[item.action] ?? item.action}</span></td>
                  <td><div className="audit-entity"><strong>{entityLabels[item.entity_type] ?? item.entity_type}</strong><small>{shortID(item.entity_id)}</small></div></td>
                  <td><span className="audit-summary">{metadataSummary(item.metadata)}</span></td>
                  <td><code>{item.ip_address || '—'}</code></td>
                  <td><button className="audit-detail-button" type="button" onClick={() => setExpandedID(expanded ? null : item.id)} aria-expanded={expanded} aria-label={`${expanded ? 'Đóng' : 'Xem'} chi tiết sự kiện`} title="Xem chi tiết"><Icon name={expanded ? 'close' : 'eye'} /></button></td>
                </tr>{expanded && <tr className="audit-detail-row"><td colSpan={7}><AuditDetail item={item} /></td></tr>}</Fragment>
              })}</tbody>
            </table></div>
          )}
        </section>
        {!error && !isLoading && logs.length >= 200 && <p className="audit-limit-note">Đang hiển thị 200 sự kiện mới nhất. Dùng bộ lọc để thu hẹp kết quả.</p>}
      </div>
    </AdminShell>
  )
}

function AuditDetail({ item }: { item: AuditLogItem }) {
  const metadata = Object.entries(item.metadata ?? {})
  return <div className="audit-detail-panel">
    <div className="audit-detail-heading"><span><Icon name="audit" /></span><div><strong>Chi tiết sự kiện</strong><small>ID nhật ký: {item.id}</small></div></div>
    <dl className="audit-detail-grid">
      <div><dt>Đối tượng</dt><dd>{entityLabels[item.entity_type] ?? item.entity_type}</dd></div>
      <div><dt>ID đối tượng</dt><dd className="mono">{item.entity_id || 'Không có'}</dd></div>
      <div><dt>Thời điểm chính xác</dt><dd>{formatFullDateTime(item.created_at)}</dd></div>
      <div><dt>Địa chỉ IP</dt><dd className="mono">{item.ip_address || 'Không ghi nhận'}</dd></div>
    </dl>
    <div className="audit-metadata"><strong>Dữ liệu đi kèm</strong>{metadata.length ? <dl>{metadata.map(([key, value]) => <div key={key}><dt>{metadataLabels[key] ?? humanize(key)}</dt><dd>{formatMetadataValue(value)}</dd></div>)}</dl> : <p>Sự kiện này không có dữ liệu đi kèm.</p>}</div>
    <p className="audit-security-note"><Icon name="check" />Các trường nhạy cảm như mật khẩu, token và license key được backend tự động loại khỏi nhật ký.</p>
  </div>
}

function AuditStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'audit' | 'calendar' | 'users' | 'alert'; loading: boolean }) {
  return <article className={`audit-stat ${tone}`}><span><Icon name={icon} /></span><div><small>{label}</small><strong>{loading ? '—' : value}</strong><p>{detail}</p></div></article>
}

function AuditError({ error, onRetry, onLogout }: { error: PageError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  const forbidden = error.status === 403
  return <div className="audit-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : forbidden ? 'Bạn không có quyền xem nhật ký' : 'Không thể tải nhật ký hoạt động'}</strong><p>{expired ? 'Đăng nhập lại để tiếp tục.' : forbidden ? 'Chỉ Admin và Quản lý IT được truy cập module này.' : error.message}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function metadataSummary(metadata: Record<string, unknown>) {
  const preferred = ['name', 'email', 'asset_code', 'status', 'publisher', 'seat_count']
  const key = preferred.find((candidate) => candidate in metadata) ?? Object.keys(metadata)[0]
  if (!key) return 'Không có dữ liệu đi kèm'
  return `${metadataLabels[key] ?? humanize(key)}: ${formatMetadataValue(metadata[key])}`
}

function formatMetadataValue(value: unknown): string {
  if (typeof value === 'boolean') return value ? 'Có' : 'Không'
  if (value === null || value === undefined || value === '') return 'Không có'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function humanize(value: string) {
  return value.replaceAll('_', ' ').replace(/^./, (character) => character.toUpperCase())
}

function initials(value: string) {
  return value.split(/\s|@/).filter(Boolean).slice(0, 2).map((part) => part[0]).join('').toUpperCase() || 'HT'
}

function shortID(value?: string) {
  if (!value) return 'Không có ID'
  return value.length > 12 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value
}

function parsedDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function formatTime(value: string) {
  const date = parsedDate(value)
  return date ? new Intl.DateTimeFormat('vi-VN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(date) : '—'
}

function formatDate(value: string) {
  const date = parsedDate(value)
  return date ? new Intl.DateTimeFormat('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(date) : 'Không xác định'
}

function formatFullDateTime(value: string) {
  const date = parsedDate(value)
  return date ? new Intl.DateTimeFormat('vi-VN', { dateStyle: 'long', timeStyle: 'medium' }).format(date) : 'Không xác định'
}

function sameLocalDay(left: Date, right: Date) {
  return !Number.isNaN(left.getTime()) && left.getFullYear() === right.getFullYear() && left.getMonth() === right.getMonth() && left.getDate() === right.getDate()
}
