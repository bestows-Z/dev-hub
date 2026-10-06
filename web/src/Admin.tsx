import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import {
  ArrowRight,
  BarChart3,
  Check,
  ChevronLeft,
  ChevronRight,
  Edit3,
  ExternalLink,
  FileText,
  FolderKanban,
  Inbox,
  Images,
  LayoutDashboard,
  Link2,
  LogOut,
  MessageCircle,
  Plus,
  Play,
  RefreshCw,
  Search,
  ShoppingBag,
  Square,
  Trash2,
  UploadCloud,
  type LucideIcon,
} from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'
import remarkGfm from 'remark-gfm'
import { ApiError, api, formatDate, formatPrice, type Page } from './api'
import { clearSession, readUser, saveSession, type LoginResult } from './session'
import './admin.css'

type Kind =
  | 'overview'
  | 'articles'
  | 'projects'
  | 'gallery'
  | 'products'
  | 'links'
  | 'orders'
  | 'comments'
  | 'applications'
  | 'author-applications'
type EditableKind = Exclude<Kind, 'overview' | 'orders' | 'comments' | 'applications' | 'author-applications'>
function isEditableKind(kind: Kind): kind is EditableKind {
  return kind === 'articles' || kind === 'projects' || kind === 'gallery' || kind === 'products' || kind === 'links'
}
type Item = Record<string, unknown> & { id: number }
type AnalyticsSummary = {
  today: number
  last_7_days: number
  daily: { day: string; count: number }[]
  top_pages: { kind: string; slug: string; count: number }[]
}
type Field = {
  key: string
  label: string
  type?: 'text' | 'textarea' | 'number' | 'url' | 'checkbox'
}

const tabs: {
  kind: Kind
  label: string
  icon: LucideIcon
  group: 'workspace' | 'content' | 'review'
}[] = [
  { kind: 'overview', label: '总览', icon: LayoutDashboard, group: 'workspace' },
  { kind: 'articles', label: '文章管理', icon: FileText, group: 'content' },
  { kind: 'projects', label: '项目管理', icon: FolderKanban, group: 'content' },
  { kind: 'gallery', label: '相册管理', icon: Images, group: 'content' },
  { kind: 'products', label: '商品管理', icon: ShoppingBag, group: 'content' },
  { kind: 'links', label: '友链管理', icon: Link2, group: 'content' },
  { kind: 'orders', label: '订单管理', icon: BarChart3, group: 'content' },
  { kind: 'comments', label: '评论审核', icon: MessageCircle, group: 'review' },
  { kind: 'applications', label: '友链申请', icon: Inbox, group: 'review' },
  { kind: 'author-applications', label: '作者申请', icon: Edit3, group: 'review' },
]
type ManagedKind = Exclude<Kind, 'overview'>
const managedKinds: ManagedKind[] = [
  'articles',
  'projects',
  'gallery',
  'products',
  'links',
  'orders',
  'comments',
  'applications',
  'author-applications',
]
const pageSize = 20
function listKey(kind: Kind, page: number, reviewStatus: string, search: string, token: string) {
  return `${kind}:${page}:${reviewStatus}:${search}:${token}`
}
const fields: Record<EditableKind, Field[]> = {
  articles: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'title', label: '标题' },
    { key: 'excerpt', label: '摘要', type: 'textarea' },
    { key: 'body_md', label: '正文（Markdown）', type: 'textarea' },
    { key: 'category', label: '栏目' },
    { key: 'tags', label: '标签（用逗号分隔）' },
    { key: 'status', label: '状态' },
  ],
  projects: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'title', label: '项目名称' },
    { key: 'description', label: '项目介绍', type: 'textarea' },
    { key: 'tags', label: '技术标签（用逗号分隔）' },
    { key: 'preview_url', label: '预览地址', type: 'url' },
    { key: 'source_url', label: '源码地址', type: 'url' },
    { key: 'status', label: '状态' },
  ],
  gallery: [
    { key: 'title', label: '照片标题' },
    { key: 'description', label: '照片说明', type: 'textarea' },
    { key: 'location', label: '拍摄地点' },
    { key: 'taken_at', label: '拍摄日期' },
    { key: 'status', label: '状态' },
  ],
  products: [
    { key: 'slug', label: '网址标识（英文和短横线）' },
    { key: 'name', label: '商品名称' },
    { key: 'description', label: '商品介绍', type: 'textarea' },
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
  gallery: { title: '', description: '', location: '', taken_at: '', status: 'draft' },
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
  if (kind === 'gallery') result.taken_at = item.taken_at ? String(item.taken_at).slice(0, 10) : ''
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
  const [kind, setKind] = useState<Kind>('overview')
  const [items, setItems] = useState<Item[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [reviewStatus, setReviewStatus] = useState('pending')
  const [stats, setStats] = useState<Partial<Record<ManagedKind, number>>>({})
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null)
  const [previewMarkdown, setPreviewMarkdown] = useState(false)
  const [form, setForm] = useState<Record<string, unknown> | null>(null)
  const [galleryFile, setGalleryFile] = useState<File | null>(null)
  const [coverFile, setCoverFile] = useState<File | null>(null)
  const [coverPreview, setCoverPreview] = useState('')
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [authorized, setAuthorized] = useState(false)
  const [authRetry, setAuthRetry] = useState(0)
  const requestSequence = useRef(0)
  useEffect(() => () => { if (coverPreview) URL.revokeObjectURL(coverPreview) }, [coverPreview])
  const currentListKey = useRef(listKey(kind, page, reviewStatus, search, token))
  currentListKey.current = listKey(kind, page, reviewStatus, search, token)

  const request = useCallback(
    <T,>(path: string, options?: RequestInit) =>
      api<T>(path, {
        ...options,
        headers: { Authorization: `Bearer ${token}`, ...options?.headers },
      }),
    [token],
  )
  const refresh = useCallback(async () => {
    if (!token || !authorized || kind === 'overview') return
    const queryKey = listKey(kind, page, reviewStatus, search, token)
    if (queryKey !== currentListKey.current) return
    const sequence = ++requestSequence.current
    try {
      const path = kind === 'applications' ? 'link-applications' : kind
	  const statusQuery =
		kind === 'comments' || kind === 'applications' || kind === 'author-applications' ? `&status=${reviewStatus}` : ''
      const data = await request<Page<Item> | Item[]>(
        `/admin/${path}?page_size=${pageSize}&page=${page}${statusQuery}&q=${encodeURIComponent(search)}`,
      )
      if (sequence === requestSequence.current && queryKey === currentListKey.current) {
        const nextTotal = Array.isArray(data) ? data.length : data.total
        const lastPage = Math.max(1, Math.ceil(nextTotal / pageSize))
        if (page > lastPage) {
          currentListKey.current = listKey(kind, lastPage, reviewStatus, search, token)
          requestSequence.current++
          setItems([])
          setTotal(nextTotal)
          setPage(lastPage)
          return
        }
        setItems(
          Array.isArray(data) ? data.slice((page - 1) * pageSize, page * pageSize) : data.items,
        )
        setTotal(nextTotal)
      }
    } catch (error) {
      if (sequence === requestSequence.current && queryKey === currentListKey.current)
        setMessage(error instanceof Error ? error.message : '列表加载失败')
    }
  }, [token, authorized, kind, page, reviewStatus, search, request])
  const refreshLatest = useRef(refresh)
  refreshLatest.current = refresh

  const loadStats = useCallback(async () => {
    if (!token || !authorized) return
    void request<AnalyticsSummary>('/admin/analytics').then(setAnalytics).catch(() => setAnalytics(null))
    const results = await Promise.allSettled(
      managedKinds.map((entry) => {
		const path = entry === 'applications' ? 'link-applications' : entry
        return request<Page<Item> | Item[]>(`/admin/${path}?page_size=1`)
      }),
    )
    setStats((previous) => {
      const next = { ...previous }
      results.forEach((result, index) => {
        if (result.status === 'fulfilled')
          next[managedKinds[index]] = Array.isArray(result.value)
            ? result.value.length
            : result.value.total
      })
      return next
    })
  }, [token, authorized, request])

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
  useEffect(() => {
    if (kind !== 'projects' || !items.some((item) => item.runtime_job_status === 'queued' || item.runtime_job_status === 'running')) return
    const interval = window.setInterval(() => {
      if (document.visibilityState === 'visible') void refreshLatest.current()
    }, 3000)
    return () => window.clearInterval(interval)
  }, [kind, items])
  useEffect(() => {
    if (kind === 'overview') void loadStats()
  }, [kind, loadStats])

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
    currentListKey.current = ''
    clearSession()
    setToken('')
    setItems([])
    setForm(null)
    setGalleryFile(null)
    setCoverFile(null)
    setCoverPreview('')
    setAuthorized(false)
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!form || !isEditableKind(kind)) return
    setBusy(true)
    setMessage('')
    let uploadedCoverID = ''
    try {
      const id = form.id as number | undefined
      if (kind === 'gallery') {
        if (!id && !galleryFile) throw new Error('先选择一张照片')
        if (galleryFile && galleryFile.size > 10 * 1024 * 1024) throw new Error('照片不能超过 10 MiB')
        const payload = toPayload(kind, form)
        let savedID = id
        if (!savedID) {
          const created = await request<Item>('/admin/gallery', { method: 'POST', body: JSON.stringify({ ...payload, status: 'draft' }) })
          savedID = created.id
        }
        if (galleryFile) {
          const body = new FormData()
          body.append('file', galleryFile)
          await request(`/admin/gallery/${savedID}/image`, { method: 'POST', body })
        }
        await request(`/admin/gallery/${savedID}`, { method: 'PUT', body: JSON.stringify(payload) })
        setGalleryFile(null)
      } else {
        const payload = toPayload(kind, form)
        if (coverFile && (kind === 'articles' || kind === 'projects' || kind === 'products')) {
          const body = new FormData()
          body.append('file', coverFile)
          const uploaded = await request<{ id: string; url: string }>('/admin/media', { method: 'POST', body })
          uploadedCoverID = uploaded.id
          payload.cover_url = uploaded.url
        }
        await request(`/admin/${kind}${id ? `/${id}` : ''}`, {
          method: id ? 'PUT' : 'POST',
          body: JSON.stringify(payload),
        })
      }
      setMessage(id ? '已保存修改' : '已创建')
      setForm(null)
      setCoverFile(null)
      setCoverPreview('')
      await refreshLatest.current()
    } catch (error) {
      if (uploadedCoverID) void request(`/admin/media/${uploadedCoverID}`, { method: 'DELETE' }).catch(() => {})
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
      await refreshLatest.current()
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
      await refreshLatest.current()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '更新失败')
    }
  }
  async function review(item: Item, status: 'approved' | 'rejected') {
    setBusy(true)
    setMessage('')
    try {
      const path = kind === 'applications' ? 'link-applications' : kind === 'author-applications' ? 'author-applications' : 'comments'
      await request(`/admin/${path}/${item.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      })
      setMessage(status === 'approved' ? '已通过审核' : '已驳回')
      await refreshLatest.current()
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
      await refreshLatest.current()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '上传失败')
    } finally {
      setBusy(false)
    }
  }

  async function uploadRuntimeBundle(item: Item, file?: File) {
    if (!file) return
    if (file.size > 50 * 1024 * 1024) {
      setMessage('完整项目 ZIP 不能超过 50 MiB')
      return
    }
    setBusy(true)
    setMessage('')
    try {
      const body = new FormData()
      body.append('file', file)
      await request(`/admin/projects/${item.id}/runtime-bundle`, { method: 'POST', body })
      setMessage('完整项目已上传。现在可以在项目列表中点击「启动」。')
      await refreshLatest.current()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '上传失败')
    } finally {
      setBusy(false)
    }
  }

  async function runtimeAction(item: Item, action: 'start' | 'stop') {
    setBusy(true)
    setMessage('')
    try {
      await request(`/admin/projects/${item.id}/runtime-jobs`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      })
      setMessage(action === 'start' ? '项目已加入构建队列，状态会自动更新。' : '停止任务已提交。')
      await refreshLatest.current()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  function selectKind(next: Kind) {
    if (kind === next) return
    currentListKey.current = listKey(next, 1, 'pending', '', token)
    requestSequence.current++
    setKind(next)
    setPage(1)
    setSearch('')
    setSearchInput('')
    setReviewStatus('pending')
    setItems([])
    setTotal(0)
    setForm(null)
    setGalleryFile(null)
    setCoverFile(null)
    setCoverPreview('')
    setPreviewMarkdown(false)
    setMessage('')
  }

  function changePage(next: number) {
    currentListKey.current = listKey(kind, next, reviewStatus, search, token)
    requestSequence.current++
    setItems([])
    setPage(next)
  }

  function changeReviewStatus(next: string) {
    currentListKey.current = listKey(kind, 1, next, search, token)
    requestSequence.current++
    setItems([])
    setPage(1)
    setReviewStatus(next)
  }

  function submitSearch(event: FormEvent) {
    event.preventDefault()
    const next = searchInput.trim()
    currentListKey.current = listKey(kind, 1, reviewStatus, next, token)
    requestSequence.current++
    setItems([])
    setPage(1)
    setSearch(next)
  }

  const visibleItems = items
  const activeTab = tabs.find((tab) => tab.kind === kind)
  const adminUser = readUser()

  if (!authorized)
    return (
      <div className="admin-login-wrap">
        <div className="admin-login-card">
          <div className="admin-login-art">
            <span className="admin-login-art-mark">
              <img src="/logo.svg" alt="" />
            </span>
            <small>DEVHUB / STUDIO</small>
            <div>
              <span>写作 · 作品 · 小店</span>
              <h2>
                把这里的每一页，
                <br />
                慢慢打理好。
              </h2>
              <p>文章、项目、商品和读者消息，登录后都能在工作台里处理。</p>
            </div>
            <div className="admin-login-art-line">
              <span>PERSONAL WORKSPACE</span>
              <span>2026</span>
            </div>
          </div>
          <form className="admin-login" onSubmit={login}>
            <span className="admin-kicker">站点管理 / 登录</span>
            <h1>回到工作台</h1>
            <p>使用站长账号登录。</p>
            <label htmlFor="admin-identifier">用户名或邮箱</label>
            <input
              id="admin-identifier"
              required
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              autoComplete="username"
              placeholder="输入用户名或邮箱"
            />
            <label htmlFor="admin-password">密码</label>
            <input
              id="admin-password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              placeholder="输入密码"
            />
            <button type="submit" disabled={busy}>
              {busy ? '登录中…' : '进入管理台'} <ArrowRight size={17} />
            </button>
            {token && (
              <button
                type="button"
                className="admin-retry"
                onClick={() => setAuthRetry((value) => value + 1)}
              >
                重试验证当前登录
              </button>
            )}
            {message && (
              <span className="admin-message" role="status">
                {message}
              </span>
            )}
            <span className="admin-login-note">读者可以在网站登录。<Link to="/login">前往登录</Link> · <Link to="/">返回网站</Link></span>
          </form>
        </div>
      </div>
    )

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar" aria-label="管理导航">
        <div className="admin-sidebar-brand">
          <span className="admin-sidebar-mark">
            <img src="/logo.svg" alt="" />
          </span>
          <span>
            <strong>DevHub</strong>
            <small>CONTROL ROOM</small>
          </span>
        </div>
        {(['workspace', 'content', 'review'] as const).map((group) => (
          <div className="admin-nav-group" key={group}>
            <p>
              {group === 'workspace' ? '工作台' : group === 'content' ? '内容与业务' : '互动审核'}
            </p>
            {tabs
              .filter((tab) => tab.group === group)
              .map((tab) => {
                const Icon = tab.icon
                return (
                  <button
                    type="button"
                    className={kind === tab.kind ? 'active' : ''}
                    aria-current={kind === tab.kind ? 'page' : undefined}
                    key={tab.kind}
                    onClick={() => selectKind(tab.kind)}
                  >
                    <Icon size={17} /> <span>{tab.label}</span>
                    {(tab.kind === 'comments' || tab.kind === 'applications' || tab.kind === 'author-applications') &&
                      stats[tab.kind] !== undefined &&
                      stats[tab.kind] !== 0 && <em>{stats[tab.kind]}</em>}
                  </button>
                )
              })}
          </div>
        ))}
        <div className="admin-sidebar-bottom">
          <div className="admin-user-avatar">
            {adminUser?.username?.slice(0, 1).toUpperCase() || 'D'}
          </div>
          <div>
            <strong>{adminUser?.username || '站长'}</strong>
            <small>站点管理员</small>
          </div>
          <button type="button" onClick={logout} aria-label="退出登录" title="退出登录">
            <LogOut size={17} />
          </button>
        </div>
      </aside>
      <main className="admin-workspace">
        <div className="admin-breadcrumb">
          DevHub <ChevronRight size={14} /> 管理台 <ChevronRight size={14} /> {activeTab?.label}
        </div>
        <div className="admin-top">
          <div>
            <span className="admin-kicker">
              站点管理 / {kind === 'overview' ? '总览' : '内容工作区'}
            </span>
            <h1>{kind === 'overview' ? '今天也把这里打理好。' : activeTab?.label}</h1>
            <p>
              {kind === 'overview'
                ? '从内容发布到读者交流，都可以在这里处理。'
                : `管理${activeTab?.label}，保存后可在前台查看最新内容。`}
            </p>
          </div>
          <a className="admin-view-site" href="/" target="_blank" rel="noopener noreferrer">
            查看网站 <ExternalLink size={15} />
          </a>
        </div>
        {message && (
          <p className="admin-message" role="status">
            {message}
          </p>
        )}
        {kind === 'overview' ? (
          <div className="admin-overview">
            <div className="admin-overview-banner">
              <div>
                <span>CONTENTS · COMMERCE · COMMUNITY</span>
                <h2>你的个人空间，正在生长。</h2>
                <p>发布一篇文章，整理一个项目，或者先处理读者发来的消息。</p>
              </div>
              <button
                type="button"
                onClick={() => {
                  selectKind('articles')
                  setForm({ ...blank.articles })
                }}
              >
                写篇文章 <ArrowRight size={17} />
              </button>
            </div>
            <div className="admin-stat-grid">
              {(['articles', 'projects', 'products', 'orders'] as const).map((entry) => {
                const tab = tabs.find((candidate) => candidate.kind === entry)!
                const Icon = tab.icon
                return (
                  <button
                    type="button"
                    className="admin-stat"
                    key={entry}
                    onClick={() => selectKind(entry)}
                  >
                    <span>
                      <Icon size={18} />
                    </span>
                    <small>{tab.label}</small>
                    <strong>{stats[entry] ?? '—'}</strong>
                    <em>
                      查看详情 <ArrowRight size={13} />
                    </em>
                  </button>
                )
              })}
            </div>
            <div className="admin-analytics-grid">
              <section className="admin-overview-card admin-analytics-card">
                <div className="admin-overview-card-title"><h2>最近 7 天访问</h2><span>匿名页面浏览</span></div>
                <div className="admin-analytics-numbers"><strong>{analytics?.last_7_days ?? '—'}</strong><span>总浏览 · 今日 {analytics?.today ?? '—'}</span></div>
                <div className="admin-analytics-chart" aria-label="最近七天访问趋势">
                  {Array.from({ length: 7 }, (_, index) => {
                    const date = new Date(Date.now() - (6 - index) * 86400000).toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
                    const count = analytics?.daily.find((entry) => entry.day === date)?.count ?? 0
                    const highest = Math.max(1, ...(analytics?.daily.map((entry) => entry.count) ?? []))
                    return <div key={date} className="admin-analytics-day" title={`${date} · ${count} 次`}><span><i style={{ height: `${Math.max(5, count / highest * 100)}%` }} /></span><small>{date.slice(5)}</small></div>
                  })}
                </div>
              </section>
              <section className="admin-overview-card admin-analytics-card">
                <div className="admin-overview-card-title"><h2>热门页面</h2><span>最近 7 天</span></div>
                {analytics?.top_pages.length ? analytics.top_pages.map((entry, index) => <div className="admin-popular-row" key={`${entry.kind}:${entry.slug}`}><em>{String(index + 1).padStart(2, '0')}</em><span><small>{entry.kind === 'article' ? '文章' : entry.kind === 'product' ? '商品' : '项目'}</small><strong>{entry.slug}</strong></span><b>{entry.count}</b></div>) : <p className="admin-analytics-empty">还没有访问记录。读者打开文章或预览项目后，这里会出现趋势。</p>}
              </section>
            </div>
            <div className="admin-overview-columns">
              <section className="admin-overview-card">
                <div className="admin-overview-card-title">
                  <h2>待处理</h2>
                  <span>读者互动</span>
                </div>
                {(['comments', 'applications', 'author-applications'] as const).map((entry) => {
                  const tab = tabs.find((candidate) => candidate.kind === entry)!
                  const Icon = tab.icon
                  return (
                    <button
                      type="button"
                      className="admin-review-link"
                      key={entry}
                      onClick={() => selectKind(entry)}
                    >
                      <span>
                        <Icon size={19} />
                      </span>
                      <div>
                        <strong>{tab.label}</strong>
                        <small>
                          {stats[entry] === undefined
                            ? '正在读取'
                            : stats[entry] === 0
                              ? '暂无待处理内容'
                              : `${stats[entry]} 条等待处理`}
                        </small>
                      </div>
                      <ChevronRight size={18} />
                    </button>
                  )
                })}
              </section>
              <section className="admin-overview-card">
                <div className="admin-overview-card-title">
                  <h2>快速开始</h2>
                  <span>常用操作</span>
                </div>
                <button
                  type="button"
                  className="admin-quick-link"
                  onClick={() => {
                    selectKind('projects')
                    setForm({ ...blank.projects })
                  }}
                >
                  <FolderKanban size={19} /> 添加项目 <ArrowRight size={16} />
                </button>
                <button
                  type="button"
                  className="admin-quick-link"
                  onClick={() => {
                    selectKind('products')
                    setForm({ ...blank.products })
                  }}
                >
                  <ShoppingBag size={19} /> 上架商品 <ArrowRight size={16} />
                </button>
                <button
                  type="button"
                  className="admin-quick-link"
                  onClick={() => selectKind('links')}
                >
                  <Link2 size={19} /> 管理友链 <ArrowRight size={16} />
                </button>
              </section>
            </div>
          </div>
        ) : (
          <>
            <div className="admin-toolbar">
              <div className="admin-list-heading">
                <strong>内容列表</strong>
                <span>共 {total} 条</span>
              </div>
              <div>
                <button type="button" onClick={() => void refresh()}>
                  <RefreshCw size={16} /> 刷新
                </button>
                {isEditableKind(kind) && (
                  <button
                    type="button"
                    className="admin-add"
                    onClick={() => {
                      setForm({ ...blank[kind] })
                      setGalleryFile(null)
                      setCoverFile(null)
                      setCoverPreview('')
                      setPreviewMarkdown(false)
                    }}
                  >
                    <Plus size={16} /> 新建{activeTab?.label.replace('管理', '')}
                  </button>
                )}
              </div>
            </div>
            <div className="admin-filters">
              <form onSubmit={submitSearch}>
              <label>
                <Search size={17} />
                <input
                  aria-label="搜索全部记录"
                  value={searchInput}
                  onChange={(event) => setSearchInput(event.target.value)}
                  placeholder="搜索全部记录"
                />
              </label>
              <button type="submit">搜索</button>
              </form>
              {(kind === 'comments' || kind === 'applications' || kind === 'author-applications') && (
                <select
                  aria-label="审核状态"
                  value={reviewStatus}
                  onChange={(event) => changeReviewStatus(event.target.value)}
                >
                  <option value="pending">待审核</option>
                  <option value="approved">已通过</option>
                  <option value="rejected">已驳回</option>
                </select>
              )}
            </div>
            {kind === 'projects' && (
              <div className="admin-project-help">
                <span>项目预览支持外部地址、静态页面 ZIP 和完整前后端 ZIP。完整项目上传后，可直接在列表里启动或停止。</span>
                <a href="/downloads/project-runtime-template.zip" download>下载完整项目模板</a>
              </div>
            )}
            {kind === 'gallery' && <div className="admin-project-help"><span>照片先上传为草稿，保存时可直接选择公开。相册目前保持空白，等你自己上传照片。</span></div>}
            {form && isEditableKind(kind) && (
              <form className="admin-editor" onSubmit={save}>
                <div className="admin-editor-head">
                  <h2>
                    {form.id ? '编辑' : '新建'}
                    {tabs.find((tab) => tab.kind === kind)?.label}
                  </h2>
                  <button type="button" onClick={() => { setForm(null); setGalleryFile(null); setCoverFile(null); setCoverPreview('') }}>
                    取消
                  </button>
                </div>
                {kind === 'articles' && (
                  <div className="admin-editor-switch" role="group" aria-label="文章编辑模式">
                    <button
                      type="button"
                      className={!previewMarkdown ? 'active' : ''}
                      onClick={() => setPreviewMarkdown(false)}
                    >
                      编辑内容
                    </button>
                    <button
                      type="button"
                      className={previewMarkdown ? 'active' : ''}
                      onClick={() => setPreviewMarkdown(true)}
                    >
                      预览排版
                    </button>
                  </div>
                )}
                {kind === 'articles' && previewMarkdown ? (
                  <div className="admin-article-preview">
                    <span>文章预览</span>
                    <h1>{String(form.title || '未填写标题')}</h1>
                    <p>{String(form.excerpt || '摘要会显示在这里。')}</p>
                    <div className="admin-preview-markdown markdown">
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>
                        {String(form.body_md || '*正文还没有内容。*')}
                      </ReactMarkdown>
                    </div>
                  </div>
                ) : (
                  <div className="admin-fields">
                    {(kind === 'articles' || kind === 'projects' || kind === 'products') && <div className="admin-cover-upload">
                      <span>封面图片</span>
                      {(coverPreview || Boolean(form.cover_url)) && <img src={coverPreview || String(form.cover_url)} alt="封面预览" />}
                      <label>选择图片上传到本站
                        <input type="file" accept="image/png,image/jpeg" disabled={busy} onChange={(event) => {
                          const file = event.target.files?.[0]
                          event.target.value = ''
                          if (!file) return
                          if (file.size > 8 * 1024 * 1024 || !['image/png', 'image/jpeg'].includes(file.type)) { setMessage('封面需为不超过 8 MiB 的 PNG 或 JPEG'); return }
                          setCoverFile(file)
                          setCoverPreview(URL.createObjectURL(file))
                          setMessage('')
                        }} />
                      </label>
                      {(coverPreview || Boolean(form.cover_url)) && <button type="button" onClick={() => { setCoverFile(null); setCoverPreview(''); setForm({ ...form, cover_url: '' }) }}>移除封面</button>}
                      <small>选择后会在保存时上传，网站自动生成图片地址。</small>
                    </div>}
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
                        ) : field.key === 'taken_at' ? (
                          <input type="date" value={String(form.taken_at || '')} onChange={(e) => setForm({ ...form, taken_at: e.target.value })} />
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
                )}
                {kind === 'gallery' && <div className="admin-gallery-upload">
                  {Boolean(form.image_url) && form.status === 'published' && <img src={String(form.image_url)} alt={String(form.title || '当前照片')} />}
                  <label><UploadCloud size={18} /> {galleryFile ? galleryFile.name : form.id ? '更换照片（可选）' : '选择照片'}
                    <input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => setGalleryFile(event.target.files?.[0] || null)} />
                  </label>
                  <small>JPEG、PNG、WebP、GIF，最大 10 MiB</small>
                </div>}
                <button className="admin-save" type="submit" disabled={busy}>
                  <Check size={17} /> {busy ? '保存中…' : '保存'}
                </button>
              </form>
            )}
            <div className="admin-list">
              {visibleItems.length === 0 ? (
                <div className="admin-empty">
                  <Inbox size={28} />
                  <strong>{search ? '没有匹配内容' : '这里还没有记录'}</strong>
                  <span>
                    {search ? '换个关键词再试试。' : '你发布的内容会出现在这里。'}
                  </span>
                </div>
              ) : (
                <div className="admin-table-scroll">
                  <table className="admin-table">
                    <thead><tr><th scope="col">内容</th><th scope="col">标识 / 联系</th><th scope="col">状态</th><th scope="col">时间</th><th scope="col">操作</th></tr></thead>
                    <tbody>
                {visibleItems.map((item) => (
                  <tr key={item.id}>
                    <td className="admin-table-title" data-label="内容">
                      {kind === 'gallery' && item.status === 'published' && Boolean(item.image_url) && <img className="admin-gallery-thumb" src={String(item.image_url)} alt="" />}
                      <strong>{String(kind === 'orders' ? `${item.username || '历史订单'} · ${item.order_no || `#${item.id}`}` : item.title || item.name || item.article_title || item.username || `#${item.id}`)}</strong>
                      <small>{String(kind === 'comments' ? `${item.reply_to_username ? `回复 ${item.reply_to_username} · ` : ''}${item.body || ''}` : kind === 'author-applications' ? item.reason || '' : kind === 'orders' ? formatPrice(Number(item.total_cents || 0)) : item.description || item.excerpt || '')}</small>
                    </td>
                    <td data-label="标识 / 联系" className="admin-table-identifier">{String(item.slug || item.location || item.url || item.email || item.username || `#${item.id}`)}</td>
                    <td data-label="状态">
                      <span className={`admin-status ${item.status === 'published' || item.status === 'approved' || item.enabled === true ? 'is-positive' : ''}`}>
                        {kind === 'links' ? (item.enabled ? '公开' : '隐藏') : ({ published: '公开', draft: '草稿', pending: '待审核', approved: '已通过', rejected: '已驳回', pending_payment: '待付款', paid: '已付款', delivered: '已交付', cancelled: '已取消' } as Record<string, string>)[String(item.status)] || String(item.status || '—')}
                      </span>
                      {kind === 'projects' && item.runtime_status === 'running' && <small className="admin-runtime-status">前后端运行中</small>}
                      {kind === 'projects' && (item.runtime_job_status === 'queued' || item.runtime_job_status === 'running') && <small className="admin-runtime-status">{item.runtime_job_action === 'start' ? '正在启动' : '正在停止'} · {item.runtime_job_status === 'queued' ? '排队中' : '执行中'}</small>}
                      {kind === 'projects' && item.runtime_job_status === 'failed' && <small className="admin-runtime-error" title={String(item.runtime_job_error || '')}>上次操作失败：{String(item.runtime_job_error || '请检查运行服务')}</small>}
                    </td>
                    <td data-label="时间" className="admin-table-date">{item.created_at ? formatDate(String(item.created_at)) : '—'}</td>
                    <td data-label="操作"><div className="admin-row-actions">
                      {(kind === 'comments' || kind === 'applications' || kind === 'author-applications') &&
                      reviewStatus === 'pending' ? (
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
                              <button
                                type="button"
                                onClick={() => void changeOrder(item, 'cancelled')}
                              >
                                取消
                              </button>
                            </>
                          )}
                          {item.status === 'paid' && (
                            <button
                              type="button"
                              onClick={() => void changeOrder(item, 'delivered')}
                            >
                              已交付
                            </button>
                          )}
                        </>
                      ) : isEditableKind(kind) ? (
                        <>
                          {kind === 'projects' && (
                            <>
                              {item.runtime_status === 'running' ? (
                                <button type="button" disabled={busy || item.runtime_job_status === 'queued' || item.runtime_job_status === 'running'} onClick={() => void runtimeAction(item, 'stop')} title="停止完整项目"><Square size={15} /> 停止</button>
                              ) : item.runtime_bundle_uploaded && (
                                <button type="button" disabled={busy || item.runtime_job_status === 'queued' || item.runtime_job_status === 'running'} onClick={() => void runtimeAction(item, 'start')} title="构建并启动完整项目"><Play size={15} /> 启动</button>
                              )}
                              {item.runtime_status !== 'running' && item.runtime_job_status === 'failed' && (
                                <button type="button" disabled={busy} onClick={() => void runtimeAction(item, 'stop')} title="清理失败任务留下的容器和网络"><Square size={15} /> 清理</button>
                              )}
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
                              <label className="admin-upload" title="上传包含 frontend、backend 与 docker-compose.yml 的完整项目 ZIP">
                                <UploadCloud size={17} /> {item.runtime_bundle_uploaded ? '替换完整项目' : '上传完整项目'}
                                <input
                                  type="file"
                                  accept=".zip,application/zip"
                                  disabled={busy}
                                  onChange={(event) => {
                                    void uploadRuntimeBundle(item, event.target.files?.[0])
                                    event.target.value = ''
                                  }}
                                />
                              </label>
                            </>
                          )}
                          <button
                            type="button"
                            onClick={() => {
                              setForm(normalize(kind, item))
                              setGalleryFile(null)
                              setCoverFile(null)
                              setCoverPreview('')
                              setPreviewMarkdown(false)
                            }}
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
                    </div></td>
                  </tr>
                ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
            {(total > pageSize || page > 1) && (
              <div className="admin-pagination">
                <span>
                  第 {page} / {Math.max(1, Math.ceil(total / pageSize))} 页
                </span>
                <div>
                  <button type="button" disabled={page <= 1} onClick={() => changePage(page - 1)}>
                    <ChevronLeft size={16} /> 上一页
                  </button>
                  <button
                    type="button"
                    disabled={page >= Math.ceil(total / pageSize)}
                    onClick={() => changePage(page + 1)}
                  >
                    下一页 <ChevronRight size={16} />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </main>
    </div>
  )
}
