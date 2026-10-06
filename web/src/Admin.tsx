import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import {
  Check,
  Edit3,
  ExternalLink,
  LogOut,
  Plus,
  RefreshCw,
  Trash2,
  UploadCloud,
} from 'lucide-react'
import { ApiError, api, formatDate, formatPrice, type Page } from './api'
import { clearSession, saveSession, type LoginResult } from './session'
import './admin.css'

type Kind = 'articles' | 'projects' | 'products' | 'links' | 'orders' | 'comments' | 'applications'
type EditableKind = Exclude<Kind, 'orders' | 'comments' | 'applications'>
function isEditableKind(kind: Kind): kind is EditableKind {
  return kind === 'articles' || kind === 'projects' || kind === 'products' || kind === 'links'
}
type Item = Record<string, unknown> & { id: number }
type Field = {
  key: string
  label: string
  type?: 'text' | 'textarea' | 'number' | 'url' | 'checkbox'
}

const tabs: { kind: Kind; label: string }[] = [
  { kind: 'articles', label: '文章' },
  { kind: 'projects', label: '项目' },
  { kind: 'products', label: '商品' },
  { kind: 'links', label: '友链' },
  { kind: 'orders', label: '订单' },
  { kind: 'comments', label: '评论审核' },
  { kind: 'applications', label: '友链申请' },
]
const fields: Record<EditableKind, Field[]> = {
  articles: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'title', label: '标题' },
    { key: 'excerpt', label: '摘要', type: 'textarea' },
    { key: 'body_md', label: '正文（Markdown）', type: 'textarea' },
    { key: 'cover_url', label: '封面图片地址', type: 'url' },
    { key: 'category', label: '栏目' },
    { key: 'tags', label: '标签（用逗号分隔）' },
    { key: 'status', label: '状态' },
  ],
  projects: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'title', label: '项目名称' },
    { key: 'description', label: '项目介绍', type: 'textarea' },
    { key: 'cover_url', label: '封面图片地址', type: 'url' },
    { key: 'tags', label: '技术标签（用逗号分隔）' },
    { key: 'preview_url', label: '预览地址', type: 'url' },
    { key: 'source_url', label: '源码地址', type: 'url' },
    { key: 'status', label: '状态' },
  ],
  products: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'name', label: '商品名称' },
    { key: 'description', label: '商品介绍', type: 'textarea' },
    { key: 'cover_url', label: '封面图片地址', type: 'url' },
    { key: 'price_cents', label: '价格（分）', type: 'number' },
    { key: 'stock', label: '库存', type: 'number' },
    { key: 'status', label: '状态' },
  ],
  links: [
    { key: 'name', label: '站点名称' },
    { key: 'url', label: '站点地址', type: 'url' },
    { key: 'avatar_url', label: '头像地址', type: 'url' },
    { key: 'description', label: '简介', type: 'textarea' },
    { key: 'sort_order', label: '排序', type: 'number' },
    { key: 'enabled', label: '公开显示', type: 'checkbox' },
  ],
}
const blank: Record<EditableKind, Record<string, unknown>> = {
  articles: {
    slug: '',
    title: '',
    excerpt: '',
    body_md: '',
    cover_url: '',
    category: 'tech',
    tags: '',
    status: 'draft',
  },
  projects: {
    slug: '',
    title: '',
    description: '',
    cover_url: '',
    tags: '',
    preview_url: '',
    source_url: '',
    status: 'draft',
  },
  products: {
    slug: '',
    name: '',
    description: '',
    cover_url: '',
    price_cents: 0,
    stock: 0,
    status: 'draft',
  },
  links: { name: '', url: '', avatar_url: '', description: '', sort_order: 0, enabled: true },
}

function normalize(kind: EditableKind, item: Item): Record<string, unknown> {
  const result: Record<string, unknown> = { ...item }
  if (kind === 'articles' || kind === 'projects')
    result.tags = Array.isArray(item.tags) ? item.tags.join(', ') : ''
  return result
}
function toPayload(kind: EditableKind, form: Record<string, unknown>): Record<string, unknown> {
  const result = { ...form }
  delete result.id
  if (kind === 'articles' || kind === 'projects')
    result.tags = String(result.tags || '')
      .split(',')
      .map((v) => v.trim())
      .filter(Boolean)
  if (kind === 'products') {
    result.price_cents = Number(result.price_cents)
    result.stock = Number(result.stock)
  }
  if (kind === 'links') result.sort_order = Number(result.sort_order)
  return result
}

export default function Admin() {
  const [token, setToken] = useState(() => sessionStorage.getItem('devhub_admin_token') || '')
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [kind, setKind] = useState<Kind>('articles')
  const [items, setItems] = useState<Item[]>([])
  const [form, setForm] = useState<Record<string, unknown> | null>(null)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [authorized, setAuthorized] = useState(false)
  const [authRetry, setAuthRetry] = useState(0)
  const requestSequence = useRef(0)
  const currentKind = useRef<Kind>('articles')

  const request = useCallback(
    <T,>(path: string, options?: RequestInit) =>
      api<T>(path, {
        ...options,
        headers: { Authorization: `Bearer ${token}`, ...options?.headers },
      }),
    [token],
  )
  const refresh = useCallback(async () => {
    if (!token || !authorized) return
    const sequence = ++requestSequence.current
    try {
      const path = kind === 'applications' ? 'link-applications' : kind
      const data = await request<Page<Item> | Item[]>(`/admin/${path}?page_size=50`)
      if (sequence === requestSequence.current && kind === currentKind.current)
        setItems(Array.isArray(data) ? data : data.items)
    } catch (error) {
      if (sequence === requestSequence.current && kind === currentKind.current)
        setMessage(error instanceof Error ? error.message : '列表加载失败')
    }
  }, [token, authorized, kind, request])

  useEffect(() => {
    if (!token) {
      setAuthorized(false)
      return
    }
    request<{ role: number }>('/auth/me')
      .then((user) => {
        if (user.role !== 1) throw new ApiError('当前账号没有管理员权限', 403)
        setAuthorized(true)
        setMessage('')
      })
      .catch((error) => {
        setAuthorized(false)
        if (error instanceof ApiError && error.status === 401) {
          clearSession()
          setToken('')
        } else if (error instanceof ApiError && error.status === 403) {
          sessionStorage.removeItem('devhub_admin_token')
          setToken('')
        }
        setMessage(error instanceof Error ? error.message : '暂时无法验证登录状态')
      })
  }, [token, request, authRetry])
  useEffect(() => {
    void refresh()
  }, [refresh])

  async function login(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setMessage('')
    try {
      const result = await api<LoginResult>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ identifier, password }),
      })
      if (result.user.role !== 1) throw new Error('当前账号没有管理员权限')
      saveSession(result)
      setToken(result.access_token)
      setPassword('')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '登录失败')
    } finally {
      setBusy(false)
    }
  }
  function logout() {
    requestSequence.current++
    clearSession()
    setToken('')
    setItems([])
    setForm(null)
    setAuthorized(false)
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!form || !isEditableKind(kind)) return
    setBusy(true)
    setMessage('')
    try {
      const id = form.id as number | undefined
      await request(`/admin/${kind}${id ? `/${id}` : ''}`, {
        method: id ? 'PUT' : 'POST',
        body: JSON.stringify(toPayload(kind, form)),
      })
      setMessage(id ? '已保存修改' : '已创建')
      setForm(null)
      await refresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '保存失败')
    } finally {
      setBusy(false)
    }
  }
  async function remove(item: Item) {
    if (!window.confirm('确定删除这条内容吗？')) return
    try {
      await request(`/admin/${kind}/${item.id}`, { method: 'DELETE' })
      setMessage('已删除')
      await refresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '删除失败')
    }
  }
  async function changeOrder(item: Item, status: string) {
    const action =
      status === 'cancelled'
        ? '取消订单并恢复库存'
        : status === 'paid'
          ? '标记为已付款'
          : '标记为已交付'
    if (!window.confirm(`确定${action}吗？`)) return
    try {
      await request(`/admin/orders/${item.id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      setMessage('订单状态已更新')
      await refresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '更新失败')
    }
  }
  async function review(item: Item, status: 'approved' | 'rejected') {
    setBusy(true)
    setMessage('')
    try {
      const path = kind === 'applications' ? 'link-applications' : 'comments'
      await request(`/admin/${path}/${item.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      setMessage(status === 'approved' ? '已通过审核' : '已驳回')
      await refresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '审核失败')
    } finally {
      setBusy(false)
    }
  }
  async function uploadBundle(item: Item, file?: File) {
    if (!file) return
    if (file.size > 20 * 1024 * 1024) {
      setMessage('ZIP 文件不能超过 20 MiB')
      return
    }
    setBusy(true)
    setMessage('')
    try {
      const body = new FormData()
      body.append('file', file)
      await request(`/admin/projects/${item.id}/bundle`, { method: 'POST', body })
      setMessage(item.status === 'published' ? '预览已更新' : '预览已上传，发布项目后即可公开访问')
      await refresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '上传失败')
    } finally {
      setBusy(false)
    }
  }

  if (!authorized)
    return (
      <div className="admin-login-wrap">
        <form className="admin-login" onSubmit={login}>
          <span className="admin-kicker">站点管理</span>
          <h1>管理员登录</h1>
          <p>登录后可以发布文章、管理项目和查看订单。</p>
          <label htmlFor="admin-identifier">用户名或邮箱</label>
          <input
            id="admin-identifier"
            required
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            autoComplete="username"
          />
          <label htmlFor="admin-password">密码</label>
          <input
            id="admin-password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
          <button type="submit" disabled={busy}>
            {busy ? '登录中…' : '登录'}
          </button>
          {token && (
            <button type="button" onClick={() => setAuthRetry((value) => value + 1)}>
              重试验证当前登录
            </button>
          )}
          {message && (
            <span className="admin-message" role="status">
              {message}
            </span>
          )}
        </form>
      </div>
    )

  return (
    <div className="container admin-page">
      <div className="admin-top">
        <div>
          <span className="admin-kicker">站点管理</span>
          <h1>内容工作台</h1>
        </div>
        <button type="button" onClick={logout}>
          <LogOut size={16} /> 退出登录
        </button>
      </div>
      <div className="admin-tabs" role="tablist" aria-label="管理模块">
        {tabs.map((tab) => (
          <button
            type="button"
            role="tab"
            aria-selected={kind === tab.kind}
            className={kind === tab.kind ? 'active' : ''}
            key={tab.kind}
            onClick={() => {
              if (kind === tab.kind) return
              currentKind.current = tab.kind
              requestSequence.current++
              setKind(tab.kind)
              setItems([])
              setForm(null)
              setMessage('')
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="admin-toolbar">
        <span>
          {tabs.find((tab) => tab.kind === kind)?.label} · {items.length} 条
        </span>
        <div>
          <button type="button" onClick={() => void refresh()}>
            <RefreshCw size={16} /> 刷新
          </button>
          {isEditableKind(kind) && (
            <button type="button" className="admin-add" onClick={() => setForm({ ...blank[kind] })}>
              <Plus size={16} /> 新建
            </button>
          )}
        </div>
      </div>
      {message && (
        <p className="admin-message" role="status">
          {message}
        </p>
      )}
      {form && isEditableKind(kind) && (
        <form className="admin-editor" onSubmit={save}>
          <div className="admin-editor-head">
            <h2>
              {form.id ? '编辑' : '新建'}
              {tabs.find((tab) => tab.kind === kind)?.label}
            </h2>
            <button type="button" onClick={() => setForm(null)}>
              取消
            </button>
          </div>
          <div className="admin-fields">
            {fields[kind].map((field) => (
              <label key={field.key} className={field.type === 'textarea' ? 'wide' : ''}>
                <span>{field.label}</span>
                {field.key === 'category' ? (
                  <select
                    value={String(form.category || 'tech')}
                    onChange={(e) => setForm({ ...form, category: e.target.value })}
                  >
                    <option value="tech">技术</option>
                    <option value="travel">游记</option>
                    <option value="essay">随笔</option>
                    <option value="record">记录</option>
                  </select>
                ) : field.key === 'status' ? (
                  <select
                    value={String(form.status || 'draft')}
                    onChange={(e) => setForm({ ...form, status: e.target.value })}
                  >
                    <option value="draft">草稿</option>
                    <option value="published">公开</option>
                  </select>
                ) : field.type === 'checkbox' ? (
                  <input
                    type="checkbox"
                    checked={Boolean(form[field.key])}
                    onChange={(e) => setForm({ ...form, [field.key]: e.target.checked })}
                  />
                ) : field.type === 'textarea' ? (
                  <textarea
                    rows={field.key === 'body_md' ? 12 : 3}
                    value={String(form[field.key] ?? '')}
                    onChange={(e) => setForm({ ...form, [field.key]: e.target.value })}
                  />
                ) : (
                  <input
                    type={field.type || 'text'}
                    value={String(form[field.key] ?? '')}
                    onChange={(e) => setForm({ ...form, [field.key]: e.target.value })}
                  />
                )}
              </label>
            ))}
          </div>
          <button className="admin-save" type="submit" disabled={busy}>
            <Check size={17} /> {busy ? '保存中…' : '保存'}
          </button>
        </form>
      )}
      <div className="admin-list">
        {items.length === 0 ? (
          <div className="admin-empty">暂无记录</div>
        ) : (
          items.map((item) => (
            <div className="admin-row" key={item.id}>
              <div>
                <strong>
                  {String(
                    item.title ||
                      item.name ||
                      item.article_title ||
                      item.username ||
                      item.order_no ||
                      `#${item.id}`,
                  )}
                </strong>
                <span>
                  {kind === 'comments'
                    ? `${String(item.username)} · ${String(item.body)}`
                    : kind === 'applications'
                      ? `${String(item.username)} · ${String(item.url)}`
                      : kind === 'orders'
                        ? `${formatPrice(Number(item.total_cents || 0))} · ${String(item.email)} · ${String(item.status)}`
                        : `${String(item.slug || item.url || '')} · ${kind === 'links' ? (item.enabled ? '公开' : '隐藏') : item.status === 'published' ? '公开' : '草稿'}${kind === 'projects' && item.runtime_status === 'running' ? ' · 前后端运行中' : ''}`}
                  {kind === 'orders' && item.created_at
                    ? ` · ${formatDate(String(item.created_at))}`
                    : ''}
                </span>
                {kind === 'applications' && (
                  <span>
                    介绍：{String(item.description || '未填写')} · 头像：
                    {String(item.avatar_url || '未填写')}
                  </span>
                )}
              </div>
              <div className="admin-row-actions">
                {kind === 'comments' || kind === 'applications' ? (
                  <>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void review(item, 'approved')}
                    >
                      通过
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void review(item, 'rejected')}
                    >
                      驳回
                    </button>
                  </>
                ) : kind === 'orders' ? (
                  <>
                    {item.status === 'pending_payment' && (
                      <>
                        <button type="button" onClick={() => void changeOrder(item, 'paid')}>
                          已付款
                        </button>
                        <button type="button" onClick={() => void changeOrder(item, 'cancelled')}>
                          取消
                        </button>
                      </>
                    )}
                    {item.status === 'paid' && (
                      <button type="button" onClick={() => void changeOrder(item, 'delivered')}>
                        已交付
                      </button>
                    )}
                  </>
                ) : isEditableKind(kind) ? (
                  <>
                    {kind === 'projects' && (
                      <>
                        {item.preview_url && item.status === 'published' && (
                          <a
                            href={String(item.preview_url)}
                            target="_blank"
                            rel="noopener noreferrer"
                            aria-label={`预览 ${String(item.title)}`}
                          >
                            <ExternalLink size={17} />
                          </a>
                        )}
                        <label className="admin-upload" title="上传静态站点 ZIP">
                          <UploadCloud size={17} /> 上传预览
                          <input
                            type="file"
                            accept=".zip,application/zip"
                            disabled={busy}
                            onChange={(event) => {
                              void uploadBundle(item, event.target.files?.[0])
                              event.target.value = ''
                            }}
                          />
                        </label>
                      </>
                    )}
                    <button
                      type="button"
                      onClick={() => setForm(normalize(kind, item))}
                      aria-label={`编辑 ${String(item.title || item.name)}`}
                    >
                      <Edit3 size={17} />
                    </button>
                    <button
                      type="button"
                      onClick={() => void remove(item)}
                      aria-label={`删除 ${String(item.title || item.name)}`}
                    >
                      <Trash2 size={17} />
                    </button>
                  </>
                ) : null}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
