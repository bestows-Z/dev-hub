import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowRight, Eye, EyeOff, LogOut, UserRound } from 'lucide-react'
import { ApiError, api, formatDate } from './api'
import {
  clearSession,
  readToken,
  readUser,
  refreshSessionUser,
  saveSession,
  type AuthUser,
  type LoginResult,
} from './session'
import './auth.css'

function AuthShell({ children, mode }: { children: ReactNode; mode: 'login' | 'register' }) {
  return (
    <div className="auth-page container">
      <div className="auth-scene" aria-hidden="true">
        <div className="auth-scene-inner">
          <span>DEVHUB / PERSONAL JOURNAL</span>
          <p>{mode === 'login' ? '好久不见，欢迎回来。' : '从这里开始，留下你的足迹。'}</p>
          <small>写点东西，做点项目，记下路上的发现。</small>
        </div>
      </div>
      <div className="auth-paper">{children}</div>
    </div>
  )
}

function PasswordField({
  value,
  onChange,
  id = 'auth-password',
  autoComplete = 'current-password',
}: {
  value: string
  onChange: (value: string) => void
  id?: string
  autoComplete?: string
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="auth-password-wrap">
      <input
        id={id}
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        autoComplete={autoComplete}
        minLength={8}
        required
        placeholder="至少 8 位"
      />
      <button
        type="button"
        onClick={() => setVisible(!visible)}
        aria-label={visible ? '隐藏密码' : '显示密码'}
      >
        {visible ? <EyeOff size={18} /> : <Eye size={18} />}
      </button>
    </div>
  )
}

export function Login() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [identifier, setIdentifier] = useState(params.get('identifier') || '')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const result = await api<LoginResult>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ identifier: identifier.trim(), password }),
      })
      saveSession(result)
      navigate(result.user.role === 1 && params.get('next') === 'admin' ? '/admin' : '/account', {
        replace: true,
      })
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : '登录失败，请稍后再试。')
    } finally {
      setBusy(false)
    }
  }
  return (
    <AuthShell mode="login">
      <span className="auth-overline">账户 / 登录</span>
      <h1>欢迎回来</h1>
      <p className="auth-subtitle">登录后可以管理自己的账户，站长也可以进入内容管理。</p>
      <form className="auth-form" onSubmit={submit}>
        <label htmlFor="auth-identifier">用户名或邮箱</label>
        <input
          id="auth-identifier"
          value={identifier}
          onChange={(event) => setIdentifier(event.target.value)}
          autoComplete="username"
          required
          placeholder="你的用户名或邮箱"
        />
        <label htmlFor="auth-password">密码</label>
        <PasswordField value={password} onChange={setPassword} />
        {error && (
          <p className="auth-error" role="alert">
            {error}
          </p>
        )}
        <button className="auth-submit" type="submit" disabled={busy}>
          {busy ? '正在登录…' : '登录'} <ArrowRight size={18} />
        </button>
      </form>
      <p className="auth-switch">
        还没有账号？<Link to="/register">去注册</Link>
      </p>
    </AuthShell>
  )
}

export function Register() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [registered, setRegistered] = useState(false)
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (registered) {
      navigate(`/login?identifier=${encodeURIComponent(username.trim())}`)
      return
    }
    setError('')
    if (!/^[a-zA-Z0-9_]{3,32}$/.test(username.trim())) {
      setError('用户名需为 3–32 位英文字母、数字或下划线。')
      return
    }
    if (password !== confirm) {
      setError('两次输入的密码不一致。')
      return
    }
    if (new TextEncoder().encode(password).length > 72) {
      setError('密码不能超过 72 字节。')
      return
    }
    setBusy(true)
    try {
      await api<AuthUser>('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ username: username.trim(), email: email.trim(), password }),
      })
      setRegistered(true)
      try {
        const result = await api<LoginResult>('/auth/login', {
          method: 'POST',
          body: JSON.stringify({ identifier: username.trim(), password }),
        })
        saveSession(result)
        navigate('/account', { replace: true })
      } catch {
        setError('账号已创建，但自动登录未成功。请前往登录页重试。')
      }
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : '注册失败，请稍后再试。')
    } finally {
      setBusy(false)
    }
  }
  return (
    <AuthShell mode="register">
      <span className="auth-overline">账户 / 注册</span>
      <h1>在这里认识你</h1>
      <p className="auth-subtitle">创建账号后，你就能登录本站。注册不会获得管理权限。</p>
      <form className="auth-form" onSubmit={submit}>
        <label htmlFor="register-username">用户名</label>
        <input
          id="register-username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          autoComplete="username"
          minLength={3}
          maxLength={32}
          required
          placeholder="3–32 位字母、数字或下划线"
        />
        <label htmlFor="register-email">邮箱</label>
        <input
          id="register-email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="email"
          maxLength={255}
          required
          placeholder="you@example.com"
        />
        <label htmlFor="register-password">密码</label>
        <PasswordField
          id="register-password"
          value={password}
          onChange={setPassword}
          autoComplete="new-password"
        />
        <label htmlFor="register-confirm">确认密码</label>
        <PasswordField
          id="register-confirm"
          value={confirm}
          onChange={setConfirm}
          autoComplete="new-password"
        />
        {error && (
          <p className="auth-error" role="alert">
            {error}
          </p>
        )}
        <button className="auth-submit" type="submit" disabled={busy}>
          {busy ? '正在创建…' : registered ? '前往登录' : '创建账号'} <ArrowRight size={18} />
        </button>
      </form>
      <p className="auth-switch">
        已经有账号？<Link to="/login">去登录</Link>
      </p>
    </AuthShell>
  )
}

export function Account() {
  const navigate = useNavigate()
  const token = readToken()
  const [user, setUser] = useState<AuthUser | null>(readUser)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(Boolean(token))
  const [retry, setRetry] = useState(0)
  useEffect(() => {
    if (!token) return
    let active = true
    setLoading(true)
    api<AuthUser>('/auth/me', { headers: { Authorization: `Bearer ${token}` } })
      .then((freshUser) => {
        if (!active) return
        setUser(freshUser)
        refreshSessionUser(freshUser)
        setError('')
      })
      .catch((failure) => {
        if (!active) return
        if (failure instanceof ApiError && failure.status === 401) {
          clearSession()
          setUser(null)
          setError('登录已失效，请重新登录。')
        } else {
          setError(failure instanceof Error ? failure.message : '资料暂时无法读取，请重试。')
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [token, retry])
  if (!token && !error) return <Navigate to="/login" replace />
  function logout() {
    clearSession()
    navigate('/login', { replace: true })
  }
  return (
    <div className="account-page container">
      <span className="auth-overline">账户 / 我的资料</span>
      <div className="account-card">
        <div className="account-avatar">
          <UserRound size={34} />
        </div>
        <div className="account-details">
          <h1>{user?.username || (loading ? '正在读取账户…' : '账户暂不可用')}</h1>
          <p>{user?.email || (loading ? '正在确认登录状态' : error)}</p>
          {user && (
            <div className="account-meta">
              <span>{user.role === 1 ? '站点管理员' : '读者'}</span>
              <span>加入于 {formatDate(user.created_at)}</span>
            </div>
          )}
        </div>
        {user && (
          <button type="button" className="account-logout" onClick={logout}>
            <LogOut size={17} /> 退出登录
          </button>
        )}
      </div>
      {error && token && (
        <button
          className="account-retry"
          type="button"
          onClick={() => setRetry((value) => value + 1)}
        >
          资料刷新失败，点击重试
        </button>
      )}
      {user?.role === 1 && !loading && !error && (
        <Link className="account-admin" to="/admin">
          进入内容管理 <ArrowRight size={17} />
        </Link>
      )}
      {!user && !loading && !token && (
        <Link className="account-admin" to="/login">
          重新登录 <ArrowRight size={17} />
        </Link>
      )}
    </div>
  )
}
