import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { AdminShell, Icon, type AdminPage } from '../../components/layout/AdminShell'
import { SoftwareCategoryBadge } from '../../components/software/SoftwareCategoryBadge'
import type { AuthSession } from '../../lib/auth-api'
import { getLicenses, LicenseAPIError, type LicenseItem } from '../../lib/license-api'
import {
  createSoftwareProduct,
  getSoftwareProducts,
  SoftwareAPIError,
  updateSoftwareProduct,
  type SoftwareInput,
  type SoftwareProduct,
} from '../../lib/software-api'
import './SoftwareManagementScreen.css'

interface SoftwareManagementScreenProps {
  session: AuthSession
  onNavigate: (page: AdminPage) => void
  onLogout: () => Promise<void>
}

interface PageError {
  message: string
  status: number
}

export function SoftwareManagementScreen({ session, onNavigate, onLogout }: SoftwareManagementScreenProps) {
  const [products, setProducts] = useState<SoftwareProduct[]>([])
  const [licenses, setLicenses] = useState<LicenseItem[]>([])
  const [search, setSearch] = useState('')
  const [publisherFilter, setPublisherFilter] = useState('all')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<PageError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [formProduct, setFormProduct] = useState<SoftwareProduct | 'new' | null>(null)
  const [successMessage, setSuccessMessage] = useState('')
  const canManage = session.user.role === 'admin' || session.user.role === 'it_manager'

  useEffect(() => {
    let cancelled = false
    setIsLoading(true)
    setError(null)

    Promise.all([
      getSoftwareProducts(session.tokens.access_token),
      getLicenses(session.tokens.access_token),
    ])
      .then(([productResult, licenseResult]) => {
        if (cancelled) return
        setProducts(productResult.items)
        setLicenses(licenseResult.items)
      })
      .catch((caughtError: unknown) => {
        if (cancelled) return
        if (caughtError instanceof SoftwareAPIError || caughtError instanceof LicenseAPIError) {
          setError({ message: caughtError.message, status: caughtError.status })
        } else {
          setError({ message: 'Đã xảy ra lỗi không mong muốn.', status: 0 })
        }
      })
      .finally(() => { if (!cancelled) setIsLoading(false) })

    return () => { cancelled = true }
  }, [reloadKey, session.tokens.access_token])

  const activeLicenses = useMemo(() => licenses.filter((license) => license.lifecycle_status !== 'archived'), [licenses])
  const licensesByProduct = useMemo(() => {
    const result = new Map<string, LicenseItem[]>()
    for (const license of activeLicenses) {
      result.set(license.software_product_id, [...(result.get(license.software_product_id) ?? []), license])
    }
    return result
  }, [activeLicenses])

  const publishers = useMemo(() => [...new Set(products.map((product) => product.publisher))].sort((a, b) => a.localeCompare(b, 'vi')), [products])
  const filteredProducts = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('vi')
    return products.filter((product) => {
      const productLicenses = licensesByProduct.get(product.id) ?? []
      const matchesSearch = !query || [product.name, product.publisher, product.version, product.description, ...productLicenses.map((license) => license.name)]
        .some((value) => value.toLocaleLowerCase('vi').includes(query))
      return matchesSearch && (publisherFilter === 'all' || product.publisher === publisherFilter)
    })
  }, [licensesByProduct, products, publisherFilter, search])

  const totals = useMemo(() => ({
    products: products.length,
    publishers: publishers.length,
    licenses: activeLicenses.length,
    seats: activeLicenses.reduce((sum, license) => sum + license.seat_count, 0),
    usedSeats: activeLicenses.reduce((sum, license) => sum + license.used_seats, 0),
  }), [activeLicenses, products.length, publishers.length])

  async function saveProduct(input: SoftwareInput) {
    if (formProduct === 'new') {
      const created = await createSoftwareProduct(session.tokens.access_token, input)
      setSuccessMessage(`Đã thêm phần mềm ${created.name}.`)
    } else if (formProduct) {
      const updated = await updateSoftwareProduct(session.tokens.access_token, formProduct.id, input)
      setSuccessMessage(`Đã cập nhật phần mềm ${updated.name}.`)
    }
    setFormProduct(null)
    setReloadKey((value) => value + 1)
  }

  const headerActions = <button className="software-refresh-button" type="button" onClick={() => setReloadKey((value) => value + 1)} disabled={isLoading}><Icon name="refresh" /><span>Làm mới</span></button>

  return (
    <AdminShell session={session} activePage="software" title="Quản lý phần mềm" onNavigate={onNavigate} onLogout={onLogout} actions={headerActions}>
      <div className="software-page">
        <section className="software-page-heading">
          <div><h2>Danh mục phần mềm</h2><p>Quản lý sản phẩm, nhà phát hành, phiên bản và các license đang liên kết.</p></div>
          {canManage && <button className="add-software-button" type="button" onClick={() => setFormProduct('new')}><Icon name="plus" />Thêm phần mềm</button>}
        </section>

        {successMessage && <div className="software-success" role="status"><Icon name="check" />{successMessage}<button type="button" onClick={() => setSuccessMessage('')} aria-label="Đóng thông báo"><Icon name="close" /></button></div>}

        <section className="software-stats" aria-label="Thống kê phần mềm">
          <SoftwareStat label="Tổng sản phẩm" value={totals.products} detail="phần mềm trong danh mục" tone="blue" icon="software" loading={isLoading} />
          <SoftwareStat label="Nhà phát hành" value={totals.publishers} detail="đối tác phần mềm" tone="violet" icon="department" loading={isLoading} />
          <SoftwareStat label="License liên kết" value={totals.licenses} detail="license chưa lưu trữ" tone="green" icon="key" loading={isLoading} />
          <SoftwareStat label="Tổng số seat" value={totals.seats} detail={`${totals.usedSeats} seat đang sử dụng`} tone="amber" icon="users" loading={isLoading} />
        </section>

        <section className="software-list-card">
          <div className="software-toolbar">
            <div className="software-search"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm phần mềm, publisher, phiên bản hoặc license..." aria-label="Tìm kiếm phần mềm" /></div>
            <div className="software-filter"><Icon name="filter" /><select value={publisherFilter} onChange={(event) => setPublisherFilter(event.target.value)} aria-label="Lọc nhà phát hành"><option value="all">Tất cả nhà phát hành</option>{publishers.map((publisher) => <option value={publisher} key={publisher}>{publisher}</option>)}</select></div>
            <span className="software-result-count">{filteredProducts.length} kết quả</span>
          </div>

          {error ? <SoftwareError error={error} onRetry={() => setReloadKey((value) => value + 1)} onLogout={onLogout} /> : isLoading ? (
            <div className="software-loading" aria-label="Đang tải phần mềm">{Array.from({ length: 6 }, (_, index) => <span key={index} />)}</div>
          ) : filteredProducts.length === 0 ? (
            <div className="software-empty"><span><Icon name="software" /></span><strong>Không tìm thấy phần mềm</strong><p>Thử thay đổi từ khóa, bộ lọc hoặc thêm sản phẩm mới.</p></div>
          ) : (
            <div className="software-table-scroll"><table className="software-table">
              <thead><tr><th>Phần mềm</th><th>Nhà phát hành</th><th>Phiên bản</th><th>License liên kết</th><th>Mức sử dụng seat</th><th>Cập nhật</th><th /></tr></thead>
              <tbody>{filteredProducts.map((product) => <SoftwareRow product={product} licenses={licensesByProduct.get(product.id) ?? []} canManage={canManage} onEdit={setFormProduct} key={product.id} />)}</tbody>
            </table></div>
          )}
        </section>
      </div>

      {formProduct && <SoftwareFormDialog product={formProduct === 'new' ? undefined : formProduct} onClose={() => setFormProduct(null)} onSubmit={saveProduct} />}
    </AdminShell>
  )
}

function SoftwareRow({ product, licenses, canManage, onEdit }: { product: SoftwareProduct; licenses: LicenseItem[]; canManage: boolean; onEdit: (product: SoftwareProduct) => void }) {
  const seats = licenses.reduce((sum, license) => sum + license.seat_count, 0)
  const usedSeats = licenses.reduce((sum, license) => sum + license.used_seats, 0)
  const usagePercent = seats > 0 ? Math.round((usedSeats / seats) * 100) : 0
  return <tr>
    <td><div className="software-identity"><SoftwareCategoryBadge name={product.name} publisher={product.publisher} /><div><strong>{product.name}</strong><small>{product.description || 'Chưa có mô tả'}</small></div></div></td>
    <td><strong className="software-publisher">{product.publisher}</strong></td>
    <td><span className={product.version ? 'software-version' : 'software-version muted'}>{product.version || 'Chưa xác định'}</span></td>
    <td>{licenses.length ? <div className="software-licenses"><strong>{licenses.length} license</strong><small>{licenses.slice(0, 2).map((license) => license.name).join(', ')}{licenses.length > 2 ? '…' : ''}</small></div> : <span className="software-no-license">Chưa có license</span>}</td>
    <td>{seats > 0 ? <div className="software-usage"><div><strong>{usedSeats}/{seats}</strong><span>{usagePercent}%</span></div><span><i style={{ width: `${Math.min(usagePercent, 100)}%` }} /></span></div> : <span className="software-no-license">Không áp dụng</span>}</td>
    <td><span className="software-updated">{formatDate(product.updated_at)}</span></td>
    <td>{canManage && <div className="software-row-actions"><button type="button" onClick={() => onEdit(product)} aria-label={`Chỉnh sửa ${product.name}`} title="Chỉnh sửa phần mềm"><Icon name="edit" /></button></div>}</td>
  </tr>
}

function SoftwareFormDialog({ product, onClose, onSubmit }: { product?: SoftwareProduct; onClose: () => void; onSubmit: (input: SoftwareInput) => Promise<void> }) {
  const [input, setInput] = useState<SoftwareInput>({ name: product?.name ?? '', publisher: product?.publisher ?? '', version: product?.version ?? '', description: product?.description ?? '' })
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function update<K extends keyof SoftwareInput>(key: K, value: SoftwareInput[K]) {
    setInput((current) => ({ ...current, [key]: value }))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!input.name.trim() || !input.publisher.trim()) {
      setError('Vui lòng nhập đầy đủ tên phần mềm và nhà phát hành.')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit({ name: input.name.trim(), publisher: input.publisher.trim(), version: input.version.trim(), description: input.description.trim() })
    } catch (caughtError) {
      setError(translateSoftwareError(caughtError instanceof Error ? caughtError.message : 'Không thể lưu phần mềm.'))
      setIsSubmitting(false)
    }
  }

  return <div className="software-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}><section className="software-form-dialog" role="dialog" aria-modal="true" aria-labelledby="software-form-title">
    <header><div><span><Icon name={product ? 'edit' : 'plus'} /></span><div><h2 id="software-form-title">{product ? 'Chỉnh sửa phần mềm' : 'Thêm phần mềm'}</h2><p>{product ? 'Cập nhật thông tin sản phẩm trong danh mục.' : 'Tạo sản phẩm để liên kết với license.'}</p></div></div><button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button></header>
    <form onSubmit={submit}><div className="software-form-body">
      <div className="software-form-grid"><label className="full">Tên phần mềm<input value={input.name} onChange={(event) => update('name', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: Adobe Photoshop" disabled={isSubmitting} autoFocus required /></label><label>Nhà phát hành<input value={input.publisher} onChange={(event) => update('publisher', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: Adobe" disabled={isSubmitting} required /></label><label>Phiên bản<input value={input.version} onChange={(event) => update('version', event.target.value)} placeholder="Ví dụ: 2026 hoặc Enterprise" disabled={isSubmitting} /></label><label className="full">Mô tả<textarea value={input.description} onChange={(event) => update('description', event.target.value)} placeholder="Mô tả ngắn về mục đích sử dụng phần mềm" disabled={isSubmitting} rows={4} /></label></div>
      {error && <div className="software-dialog-error" role="alert"><Icon name="alert" />{error}</div>}
    </div><footer><button className="software-cancel" type="button" onClick={onClose} disabled={isSubmitting}>Hủy</button><button className="software-submit" type="submit" disabled={isSubmitting}>{isSubmitting ? 'Đang lưu...' : product ? 'Lưu thay đổi' : 'Thêm phần mềm'}</button></footer></form>
  </section></div>
}

function SoftwareStat({ label, value, detail, tone, icon, loading }: { label: string; value: number; detail: string; tone: string; icon: 'software' | 'department' | 'key' | 'users'; loading: boolean }) {
  return <article className={`software-stat ${tone}`}><span><Icon name={icon} /></span><div><small>{label}</small><strong>{loading ? '—' : value}</strong><p>{detail}</p></div></article>
}

function SoftwareError({ error, onRetry, onLogout }: { error: PageError; onRetry: () => void; onLogout: () => Promise<void> }) {
  const expired = error.status === 401
  const forbidden = error.status === 403
  return <div className="software-error"><Icon name="alert" /><strong>{expired ? 'Phiên đăng nhập đã hết hạn' : forbidden ? 'Bạn không có quyền xem phần mềm' : 'Không thể tải danh mục phần mềm'}</strong><p>{expired ? 'Đăng nhập lại để tiếp tục.' : forbidden ? 'Chỉ Admin và Quản lý IT được truy cập module này.' : translateSoftwareError(error.message)}</p><button type="button" onClick={expired ? onLogout : onRetry}>{expired ? 'Đăng nhập lại' : 'Thử lại'}</button></div>
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : new Intl.DateTimeFormat('vi-VN').format(date)
}

function translateSoftwareError(message: string) {
  const translations: Record<string, string> = {
    'software product already exists': 'Sản phẩm cùng tên, nhà phát hành và phiên bản đã tồn tại.',
    'software name and publisher are required': 'Tên phần mềm và nhà phát hành không được bỏ trống.',
    'software product not found': 'Sản phẩm phần mềm không còn tồn tại.',
  }
  return translations[message.toLocaleLowerCase('en')] ?? message
}
