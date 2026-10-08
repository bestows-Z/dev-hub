import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Github, LoaderCircle, ShieldCheck, Unlink } from 'lucide-react'
import { toast } from 'sonner'
import { api } from './api'
import { readToken, saveSession, type LoginResult } from './session'
import { useConfirm } from './Feedback'
import './social-auth.css'

type Provider = { id: 'github' | 'google'; enabled: boolean }
type Identity = { provider: string; email: string; created_at: string }
const names = { github: 'GitHub', google: 'Google' }

function ProviderIcon({ id }: { id: string }) {
  if (id === 'github') return <Github size={18} />
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#4285F4"
        d="M21.6 12.23c0-.71-.06-1.39-.18-2.05H12v3.88h5.38a4.61 4.61 0 0 1-2 3.03v2.52h3.24c1.89-1.74 2.98-4.3 2.98-7.38Z"
      />
      <path
        fill="#34A853"
        d="M12 22c2.7 0 4.96-.9 6.62-2.43l-3.24-2.52c-.9.6-2.06.97-3.38.97-2.6 0-4.8-1.76-5.59-4.12H3.07v2.6A10 10 0 0 0 12 22Z"
      />
      <path fill="#FBBC05" d="M6.41 13.9a6 6 0 0 1 0-3.8V7.5H3.07a10 10 0 0 0 0 9l3.34-2.6Z" />
      <path
        fill="#EA4335"
        d="M12 5.98c1.47 0 2.79.5 3.83 1.51l2.87-2.87A9.62 9.62 0 0 0 12 2a10 10 0 0 0-8.93 5.5l3.34 2.6C7.2 7.74 9.4 5.98 12 5.98Z"
      />
    </svg>
  )
}

export function SocialLogin({ next = '/account' }: { next?: string }) {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  useEffect(() => {
    let active = true
    api<Provider[]>('/auth/oauth/providers')
      .then((data) => {
        if (active) setProviders(data)
      })
      .catch(() => {})
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])
  return (
    <div className="social-login">
      <div className="social-divider">
        <span>也可以使用</span>
      </div>
      <div className="social-buttons">
        {(['github', 'google'] as const).map((id) => {
          const enabled = providers.some((item) => item.id === id && item.enabled)
          return (
            <button
              key={id}
              type="button"
              disabled={loading || !enabled}
              title={enabled ? `使用 ${names[id]} 登录` : '站长暂未开放此登录方式'}
              onClick={() => {
                const destination =
                  next === 'admin'
                    ? '/admin'
                    : next.startsWith('/') && !next.startsWith('//') && !next.includes('\\')
                      ? next
                      : '/account'
                sessionStorage.setItem('devhub_oauth_next', destination)
                window.location.assign(`/api/v1/auth/oauth/${id}/start`)
              }}
            >
              <ProviderIcon id={id} />
              {names[id]}
            </button>
          )
        })}
      </div>
      {!loading && !providers.some((item) => item.enabled) && (
        <p className="social-unavailable">第三方登录暂未开放</p>
      )}
    </div>
  )
}

const callbackErrors: Record<string, string> = {
  unavailable: '第三方登录暂时不可用，请使用邮箱登录。',
  expired: '登录授权已过期，请重新发起登录。',
  cancelled: '你已取消授权，可以重新选择登录方式。',
  email: '第三方账号需要一个已验证的邮箱，请先到对应平台验证。',
  provider: '第三方平台暂时无法完成登录，请稍后再试。',
  account: '当前账号不可用，请联系站长。',
  link_required: '这个邮箱已有本站账号。请先登录，再到个人中心绑定第三方账号。',
  identity_in_use: '这个第三方账号已经绑定，请先解除原绑定。',
}

export function OAuthCallback() {
  const [params] = useSearchParams()
  const [initial] = useState(() => ({
    ticket: params.get('ticket'),
    linked: params.get('linked') === '1',
    error: params.get('error'),
  }))
  const [error, setError] = useState(
    initial.error ? callbackErrors[initial.error] || '登录未完成，请重试。' : '',
  )
  const exchange = useRef<Promise<LoginResult> | null>(null)
  const handled = useRef(false)
  const navigate = useNavigate()
  useEffect(() => {
    // Remove the temporary ticket before the user follows any page links.
    window.history.replaceState(window.history.state, '', window.location.pathname)
    if (initial.error) return
    if (initial.linked) {
      if (!handled.current) {
        handled.current = true
        toast.success('第三方账号已绑定')
        navigate('/account', { replace: true })
      }
      return
    }
    if (!initial.ticket) {
      setError('缺少登录凭证，请重新发起登录。')
      return
    }
    let active = true
    exchange.current ??= api<LoginResult>('/auth/oauth/exchange', {
      method: 'POST',
      body: JSON.stringify({ ticket: initial.ticket }),
    })
    exchange.current
      .then((result) => {
        if (!active || handled.current) return
        handled.current = true
        saveSession(result)
        const next = sessionStorage.getItem('devhub_oauth_next') || '/account'
        sessionStorage.removeItem('devhub_oauth_next')
        const safe =
          next.startsWith('/') &&
          !next.startsWith('//') &&
          !next.includes('\\') &&
          (!next.startsWith('/admin') || result.user.role === 1)
        toast.success('登录成功')
        navigate(safe ? next : '/account', { replace: true })
      })
      .catch((failure) => {
        if (active) setError(failure instanceof Error ? failure.message : '登录未完成，请重试。')
      })
    return () => {
      active = false
    }
  }, [initial, navigate])
  return (
    <section className="oauth-callback container">
      <div className="oauth-callback-card">
        <ShieldCheck size={36} />
        <h1>{error ? '登录还差一步' : '正在完成登录'}</h1>
        {error ? (
          <>
            <p role="alert">{error}</p>
            <div>
              <Link className="button button-primary" to="/login">
                返回登录
              </Link>
              {initial.error === 'link_required' && (
                <Link className="button" to="/account">
                  前往个人中心
                </Link>
              )}
            </div>
          </>
        ) : (
          <p className="oauth-working">
            <LoaderCircle size={18} />
            正在验证授权，请稍候。
          </p>
        )}
      </div>
    </section>
  )
}

export function ConnectedAccounts() {
  const confirm = useConfirm()
  const token = readToken()
  const [providers, setProviders] = useState<Provider[]>([])
  const [identities, setIdentities] = useState<Identity[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [retry, setRetry] = useState(0)
  useEffect(() => {
    let active = true
    Promise.all([
      api<Provider[]>('/auth/oauth/providers'),
      api<Identity[]>('/auth/me/identities', { headers: { Authorization: `Bearer ${token}` } }),
    ])
      .then(([available, linked]) => {
        if (active) {
          setProviders(available)
          setIdentities(linked)
          setError('')
        }
      })
      .catch((failure) => {
        if (active) setError(failure instanceof Error ? failure.message : '绑定信息暂时无法读取')
      })
    return () => {
      active = false
    }
  }, [token, retry])
  async function link(provider: string) {
    setBusy(true)
    try {
      const result = await api<{ url: string }>(`/auth/oauth/${provider}/link`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
      window.location.assign(result.url)
    } catch (failure) {
      toast.error(failure instanceof Error ? failure.message : '暂时无法绑定')
      setBusy(false)
    }
  }
  async function unlink(provider: string) {
    if (
      !(await confirm({
        title: '解除账号绑定？',
        description: '解除后，这个第三方账号将无法登录当前账号。你仍可使用邮箱验证码登录。',
        confirmLabel: '解除绑定',
        danger: true,
      }))
    )
      return
    setBusy(true)
    try {
      await api(`/auth/oauth/${provider}/link`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      setIdentities((items) => items.filter((item) => item.provider !== provider))
      toast.success('账号绑定已解除')
    } catch (failure) {
      toast.error(failure instanceof Error ? failure.message : '解除失败')
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="account-connections">
      <div className="account-section-head">
        <span>ACCOUNT / CONNECTIONS</span>
        <h2>第三方账号</h2>
        <p>绑定后可用对应平台登录，邮箱仍用于本站通知和验证码。</p>
      </div>
      {error ? (
        <p role="alert">
          {error}{' '}
          <button type="button" onClick={() => setRetry((n) => n + 1)}>
            重试
          </button>
        </p>
      ) : (
        <div className="account-connection-list">
          {(['github', 'google'] as const).map((id) => {
            const identity = identities.find((item) => item.provider === id)
            const enabled = providers.some((item) => item.id === id && item.enabled)
            return (
              <div className="account-connection" key={id}>
                <div className="account-connection-icon">
                  <ProviderIcon id={id} />
                </div>
                <div>
                  <strong>{names[id]}</strong>
                  <small>{identity ? identity.email : enabled ? '尚未绑定' : '暂未开放'}</small>
                </div>
                <button
                  disabled={busy || (!identity && !enabled)}
                  type="button"
                  onClick={() => void (identity ? unlink(id) : link(id))}
                >
                  {identity ? (
                    <>
                      <Unlink size={14} />
                      解除绑定
                    </>
                  ) : (
                    '绑定账号'
                  )}
                </button>
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}
