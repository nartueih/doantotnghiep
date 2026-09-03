import { useEffect, useMemo, useState } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import { SoftwareCategoryBadge } from '../../components/software/SoftwareCategoryBadge'
import type { AuthSession } from '../../lib/auth-api'
import {
  archiveLicense,
  createLicense,
  getLicenses,
  getSoftwareProducts,
  LicenseAPIError,
  revealLicenseKey,
  updateLicense,
  type LicenseInput,
  type LicenseItem,
  type SoftwareProduct,
} from '../../lib/license-api'
import { LicenseFormDialog } from './LicenseFormDialog'
import './LicenseManagementScreen.css'

interface LicenseManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

type LicenseFilter = 'all' | 'active' | 'expiring' | 'expired' | 'high_usage' | 'exhausted' | 'archived'

const filterLabels: Record<LicenseFilter, string> = {
  all: 'Tất cả trạng thái',
  active: 'Đang hoạt động',
  expiring: 'Sắp hết hạn',
  expired: 'Đã hết hạn',
  high_usage: 'Sử dụng cao',
  exhausted: 'Hết seat',
  archived: 'Đã lưu trữ',
}

export function LicenseManagementScreen({ session, onNavigate, onLogout }: LicenseManagementScreenProps) {
  const [licenses, setLicenses] = useState<LicenseItem[]>([])
  const [products, setProducts] = useState<SoftwareProduct[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<LicenseFilter>('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<LicenseAPIError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [keyDialog, setKeyDialog] = useState<{ license: LicenseItem; key?: string; error?: string; loading: boolean } | null>(null)
  const [formLicense, setFormLicense] = useState<LicenseItem | 'new' | null>(null)
  const [archiveDialog, setArchiveDialog] = useState<{ license: LicenseItem; loading: boolean; error?: string } | null>(null)
  const [successMessage, setSuccessMessage] = useState('')
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)
    Promise.all([
      getLicenses(session.tokens.access_token),
      getSoftwareProducts(session.tokens.access_token),
    ])
      .then(([licenseResult, productResult]) => {
        if (cancelled) return
        setLicenses(licenseResult.items)
        setProducts(productResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        setError(caughtError instanceof LicenseAPIError
          ? caughtError
          : new LicenseAPIError('Đã xảy ra lỗi không mong muốn.', 0))
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })
    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const productMap = useMemo(() => new Map(products.map((product) => [product.id, product])), [products])
  const filteredLicenses = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return licenses.filter((license) => {
      const product = productMap.get(license.software_product_id)
      const matchesSearch = !query || [license.name, license.vendor, product?.name, product?.publisher]
        .some((value) => value?.toLocaleLowerCase('vi').includes(query))
      return matchesSearch && matchesFilter(license, filter)
    })
  }, [filter, licenses, productMap, search])

  const counts = useMemo(() => ({
    total: licenses.length,
    active: licenses.filter((item) => item.lifecycle_status === 'active').length,
    expiring: licenses.filter((item) => item.lifecycle_status !== 'archived' && isExpiring(item)).length,
    attention: licenses.filter((item) => item.lifecycle_status !== 'archived' && (item.lifecycle_status === 'expired' || item.available_seats === 0 || utilization(item) >= 80)).length,
  }), [licenses])

  async function showKey(license: LicenseItem) {
    setCopied(false)
    setKeyDialog({ license, loading: true })
    try {
      const key = await revealLicenseKey(session.tokens.access_token, license.id)
      setKeyDialog({ license, key, loading: false })
    } catch (caughtError) {
      const message = caughtError instanceof LicenseAPIError && caughtError.status === 404
        ? 'License này chưa được cấu hình activation key.'
        : caughtError instanceof Error ? caughtError.message : 'Không thể xem activation key.'
      setKeyDialog({ license, error: message, loading: false })
    }
  }

  async function copyKey() {
    if (!keyDialog?.key) return
    await navigator.clipboard.writeText(keyDialog.key)
    setCopied(true)
  }

  async function saveLicense(input: LicenseInput) {
    if (formLicense === 'new') {
      await createLicense(session.tokens.access_token, input)
      setSuccessMessage('Đã tạo license mới thành công.')
    } else if (formLicense) {
      await updateLicense(session.tokens.access_token, formLicense.id, input)
      setSuccessMessage(`Đã cập nhật ${formLicense.name}.`)
    }
    setFormLicense(null)
    setReloadKey((key) => key + 1)
  }

  async function confirmArchive() {
    if (!archiveDialog) return
    const license = archiveDialog.license
    setArchiveDialog({ license, loading: true })
    try {
      await archiveLicense(session.tokens.access_token, license.id)
      setArchiveDialog(null)
      setSuccessMessage(`Đã lưu trữ ${license.name}. Lịch sử cấp phát vẫn được giữ nguyên.`)
      setReloadKey((key) => key + 1)
    } catch (caughtError) {
      const message = caughtError instanceof Error ? caughtError.message : 'Không thể lưu trữ license.'
      setArchiveDialog({
        license,
        loading: false,
        error: message.includes('active assignments')
          ? 'Hãy thu hồi toàn bộ cấp phát đang hoạt động trước khi lưu trữ license.'
          : message,
      })
    }
  }

  const headerActions = (
    <button className="license-refresh-button" type="button" onClick={() => setReloadKey((key) => key + 1)} disabled={isLoading}>
      <Icon name="refresh" /><span>Làm mới</span>
    </button>
  )

  return (
    <AdminShell
      session={session}
      activePage="licenses"
      title="Quản lý license"
      onNavigate={onNavigate}
      onLogout={onLogout}
      actions={headerActions}
    >
      <div className="license-page">
        <section className="license-page-heading">
          <div><h2>Danh sách license</h2><p>Theo dõi thời hạn, chi phí và mức sử dụng seat của toàn doanh nghiệp.</p></div>
          <div className="license-heading-actions"><span>{licenses.length} license</span><button className="add-license-button" type="button" onClick={() => setFormLicense('new')}><Icon name="plus" />Thêm license</button></div>
        </section>

        {successMessage && <div className="license-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="license-stats" aria-label="Thống kê license">
          <StatCard label="Tổng license" value={counts.total} detail="trong hệ thống" tone="blue" icon="key" loading={isLoading} />
          <StatCard label="Đang hoạt động" value={counts.active} detail="có thể cấp phát" tone="green" icon="check" loading={isLoading} />
          <StatCard label="Sắp hết hạn" value={counts.expiring} detail="trong 90 ngày" tone="amber" icon="calendar" loading={isLoading} />
          <StatCard label="Cần chú ý" value={counts.attention} detail="hết hạn hoặc hết seat" tone="red" icon="alert" loading={isLoading} />
        </section>

        <section className="license-list-card">
          <div className="license-toolbar">
            <div className="license-search">
              <Icon name="search" />
              <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm theo license, phần mềm, nhà cung cấp..." aria-label="Tìm kiếm license" />
            </div>
            <div className="license-filter">
              <Icon name="filter" />
              <select value={filter} onChange={(event) => setFilter(event.target.value as LicenseFilter)} aria-label="Lọc trạng thái license">
                {(Object.keys(filterLabels) as LicenseFilter[]).map((value) => <option value={value} key={value}>{filterLabels[value]}</option>)}
              </select>
            </div>
            <span className="license-result-count">{filteredLicenses.length} kết quả</span>
          </div>

          {error ? (
            <LicenseError error={error} onRetry={() => setReloadKey((key) => key + 1)} onLogout={onLogout} />
          ) : isLoading ? (
            <div className="license-loading" aria-label="Đang tải license">{Array.from({ length: 5 }, (_, index) => <span key={index} />)}</div>
          ) : filteredLicenses.length === 0 ? (
            <div className="license-empty"><span><Icon name="search" /></span><strong>Không tìm thấy license</strong><p>Thử thay đổi từ khóa hoặc bộ lọc trạng thái.</p></div>
          ) : (
            <div className="license-table-scroll">
              <table className="license-table">
                <thead><tr><th>License / Phần mềm</th><th>Loại</th><th>Nhà cung cấp</th><th>Sử dụng seat</th><th>Thời hạn</th><th>Trạng thái</th><th /></tr></thead>
                <tbody>{filteredLicenses.map((license) => (
                  <LicenseRow license={license} product={productMap.get(license.software_product_id)} onShowKey={showKey} onEdit={setFormLicense} onArchive={(item) => setArchiveDialog({ license: item, loading: false })} key={license.id} />
                ))}</tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      {keyDialog && (
        <div className="key-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setKeyDialog(null) }}>
          <section className="key-dialog" role="dialog" aria-modal="true" aria-labelledby="key-dialog-title">
            <button className="key-dialog-close" type="button" onClick={() => setKeyDialog(null)} aria-label="Đóng"><Icon name="close" /></button>
            <span className="key-dialog-icon"><Icon name="key" /></span>
            <h2 id="key-dialog-title">Activation key</h2>
            <p>{keyDialog.license.name}</p>
            {keyDialog.loading ? <span className="key-loading">Đang giải mã an toàn...</span> : keyDialog.error ? (
              <div className="key-error">{keyDialog.error}</div>
            ) : (
              <><code>{keyDialog.key}</code><button className="copy-key-button" type="button" onClick={copyKey}>{copied ? 'Đã sao chép' : 'Sao chép key'}</button></>
            )}
            <small>Thao tác xem key đã được ghi vào Audit Log.</small>
          </section>
        </div>
      )}
      {formLicense && (
        <LicenseFormDialog
          license={formLicense === 'new' ? undefined : formLicense}
          products={products}
          onClose={() => setFormLicense(null)}
          onSubmit={saveLicense}
        />
      )}
      {archiveDialog && (
        <div className="archive-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!archiveDialog.loading && event.target === event.currentTarget) setArchiveDialog(null) }}>
          <section className="archive-dialog" role="dialog" aria-modal="true" aria-labelledby="archive-dialog-title">
            <span className="archive-dialog-icon"><Icon name="archive" /></span>
            <h2 id="archive-dialog-title">Lưu trữ license?</h2>
            <p><strong>{archiveDialog.license.name}</strong> sẽ không thể chỉnh sửa hoặc cấp phát thêm sau khi lưu trữ.</p>
            <div className="archive-dialog-note"><Icon name="check" /><span>Dữ liệu license, activation key và lịch sử audit/cấp phát vẫn được giữ nguyên.</span></div>
            {archiveDialog.error && <div className="archive-dialog-error" role="alert"><Icon name="alert" />{archiveDialog.error}</div>}
            <footer>
              <button type="button" className="archive-cancel" onClick={() => setArchiveDialog(null)} disabled={archiveDialog.loading}>Hủy</button>
              <button type="button" className="archive-confirm" onClick={confirmArchive} disabled={archiveDialog.loading}>{archiveDialog.loading ? 'Đang lưu trữ...' : 'Xác nhận lưu trữ'}</button>
            </footer>
          </section>
        </div>
      )}
    </AdminShell>
  )
}

function StatCard({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'key' | 'check' | 'calendar' | 'alert'; loading: boolean }) {
  return <article className="license-stat"><span className={tone}><Icon name={icon} /></span><div><p>{label}</p>{loading ? <i /> : <strong>{value}</strong>}<small>{detail}</small></div></article>
}

function LicenseRow({ license, product, onShowKey, onEdit, onArchive }: { license: LicenseItem; product?: SoftwareProduct; onShowKey: (license: LicenseItem) => void; onEdit: (license: LicenseItem) => void; onArchive: (license: LicenseItem) => void }) {
  const percent = utilization(license)
  const status = displayStatus(license)
  return (
    <tr className={license.lifecycle_status === 'archived' ? 'archived' : undefined}>
      <td><div className="license-identity"><SoftwareCategoryBadge name={product?.name ?? license.name} publisher={product?.publisher ?? license.vendor} size="compact" /><div><strong>{license.name}</strong><small>{product ? `${product.publisher} · ${product.name}` : 'Chưa xác định phần mềm'}</small></div></div></td>
      <td><span className="license-type">{license.license_type === 'subscription' ? 'Thuê bao' : 'Vĩnh viễn'}</span><small className="assignment-type">{assignmentLabel(license.assignment_type)}</small></td>
      <td><strong className="vendor-name">{license.vendor || '—'}</strong><small className="key-hint">{license.key_hint || 'Chưa có key'}</small><small className={license.allow_employee_key_view ? 'employee-key-policy allowed' : 'employee-key-policy'}>{license.allow_employee_key_view ? 'Nhân viên được xem key' : 'Chỉ IT được xem key'}</small></td>
      <td><div className="seat-value"><strong>{license.used_seats}</strong><span>/ {license.seat_count}</span><small>{Math.round(percent)}%</small></div><span className="license-seat-track"><i className={percent >= 100 ? 'full' : percent >= 80 ? 'high' : ''} style={{ width: `${Math.min(percent, 100)}%` }} /></span></td>
      <td><strong className="expiry-date">{license.expires_at ? formatDate(license.expires_at) : 'Không thời hạn'}</strong><small className="expiry-detail">{expiryDetail(license)}</small></td>
      <td><span className={`license-status ${status.tone}`}><i />{status.label}</span></td>
      <td><div className="license-row-actions">{license.lifecycle_status !== 'archived' && <button className="license-row-action" type="button" onClick={() => onEdit(license)} aria-label={`Chỉnh sửa ${license.name}`} title="Chỉnh sửa"><Icon name="edit" /></button>}<button className="license-row-action" type="button" onClick={() => onShowKey(license)} aria-label={`Xem activation key của ${license.name}`} title="Xem activation key"><Icon name="eye" /></button>{license.lifecycle_status !== 'archived' && <button className="license-row-action archive" type="button" onClick={() => onArchive(license)} disabled={license.used_seats > 0} aria-label={`Lưu trữ ${license.name}`} title={license.used_seats > 0 ? 'Cần thu hồi toàn bộ cấp phát trước' : 'Lưu trữ license'}><Icon name="archive" /></button>}</div></td>
    </tr>
  )
}

function LicenseError({ error, onRetry, onLogout }: { error: LicenseAPIError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const authError = error.status === 401 || error.status === 403
  return <div className="license-error"><Icon name="alert" /><strong>{authError ? 'Không thể truy cập' : 'Không thể tải license'}</strong><p>{error.status === 401 ? 'Phiên đăng nhập đã hết hạn.' : error.status === 403 ? 'Tài khoản không có quyền quản lý license.' : error.status === 0 ? 'Hãy kiểm tra backend đang chạy ở cổng 8080.' : error.message}</p><button type="button" onClick={authError ? onLogout : onRetry}>{authError ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function utilization(license: LicenseItem): number { return license.seat_count > 0 ? (license.used_seats / license.seat_count) * 100 : 0 }
function isExpiring(license: LicenseItem): boolean { const days = daysUntil(license.expires_at); return days !== null && days >= 0 && days <= 90 }
function matchesFilter(license: LicenseItem, filter: LicenseFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'active') return license.lifecycle_status === 'active'
  if (filter === 'expiring') return license.lifecycle_status !== 'archived' && isExpiring(license)
  if (filter === 'expired') return license.lifecycle_status === 'expired'
  if (filter === 'archived') return license.lifecycle_status === 'archived'
  if (filter === 'high_usage') return license.lifecycle_status !== 'archived' && utilization(license) >= 80 && license.available_seats > 0
  return license.lifecycle_status !== 'archived' && license.available_seats === 0
}
function daysUntil(value?: string): number | null { if (!value) return null; const today = new Date(); today.setHours(0, 0, 0, 0); const target = new Date(`${value}T00:00:00`); return Math.round((target.getTime() - today.getTime()) / 86400000) }
function displayStatus(license: LicenseItem): { label: string; tone: string } {
  if (license.lifecycle_status === 'archived') return { label: 'Đã lưu trữ', tone: 'archived' }
  if (license.lifecycle_status === 'expired') return { label: 'Đã hết hạn', tone: 'expired' }
  if (license.lifecycle_status === 'upcoming') return { label: 'Sắp hiệu lực', tone: 'upcoming' }
  if (license.available_seats === 0) return { label: 'Hết seat', tone: 'exhausted' }
  if (isExpiring(license)) return { label: 'Sắp hết hạn', tone: 'expiring' }
  return { label: 'Hoạt động', tone: 'active' }
}
function expiryDetail(license: LicenseItem): string { const days = daysUntil(license.expires_at); if (days === null) return 'License vĩnh viễn'; if (days < 0) return `Quá hạn ${Math.abs(days)} ngày`; if (days === 0) return 'Hết hạn hôm nay'; return `Còn ${days} ngày` }
function assignmentLabel(value: LicenseItem['assignment_type']): string { return value === 'user' ? 'Theo người dùng' : value === 'device' ? 'Theo thiết bị' : 'Hỗn hợp' }
function formatDate(value: string): string { return new Intl.DateTimeFormat('vi-VN').format(new Date(`${value}T00:00:00`)) }
