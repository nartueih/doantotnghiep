import { useCallback, useEffect, useRef, useState } from 'react'
import { Icon } from '../../components/layout/AdminShell'
import { normalizeAPIError } from '../../lib/api-error'
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type WebsiteNotification,
} from '../../lib/license-request-api'

interface EmployeeNotificationBellProps {
  accessToken: string
  onSessionExpired: () => void
}

export function EmployeeNotificationBell({ accessToken, onSessionExpired }: EmployeeNotificationBellProps) {
  const rootRef = useRef<HTMLDivElement>(null)
  const [items, setItems] = useState<WebsiteNotification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionID, setActionID] = useState('')

  const load = useCallback(async (quiet = false) => {
    if (!quiet) setIsLoading(true)
    setError('')
    try {
      const result = await listNotifications(accessToken)
      setItems(result.items)
      setUnreadCount(result.unread_count)
    } catch (caughtError) {
      const normalized = normalizeAPIError(caughtError, 'Không thể tải thông báo.')
      if (normalized.status === 401) onSessionExpired()
      if (!quiet) setError(normalized.message)
    } finally {
      if (!quiet) setIsLoading(false)
    }
  }, [accessToken, onSessionExpired])

  useEffect(() => {
    void load()
    const interval = window.setInterval(() => { void load(true) }, 30_000)
    return () => window.clearInterval(interval)
  }, [load])

  useEffect(() => {
    function closeOnOutsideClick(event: MouseEvent) {
      if (isOpen && rootRef.current && !rootRef.current.contains(event.target as Node)) setIsOpen(false)
    }
    document.addEventListener('mousedown', closeOnOutsideClick)
    return () => document.removeEventListener('mousedown', closeOnOutsideClick)
  }, [isOpen])

  async function openNotification(item: WebsiteNotification) {
    if (item.read_at) return
    setActionID(item.id)
    try {
      const updated = await markNotificationRead(accessToken, item.id)
      setItems((current) => current.map((notification) => notification.id === updated.id ? updated : notification))
      setUnreadCount((current) => Math.max(0, current - 1))
    } catch (caughtError) {
      const normalized = normalizeAPIError(caughtError, 'Không thể cập nhật thông báo.')
      if (normalized.status === 401) onSessionExpired()
      setError(normalized.message)
    } finally {
      setActionID('')
    }
  }

  async function readAll() {
    setActionID('all')
    try {
      await markAllNotificationsRead(accessToken)
      const now = new Date().toISOString()
      setItems((current) => current.map((item) => item.read_at ? item : { ...item, read_at: now }))
      setUnreadCount(0)
    } catch (caughtError) {
      const normalized = normalizeAPIError(caughtError, 'Không thể cập nhật thông báo.')
      if (normalized.status === 401) onSessionExpired()
      setError(normalized.message)
    } finally {
      setActionID('')
    }
  }

  return <div className="employee-notifications" ref={rootRef}>
    <button className="employee-notification-trigger" type="button" aria-label={`Thông báo${unreadCount ? `, ${unreadCount} chưa đọc` : ''}`} aria-expanded={isOpen} onClick={() => { setIsOpen((open) => !open); if (!isOpen) void load() }}>
      <Icon name="bell" />{unreadCount > 0 && <span>{unreadCount > 99 ? '99+' : unreadCount}</span>}
    </button>
    {isOpen && <section className="employee-notification-panel" aria-label="Thông báo của tôi">
      <header><div><strong>Thông báo</strong><span>{unreadCount} chưa đọc</span></div><button type="button" onClick={() => void readAll()} disabled={!unreadCount || actionID === 'all'}>Đánh dấu tất cả đã đọc</button></header>
      {error && <p className="employee-notification-error" role="alert">{error}</p>}
      {isLoading ? <div className="employee-notification-loading"><span /><span /></div> : items.length ? <div className="employee-notification-list">
        {items.map((item) => <button className={item.read_at ? '' : 'unread'} type="button" key={item.id} onClick={() => void openNotification(item)} disabled={actionID === item.id}>
          <span className="employee-notification-icon"><Icon name={notificationIcon(item)} /></span>
          <span><strong>{item.title}</strong><p>{item.message}</p><small>{formatNotificationTime(item.created_at)}</small></span>
          {!item.read_at && <i aria-label="Chưa đọc" />}
        </button>)}
      </div> : <div className="employee-notification-empty"><Icon name="bell" /><strong>Chưa có thông báo</strong><p>Phản hồi từ bộ phận IT sẽ xuất hiện tại đây.</p></div>}
    </section>}
  </div>
}

function notificationIcon(item: WebsiteNotification): 'check' | 'settings' | 'alert' {
  if (item.type === 'license_request_approved' || item.type === 'maintenance_completed') return 'check'
  if (item.type === 'maintenance_accepted') return 'settings'
  return 'alert'
}

function formatNotificationTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: 'short' }).format(date)
}
