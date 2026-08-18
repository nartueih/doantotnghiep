import { useState, type FormEvent } from 'react'
import { Icon } from '../../components/layout/AdminShell'
import type { LicenseInput, LicenseItem, SoftwareProduct } from '../../lib/license-api'

interface LicenseFormDialogProps {
  license?: LicenseItem
  products: SoftwareProduct[]
  onClose: () => void
  onSubmit: (input: LicenseInput) => Promise<void>
}

function initialInput(license?: LicenseItem): LicenseInput {
  return {
    software_product_id: license?.software_product_id ?? '',
    name: license?.name ?? '',
    license_type: license?.license_type ?? 'subscription',
    assignment_type: license?.assignment_type ?? 'user',
    seat_count: license?.seat_count ?? 1,
    license_key: '',
    vendor: license?.vendor ?? '',
    purchased_at: license?.purchased_at ?? '',
    starts_at: license?.starts_at ?? '',
    expires_at: license?.expires_at ?? '',
    cost: license?.cost ?? 0,
    currency: license?.currency ?? 'VND',
    notes: license?.notes ?? '',
  }
}

export function LicenseFormDialog({ license, products, onClose, onSubmit }: LicenseFormDialogProps) {
  const [input, setInput] = useState<LicenseInput>(() => initialInput(license))
  const [showKey, setShowKey] = useState(false)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const editing = Boolean(license)

  function update<K extends keyof LicenseInput>(key: K, value: LicenseInput[K]) {
    setInput((current) => ({ ...current, [key]: value }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const validationError = validate(input, license)
    if (validationError) {
      setError(validationError)
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit({ ...input, currency: input.currency.trim().toUpperCase() })
    } catch (caughtError) {
      setError(translateError(caughtError instanceof Error ? caughtError.message : 'Không thể lưu license.'))
      setIsSubmitting(false)
    }
  }

  return (
    <div className="license-form-backdrop" role="presentation" onMouseDown={(event) => { if (!isSubmitting && event.target === event.currentTarget) onClose() }}>
      <section className="license-form-dialog" role="dialog" aria-modal="true" aria-labelledby="license-form-title">
        <header>
          <div><span><Icon name={editing ? 'edit' : 'plus'} /></span><div><h2 id="license-form-title">{editing ? 'Chỉnh sửa license' : 'Thêm license mới'}</h2><p>{editing ? 'Cập nhật thông tin và số lượng seat.' : 'Tạo license để bắt đầu theo dõi và cấp phát.'}</p></div></div>
          <button type="button" onClick={onClose} disabled={isSubmitting} aria-label="Đóng"><Icon name="close" /></button>
        </header>

        <form onSubmit={handleSubmit}>
          <div className="license-form-body">
            <div className="form-section-title"><span>01</span><div><strong>Thông tin chung</strong><small>Sản phẩm và hình thức license</small></div></div>
            <div className="license-form-grid">
              <label className="full">Sản phẩm phần mềm
                <select value={input.software_product_id} onChange={(event) => update('software_product_id', event.target.value)} disabled={isSubmitting} required>
                  <option value="">Chọn sản phẩm phần mềm — không được bỏ trống</option>
                  {products.map((product) => <option value={product.id} key={product.id}>{product.publisher} — {product.name} {product.version}</option>)}
                </select>
              </label>
              <label className="full">Tên license
                <input value={input.name} onChange={(event) => update('name', event.target.value)} placeholder="Không được bỏ trống — Ví dụ: Microsoft 365 Business Premium" disabled={isSubmitting} required />
              </label>
              <label>Loại license
                <select value={input.license_type} onChange={(event) => update('license_type', event.target.value as LicenseInput['license_type'])} disabled={isSubmitting}>
                  <option value="subscription">Thuê bao</option><option value="perpetual">Vĩnh viễn</option>
                </select>
              </label>
              <label>Đối tượng cấp phát
                <select value={input.assignment_type} onChange={(event) => update('assignment_type', event.target.value as LicenseInput['assignment_type'])} disabled={isSubmitting}>
                  <option value="user">Người dùng</option><option value="device">Thiết bị</option><option value="mixed">Người dùng hoặc thiết bị</option>
                </select>
              </label>
              <label>Số lượng seat
                <input type="number" min={license ? Math.max(1, license.used_seats) : 1} value={input.seat_count} onChange={(event) => update('seat_count', Number(event.target.value))} disabled={isSubmitting} required />
                {license && <small>Đang dùng {license.used_seats} seat</small>}
              </label>
              <label>Nhà cung cấp
                <input value={input.vendor} onChange={(event) => update('vendor', event.target.value)} placeholder="Tên vendor hoặc đối tác" disabled={isSubmitting} />
              </label>
            </div>

            <div className="form-section-title"><span>02</span><div><strong>Thời hạn và chi phí</strong><small>Ngày hiệu lực và ngân sách</small></div></div>
            <div className="license-form-grid three">
              <label>Ngày mua<input type="date" value={input.purchased_at} onChange={(event) => update('purchased_at', event.target.value)} disabled={isSubmitting} /></label>
              <label>Ngày bắt đầu<input type="date" value={input.starts_at} onChange={(event) => update('starts_at', event.target.value)} disabled={isSubmitting} /></label>
              <label>Ngày hết hạn<input type="date" value={input.expires_at} onChange={(event) => update('expires_at', event.target.value)} disabled={isSubmitting || input.license_type === 'perpetual'} required={input.license_type === 'subscription'} />{input.license_type === 'subscription' && <small>Không được bỏ trống với license thuê bao</small>}</label>
              <label className="cost-field">Chi phí<input type="number" min="0" step="1" value={input.cost} onChange={(event) => update('cost', Number(event.target.value))} disabled={isSubmitting} /></label>
              <label>Tiền tệ<input maxLength={3} value={input.currency} onChange={(event) => update('currency', event.target.value)} placeholder="VND" disabled={isSubmitting} /></label>
            </div>

            <div className="form-section-title"><span>03</span><div><strong>Bảo mật và ghi chú</strong><small>Activation key được mã hóa tại backend</small></div></div>
            <div className="license-form-grid">
              <label className="full">Activation key
                <div className="license-key-input"><input type={showKey ? 'text' : 'password'} value={input.license_key} onChange={(event) => update('license_key', event.target.value)} placeholder={editing ? 'Để trống để giữ key hiện tại' : 'Nhập activation key nếu có'} autoComplete="off" disabled={isSubmitting} /><button type="button" onClick={() => setShowKey((visible) => !visible)}>{showKey ? 'Ẩn' : 'Hiện'}</button></div>
                <small>{editing && license?.key_hint ? `Key hiện tại: ${license.key_hint}` : 'Key sẽ không xuất hiện trong API danh sách.'}</small>
              </label>
              <label className="full">Ghi chú<textarea rows={3} value={input.notes} onChange={(event) => update('notes', event.target.value)} placeholder="Thông tin gia hạn, hợp đồng hoặc phạm vi sử dụng..." disabled={isSubmitting} /></label>
            </div>

            {error && <div className="license-form-error" role="alert"><Icon name="alert" />{error}</div>}
          </div>
          <footer><button type="button" className="form-cancel" onClick={onClose} disabled={isSubmitting}>Hủy</button><button type="submit" className="form-submit" disabled={isSubmitting}>{isSubmitting ? 'Đang lưu...' : editing ? 'Lưu thay đổi' : 'Tạo license'}</button></footer>
        </form>
      </section>
    </div>
  )
}

function validate(input: LicenseInput, license?: LicenseItem): string {
  if (!input.software_product_id || !input.name.trim()) return 'Vui lòng chọn sản phẩm và nhập tên license.'
  if (!Number.isInteger(input.seat_count) || input.seat_count <= 0) return 'Số lượng seat phải là số nguyên lớn hơn 0.'
  if (license && input.seat_count < license.used_seats) return `Không thể giảm xuống dưới ${license.used_seats} seat đang sử dụng.`
  if (input.license_type === 'subscription' && !input.expires_at) return 'License thuê bao phải có ngày hết hạn.'
  if (input.starts_at && input.expires_at && input.expires_at < input.starts_at) return 'Ngày hết hạn không được trước ngày bắt đầu.'
  if (input.purchased_at && input.expires_at && input.expires_at < input.purchased_at) return 'Ngày hết hạn không được trước ngày mua.'
  if (input.cost < 0) return 'Chi phí không được âm.'
  if ((input.cost > 0 || input.currency) && input.currency.trim().length !== 3) return 'Mã tiền tệ phải có đúng 3 ký tự, ví dụ VND.'
  return ''
}

function translateError(message: string): string {
  if (message.includes('seat count cannot be lower')) return 'Số seat không được thấp hơn số lượt cấp phát đang hoạt động.'
  if (message.includes('expiration date')) return 'Ngày hết hạn không hợp lệ hoặc đang thiếu.'
  if (message.includes('software product')) return 'Sản phẩm phần mềm đã chọn không tồn tại.'
  if (message.includes('license data is invalid')) return 'Thông tin license chưa hợp lệ.'
  return message
}
