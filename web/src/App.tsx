import { type FormEvent, useState } from 'react'
import './App.css'
import {
  APIError,
  type AuthSession,
  login,
  logout,
  type UserRole,
} from './lib/auth-api'

const SESSION_KEY = 'enterprise-license-manager.session'

const roleLabels: Record<UserRole, string> = {
  admin: 'Quản trị viên',
  it_manager: 'Quản lý IT',
  employee: 'Nhân viên',
}

function readSession(): AuthSession | null {
  const value = sessionStorage.getItem(SESSION_KEY)
  if (!value) return null

  try {
    return JSON.parse(value) as AuthSession
  } catch {
    sessionStorage.removeItem(SESSION_KEY)
    return null
  }
}

function App() {
  const [session, setSession] = useState<AuthSession | null>(readSession)

  function handleAuthenticated(nextSession: AuthSession) {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(nextSession))
    setSession(nextSession)
  }

  async function handleLogout() {
    if (session) await logout(session.tokens.refresh_token)
    sessionStorage.removeItem(SESSION_KEY)
    setSession(null)
  }

  if (session) {
    return <WelcomeScreen session={session} onLogout={handleLogout} />
  }

  return <LoginScreen onAuthenticated={handleAuthenticated} />
}

interface LoginScreenProps {
  onAuthenticated: (session: AuthSession) => void
}

function LoginScreen({ onAuthenticated }: LoginScreenProps) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)

    try {
      onAuthenticated(await login(email.trim(), password))
    } catch (caughtError) {
      if (caughtError instanceof APIError) {
        const messages: Record<number, string> = {
          401: 'Email hoặc mật khẩu không chính xác.',
          403: 'Tài khoản này đang bị khóa.',
        }
        setError(messages[caughtError.status] ?? caughtError.message)
      } else {
        setError('Đã xảy ra lỗi không mong muốn. Vui lòng thử lại.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  function fillDevelopmentAccount() {
    setEmail('admin@local.test')
    setPassword('ChangeMe123!')
    setError('')
  }

  return (
    <main className="auth-page">
      <section className="brand-panel" aria-label="Giới thiệu hệ thống">
        <div className="brand-lockup">
          <span className="brand-mark" aria-hidden="true">LM</span>
          <div>
            <strong>License Manager</strong>
            <span>Enterprise workspace</span>
          </div>
        </div>

        <div className="brand-message">
          <span className="eyebrow">Quản trị tập trung</span>
          <h1>Mọi license.<br />Một nơi quản lý.</h1>
          <p>
            Theo dõi phần mềm, thiết bị và quyền sử dụng trong toàn doanh nghiệp
            với dữ liệu rõ ràng, an toàn và luôn sẵn sàng.
          </p>
        </div>

        <div className="feature-list" aria-label="Tính năng nổi bật">
          <span><i aria-hidden="true">01</i> Kiểm soát số lượng seat</span>
          <span><i aria-hidden="true">02</i> Cảnh báo ngày hết hạn</span>
          <span><i aria-hidden="true">03</i> Lịch sử thao tác minh bạch</span>
        </div>

        <div className="brand-decoration" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
      </section>

      <section className="login-panel">
        <div className="login-card">
          <div className="mobile-brand">
            <span className="brand-mark" aria-hidden="true">LM</span>
            <strong>License Manager</strong>
          </div>

          <div className="login-heading">
            <span className="status-dot"><i /> Hệ thống sẵn sàng</span>
            <h2>Chào mừng trở lại</h2>
            <p>Đăng nhập để tiếp tục vào không gian quản trị.</p>
          </div>

          <form onSubmit={handleSubmit} noValidate>
            <label htmlFor="email">Email công việc</label>
            <div className="input-wrap">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 6h16v12H4zM4 7l8 6 8-6" />
              </svg>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="username"
                placeholder="ten@congty.vn"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                disabled={isSubmitting}
                required
              />
            </div>

            <div className="label-row">
              <label htmlFor="password">Mật khẩu</label>
              <button type="button" className="text-button" onClick={fillDevelopmentAccount}>
                Dùng tài khoản thử
              </button>
            </div>
            <div className="input-wrap">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <rect x="5" y="10" width="14" height="10" rx="2" />
                <path d="M8 10V7a4 4 0 018 0v3" />
              </svg>
              <input
                id="password"
                name="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="Nhập mật khẩu"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                disabled={isSubmitting}
                required
              />
              <button
                type="button"
                className="password-toggle"
                onClick={() => setShowPassword((visible) => !visible)}
                aria-label={showPassword ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'}
              >
                {showPassword ? 'Ẩn' : 'Hiện'}
              </button>
            </div>

            {error && <div className="error-message" role="alert">{error}</div>}

            <button className="primary-button" type="submit" disabled={isSubmitting || !email || !password}>
              {isSubmitting ? <><span className="spinner" /> Đang xác thực...</> : <>Đăng nhập <span>→</span></>}
            </button>
          </form>

          <p className="security-note">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 3l7 3v5c0 4.5-2.8 8.1-7 10-4.2-1.9-7-5.5-7-10V6z" />
            </svg>
            Phiên đăng nhập được bảo vệ và tự xóa khi đóng tab.
          </p>
        </div>
      </section>
    </main>
  )
}

interface WelcomeScreenProps {
  session: AuthSession
  onLogout: () => Promise<void>
}

function WelcomeScreen({ session, onLogout }: WelcomeScreenProps) {
  const initials = session.user.full_name
    .split(' ')
    .slice(-2)
    .map((part) => part[0])
    .join('')
    .toUpperCase()

  return (
    <main className="welcome-page">
      <header className="app-header">
        <div className="brand-lockup dark">
          <span className="brand-mark" aria-hidden="true">LM</span>
          <div>
            <strong>License Manager</strong>
            <span>Enterprise workspace</span>
          </div>
        </div>
        <button className="secondary-button" type="button" onClick={onLogout}>Đăng xuất</button>
      </header>

      <section className="welcome-content">
        <div className="welcome-card">
          <div className="avatar">{initials || 'U'}</div>
          <span className="eyebrow blue">Đăng nhập thành công</span>
          <h1>Xin chào, {session.user.full_name}</h1>
          <p>
            Nền móng xác thực của web quản trị đã kết nối thành công với Go backend.
            Tiếp theo chúng ta sẽ xây dựng bố cục dashboard và điều hướng theo vai trò.
          </p>
          <dl className="account-summary">
            <div><dt>Email</dt><dd>{session.user.email}</dd></div>
            <div><dt>Vai trò</dt><dd>{roleLabels[session.user.role]}</dd></div>
            <div><dt>Mã nhân viên</dt><dd>{session.user.employee_code || 'Chưa cập nhật'}</dd></div>
          </dl>
        </div>
      </section>
    </main>
  )
}

export default App
