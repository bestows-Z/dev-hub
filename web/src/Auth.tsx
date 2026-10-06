import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowRight, Eye, EyeOff, LogOut, Save, ShoppingBag, UploadCloud, UserRound } from 'lucide-react'
import { ApiError, api, formatDate, formatPrice, type Page } from './api'
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

type MyOrder = {
  id: number
  order_no: string
  product_name: string
  quantity: number
  total_cents: number
  status: string
  created_at: string
}
type AuthorApplication = { id: number; reason: string; status: 'pending' | 'approved' | 'rejected'; created_at: string }

function AuthShell({ children, mode }: { children: ReactNode; mode: 'login' | 'register' }) {
  return (
    <div className="auth-page container" data-mode={mode}>
      <div className="auth-scene" aria-hidden="true">
        <span className="auth-scene-ring" />
        <span className="auth-scene-ring second" />
        <div className="auth-scene-inner">
          <span>一个正在更新的个人站点</span>
          <p>{mode === 'login' ? '好久不见，欢迎回来。' : '从这里开始，留下你的足迹。'}</p>
          <small>文章、项目，还有偶尔分享的小东西。</small>
        </div>
      </div>
      <div className="auth-paper">
        <nav className="auth-tabs" aria-label="账户入口">
          <Link to="/login" viewTransition className={mode === 'login' ? 'active' : ''} aria-current={mode === 'login' ? 'page' : undefined}>登录</Link>
          <Link to="/register" viewTransition className={mode === 'register' ? 'active' : ''} aria-current={mode === 'register' ? 'page' : undefined}>注册</Link>
        </nav>
        {children}
      </div>
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
      const next = params.get('next')
      const destination = result.user.role === 1 && next === 'admin'
        ? '/admin'
        : next?.startsWith('/') && !next.startsWith('//') && !next.startsWith('/admin')
          ? next
          : '/account'
      navigate(destination, {
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
        还没有账号？<Link to="/register" viewTransition>去注册</Link>
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
        已经有账号？<Link to="/login" viewTransition>去登录</Link>
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
  const [saving, setSaving] = useState(false)
  const [profileMessage, setProfileMessage] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [bio, setBio] = useState('')
  const [websiteURL, setWebsiteURL] = useState('')
  const [orders, setOrders] = useState<Page<MyOrder> | null>(null)
  const [ordersPage, setOrdersPage] = useState(1)
  const [ordersError, setOrdersError] = useState('')
  const [ordersBusy, setOrdersBusy] = useState<number | null>(null)
  const [authorApplications, setAuthorApplications] = useState<AuthorApplication[]>([])
  const [authorReason, setAuthorReason] = useState('')
  const [authorMessage, setAuthorMessage] = useState('')
  const [authorBusy, setAuthorBusy] = useState(false)
  const mounted = useRef(false)
  const draftUserID = useRef<number | null>(null)
  const draftDirty = useRef(false)
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])
  useEffect(() => {
    if (!user || (draftUserID.current === user.id && draftDirty.current)) return
    if (draftUserID.current !== user.id) draftDirty.current = false
    draftUserID.current = user.id
    setDisplayName(user.display_name || '')
    setEmail(user.email)
    setBio(user.bio || '')
    setWebsiteURL(user.website_url || '')
  }, [user])
  useEffect(() => {
    if (!token) return
    let active = true
    setLoading(true)
    api<AuthUser>('/auth/me', { headers: { Authorization: `Bearer ${token}` } })
      .then((freshUser) => {
        if (!active || readToken() !== token) return
        setUser(freshUser)
        refreshSessionUser(freshUser)
        setError('')
      })
      .catch((failure) => {
        if (!active || readToken() !== token) return
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
  useEffect(() => {
    if (!token || !user) return
    let active = true
    api<Page<MyOrder>>(`/orders?page=${ordersPage}`, { headers: { Authorization: `Bearer ${token}` } })
      .then((result) => { if (active) { setOrders(result); setOrdersError('') } })
      .catch((failure) => { if (active) setOrdersError(failure instanceof Error ? failure.message : '订单加载失败') })
    return () => { active = false }
  }, [token, user?.id, ordersPage])
  useEffect(() => {
    if (!token || !user) return
    let active = true
    api<AuthorApplication[]>('/author-applications/mine', { headers: { Authorization: `Bearer ${token}` } })
      .then((items) => { if (active) setAuthorApplications(items) })
      .catch(() => {})
    return () => { active = false }
  }, [token, user?.id, retry])
  if (!token && !error) return <Navigate to="/login" replace />
  function logout() {
    clearSession()
    navigate('/login', { replace: true })
  }
  async function saveProfile(event: FormEvent) {
    event.preventDefault()
    if (!token) return
    const requestToken = token
    setSaving(true)
    setProfileMessage('')
    try {
      const updated = await api<AuthUser>('/auth/me', {
        method: 'PUT',
        headers: { Authorization: `Bearer ${token}` },
        body: JSON.stringify({ display_name: displayName, email, bio, website_url: websiteURL }),
      })
      if (!mounted.current || readToken() !== requestToken) return
      draftDirty.current = false
      setUser(updated)
      refreshSessionUser(updated)
      setProfileMessage('资料已保存')
    } catch (failure) {
      if (!mounted.current || readToken() !== requestToken) return
      if (failure instanceof ApiError && failure.status === 401) {
        setSaving(false)
        clearSession()
        setUser(null)
        setError('登录已失效，请重新登录。')
        return
      }
      setProfileMessage(failure instanceof Error ? failure.message : '保存失败')
    } finally {
      if (mounted.current && readToken() === requestToken) setSaving(false)
    }
  }
  async function uploadAvatar(file?: File) {
    if (!file || !token) return
    if (file.size > 2 * 1024 * 1024 || !['image/png', 'image/jpeg'].includes(file.type)) {
      setProfileMessage('请选择不超过 2 MiB 的 PNG 或 JPEG 图片。')
      return
    }
    setSaving(true)
    setProfileMessage('')
    const requestToken = token
    try {
      const body = new FormData()
      body.append('file', file)
      const updated = await api<AuthUser>('/auth/me/avatar', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        body,
      })
      if (!mounted.current || readToken() !== requestToken) return
      setUser(updated)
      refreshSessionUser(updated)
      setProfileMessage('头像已更新')
    } catch (failure) {
      if (!mounted.current || readToken() !== requestToken) return
      if (failure instanceof ApiError && failure.status === 401) {
        setSaving(false)
        clearSession()
        setUser(null)
        setError('登录已失效，请重新登录。')
        return
      }
      setProfileMessage(failure instanceof Error ? failure.message : '头像上传失败')
    } finally {
      if (mounted.current && readToken() === requestToken) setSaving(false)
    }
  }
  async function deleteAvatar() {
    if (!token) return
    const requestToken = token
    setSaving(true)
    setProfileMessage('')
    try {
      const updated = await api<AuthUser>('/auth/me/avatar', {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${requestToken}` },
      })
      if (!mounted.current || readToken() !== requestToken) return
      setUser(updated)
      refreshSessionUser(updated)
      setProfileMessage('头像已移除')
    } catch (failure) {
      if (!mounted.current || readToken() !== requestToken) return
      if (failure instanceof ApiError && failure.status === 401) {
        setSaving(false)
        clearSession()
        setUser(null)
        setError('登录已失效，请重新登录。')
        return
      }
      setProfileMessage(failure instanceof Error ? failure.message : '头像移除失败')
    } finally {
      if (mounted.current && readToken() === requestToken) setSaving(false)
    }
  }
  async function cancelOrder(order: MyOrder) {
    if (!token) return
    setOrdersBusy(order.id)
    setOrdersError('')
    try {
      await api(`/orders/${order.id}/cancel`, { method: 'PATCH', headers: { Authorization: `Bearer ${token}` } })
      const latest = await api<Page<MyOrder>>(`/orders?page=${ordersPage}`, { headers: { Authorization: `Bearer ${token}` } })
      setOrders(latest)
    } catch (failure) {
      setOrdersError(failure instanceof Error ? failure.message : '取消订单失败')
    } finally {
      setOrdersBusy(null)
    }
  }
  async function applyAuthor(event: FormEvent) {
    event.preventDefault()
    if (!token || authorReason.trim().length < 10) { setAuthorMessage('请写至少 10 个字，介绍你想分享的内容。'); return }
    setAuthorBusy(true)
    setAuthorMessage('')
    try {
      await api('/author-applications', { method: 'POST', headers: { Authorization: `Bearer ${token}` }, body: JSON.stringify({ reason: authorReason.trim() }) })
      const items = await api<AuthorApplication[]>('/author-applications/mine', { headers: { Authorization: `Bearer ${token}` } })
      setAuthorApplications(items)
      setAuthorReason('')
      setAuthorMessage('申请已提交，审核结果会显示在这里。')
    } catch (failure) { setAuthorMessage(failure instanceof Error ? failure.message : '申请提交失败') }
    finally { setAuthorBusy(false) }
  }
  return (
    <div className="account-page container">
      <span className="auth-overline">账户 / 我的资料</span>
      <div className="account-card">
        <div className="account-cover" aria-hidden="true" />
        <div className="account-avatar">
          {user?.avatar_url ? (
            <img src={user.avatar_url} alt="我的头像" />
          ) : (
            <UserRound size={34} />
          )}
        </div>
        <div className="account-details">
          <h1>
            {user?.display_name || user?.username || (loading ? '正在读取账户…' : '账户暂不可用')}
          </h1>
          <p>{user?.email || (loading ? '正在确认登录状态' : error)}</p>
          {user?.bio && <p className="account-bio">{user.bio}</p>}
          {user && (
            <div className="account-meta">
              <span>{user.role === 1 ? '站点管理员' : user.role === 3 ? '本站作者' : '读者'}</span>
              <span>加入于 {formatDate(user.created_at)}</span>
              {user.ip_region && <span>IP 属地 · {user.ip_region}</span>}
            </div>
          )}
          {user?.website_url && <a className="account-website" href={user.website_url} target="_blank" rel="noopener noreferrer">我的网站 <ArrowRight size={15} /></a>}
        </div>
        {user && (
          <button type="button" className="account-logout" onClick={logout} disabled={saving}>
            <LogOut size={17} /> 退出登录
          </button>
        )}
      </div>
      {user && !loading && !error && (
        <div className="account-edit-layout">
          <form className="account-edit" onSubmit={saveProfile}>
            <div className="account-section-head">
              <span>01 / PROFILE</span>
              <h2>个人资料</h2>
              <p>这些内容会显示在你的账户页。邮箱用于登录，不会公开展示。</p>
            </div>
            <fieldset className="account-edit-grid" disabled={saving}>
              <label>
                <span>显示名称</span>
                <input
                  value={displayName}
                  onChange={(event) => {
                    draftDirty.current = true
                    setDisplayName(event.target.value)
                  }}
                  maxLength={60}
                  placeholder="怎么称呼你"
                />
              </label>
              <label>
                <span>邮箱</span>
                <input
                  type="email"
                  value={email}
                  onChange={(event) => {
                    draftDirty.current = true
                    setEmail(event.target.value)
                  }}
                  maxLength={255}
                  required
                />
              </label>
              <label className="wide">
                <span>个人网站</span>
                <input
                  type="url"
                  value={websiteURL}
                  onChange={(event) => {
                    draftDirty.current = true
                    setWebsiteURL(event.target.value)
                  }}
                  maxLength={255}
                  placeholder="https://"
                />
              </label>
              <label className="wide">
                <span>个人简介</span>
                <textarea
                  value={bio}
                  onChange={(event) => {
                    draftDirty.current = true
                    setBio(event.target.value)
                  }}
                  maxLength={500}
                  rows={4}
                  placeholder="写一点关于自己的介绍"
                />
              </label>
            </fieldset>
            <div className="account-edit-actions">
              <button type="submit" disabled={saving}>
                <Save size={16} /> {saving ? '保存中…' : '保存资料'}
              </button>
              <span role="status">{profileMessage}</span>
            </div>
          </form>
          <aside className="account-avatar-editor">
            <span>02 / AVATAR</span>
            <h2>我的头像</h2>
            <div className="account-avatar-preview">
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="当前头像" />
              ) : (
                <UserRound size={44} />
              )}
            </div>
            <p>上传 PNG 或 JPEG，文件不超过 2 MiB。建议使用方形图片。</p>
            <label className="account-avatar-upload">
              <UploadCloud size={16} /> 选择图片
              <input
                type="file"
                accept="image/png,image/jpeg"
                disabled={saving}
                onChange={(event) => {
                  void uploadAvatar(event.target.files?.[0])
                  event.target.value = ''
                }}
              />
            </label>
            {user.avatar_url && (
              <button
                className="account-avatar-remove"
                type="button"
                disabled={saving}
                onClick={() => void deleteAvatar()}
              >
                移除头像
              </button>
            )}
          </aside>
        </div>
      )}
      {user && !loading && !error && <section className="account-orders" aria-labelledby="account-orders-title">
        <div className="account-section-head">
          <span>03 / ORDERS</span>
          <h2 id="account-orders-title">我的订单</h2>
          <p>查看订单进度；待付款订单可自行取消，库存会立即退回。</p>
        </div>
        {ordersError && <p className="auth-error" role="alert">{ordersError}</p>}
        {!orders ? <p className="account-orders-empty">正在读取订单…</p> : orders.items.length === 0 ? <p className="account-orders-empty"><ShoppingBag size={19} /> 还没有订单，去小店看看吧。</p> : <div className="account-order-list">
          {orders.items.map((order) => <div className="account-order" key={order.id}>
            <div><strong>{order.product_name}</strong><small>订单 {order.order_no} · {formatDate(order.created_at)} · {order.quantity} 件</small></div>
            <span>{formatPrice(order.total_cents)}</span>
            <em>{({pending_payment:'待付款',paid:'已付款',delivered:'已交付',cancelled:'已取消'} as Record<string,string>)[order.status] || order.status}</em>
            {order.status === 'pending_payment' && <button type="button" disabled={ordersBusy === order.id} onClick={() => void cancelOrder(order)}>{ordersBusy === order.id ? '取消中…' : '取消订单'}</button>}
          </div>)}
        </div>}
        {orders && orders.total > 10 && <div className="account-order-pages"><button type="button" disabled={ordersPage <= 1} onClick={() => setOrdersPage((page) => page - 1)}>上一页</button><span>{ordersPage} / {Math.ceil(orders.total / 10)}</span><button type="button" disabled={ordersPage >= Math.ceil(orders.total / 10)} onClick={() => setOrdersPage((page) => page + 1)}>下一页</button></div>}
      </section>}
      {user && !loading && !error && <section className="account-author" aria-labelledby="account-author-title">
        <div className="account-section-head"><span>04 / WRITING</span><h2 id="account-author-title">我的创作</h2><p>通过审核后，你可以发布文章、管理自己的草稿和作品。</p></div>
        {user.role === 1 || user.role === 3 ? <Link className="account-author-enter" to="/studio">进入写作台 <ArrowRight size={17} /></Link> : <>
          {authorApplications[0] && <p className="account-author-status">最近申请：{({ pending: '等待审核', approved: '已通过，请刷新账户资料', rejected: '未通过，可再次申请' } as const)[authorApplications[0].status]} · {formatDate(authorApplications[0].created_at)}</p>}
          {!authorApplications.some((item) => item.status === 'pending') && <form onSubmit={applyAuthor}><label htmlFor="author-reason">想在这里分享什么？</label><textarea id="author-reason" rows={4} maxLength={1000} minLength={10} required value={authorReason} onChange={(event) => setAuthorReason(event.target.value)} placeholder="介绍你准备写的主题，以及申请成为作者的原因" /><button type="submit" disabled={authorBusy}>{authorBusy ? '提交中…' : '申请成为作者'} <ArrowRight size={16} /></button></form>}
          {authorMessage && <p role="status">{authorMessage}</p>}
        </>}
      </section>}
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
