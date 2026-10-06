import { useEffect, useState, type FormEvent } from 'react'
import { ArrowLeft, BookOpen, Check, ChevronLeft, ChevronRight, Edit3, ExternalLink, LogOut, Plus, Trash2, UploadCloud } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Link, Navigate } from 'react-router-dom'
import { api, formatDate, type Page } from './api'
import { clearSession, readToken, readUser } from './session'
import './admin.css'
import './studio.css'

type Draft = {
  id?: number
  slug: string
  title: string
  excerpt: string
  body_md: string
  cover_url: string
  category: string
  tags: string[]
  status: string
  created_at?: string
}

const emptyDraft: Draft = { slug: '', title: '', excerpt: '', body_md: '', cover_url: '', category: 'tech', tags: [], status: 'draft' }
const categories = { tech: '技术', travel: '游记', essay: '随笔', record: '记录' } as const

export default function Studio() {
  const token = readToken()
  const user = readUser()
  const [verified, setVerified] = useState(false)
  const [unauthorized, setUnauthorized] = useState(false)
  const [items, setItems] = useState<Draft[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [tagsText, setTagsText] = useState('')
  const [cover, setCover] = useState<File | null>(null)
  const [coverPreview, setCoverPreview] = useState('')
  const [preview, setPreview] = useState(false)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => () => { if (coverPreview) URL.revokeObjectURL(coverPreview) }, [coverPreview])
  useEffect(() => {
    if (!token) return
    let active = true
    api<{ role: number }>('/auth/me', { headers: { Authorization: `Bearer ${token}` } })
      .then((account) => { if (active) { setVerified(true); setUnauthorized(account.role !== 1 && account.role !== 3) } })
      .catch(() => { if (active) { setVerified(true); setUnauthorized(true) } })
    return () => { active = false }
  }, [token])
  useEffect(() => {
    if (!verified || unauthorized || !token) return
    let active = true
    api<Page<Draft>>(`/studio/articles?page=${page}&page_size=12`, { headers: { Authorization: `Bearer ${token}` } })
      .then((result) => { if (active) { setItems(result.items); setTotal(result.total) } })
      .catch((error) => { if (active) setMessage(error instanceof Error ? error.message : '文章加载失败') })
    return () => { active = false }
  }, [token, verified, unauthorized, page, message === '已保存' || message === '已删除'])

  if (!token) return <Navigate to="/login?next=/studio" replace />
  if (verified && unauthorized) return <Navigate to="/account" replace />
  const headers = { Authorization: `Bearer ${token}` }
  async function reload() {
    const result = await api<Page<Draft>>(`/studio/articles?page=${page}&page_size=12`, { headers })
    setItems(result.items)
    setTotal(result.total)
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!draft) return
    setBusy(true)
    setMessage('')
    let uploadedID = ''
    try {
      let coverURL = draft.cover_url
      if (cover) {
        const body = new FormData()
        body.append('file', cover)
        const uploaded = await api<{ id: string; url: string }>('/studio/media', { method: 'POST', headers, body })
        uploadedID = uploaded.id
        coverURL = uploaded.url
      }
      const payload = { ...draft, cover_url: coverURL, tags: tagsText.split(',').map((tag) => tag.trim()).filter(Boolean) }
      await api(`/studio/articles${draft.id ? `/${draft.id}` : ''}`, { method: draft.id ? 'PUT' : 'POST', headers, body: JSON.stringify(payload) })
      setDraft(null)
      setCover(null)
      setCoverPreview('')
      setMessage('已保存')
      await reload()
    } catch (error) {
      if (uploadedID) void api(`/studio/media/${uploadedID}`, { method: 'DELETE', headers }).catch(() => {})
      setMessage(error instanceof Error ? error.message : '保存失败')
    } finally { setBusy(false) }
  }
  async function remove(item: Draft) {
    if (!window.confirm(`删除《${item.title}》？此操作无法撤销。`)) return
    setBusy(true)
    try {
      await api(`/studio/articles/${item.id}`, { method: 'DELETE', headers })
      setMessage('已删除')
      await reload()
    } catch (error) { setMessage(error instanceof Error ? error.message : '删除失败') }
    finally { setBusy(false) }
  }
  function edit(item?: Draft) {
    setDraft(item ? { ...item, tags: [...item.tags] } : { ...emptyDraft, tags: [] })
    setTagsText(item?.tags.join(', ') || '')
    setCover(null)
    setCoverPreview('')
    setPreview(false)
    setMessage('')
  }
  return <div className="studio-shell">
    <aside className="studio-side">
      <Link className="studio-brand" to="/"><img src="/logo.svg" alt="" /><span><strong>DevHub</strong><small>WRITING STUDIO</small></span></Link>
      <div className="studio-side-body"><span>写作空间</span><h1>慢慢写，<br />认真分享。</h1><p>草稿和已发布文章都在这里。你的作品由你自己管理。</p><BookOpen size={92} strokeWidth={1} /></div>
      <div className="studio-side-foot"><strong>{user?.display_name || user?.username || '作者'}</strong><span>本站作者</span></div>
    </aside>
    <main className="studio-main">
      <div className="studio-top"><Link to="/account"><ArrowLeft size={16} /> 我的账户</Link><Link to="/articles">阅读文章 <ExternalLink size={15} /></Link></div>
      <header className="studio-heading"><div><span>CONTENT / ARTICLES</span><h1>我的文章</h1><p>记录想法、整理经验，也可以先存成草稿。</p></div><button type="button" onClick={() => edit()}><Plus size={17} /> 写新文章</button></header>
      {message && <p className="studio-message" role="status">{message}</p>}
      {!verified ? <p className="studio-empty">正在确认写作权限…</p> : draft ? <form className="studio-editor" onSubmit={save}>
        <div className="studio-editor-head"><h2>{draft.id ? '编辑文章' : '新建文章'}</h2><button type="button" onClick={() => setDraft(null)}>关闭编辑</button></div>
        <div className="studio-fields"><label>标题<input required maxLength={240} value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="给这篇文章起个名字" /></label><label>网址标识<input required pattern="[a-z0-9]+(-[a-z0-9]+)*" value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="my-first-article" /></label><label>栏目<select value={draft.category} onChange={(event) => setDraft({ ...draft, category: event.target.value })}>{Object.entries(categories).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label><label>状态<select value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value })}><option value="draft">草稿</option><option value="published">公开发布</option></select></label><label className="wide">摘要<textarea rows={3} value={draft.excerpt} onChange={(event) => setDraft({ ...draft, excerpt: event.target.value })} placeholder="用一两句话介绍这篇文章" /></label><label className="wide">标签（逗号分隔）<input value={tagsText} onChange={(event) => setTagsText(event.target.value)} placeholder="前端, 随笔" /></label><div className="studio-cover wide"><span>封面</span>{(coverPreview || draft.cover_url) && <img src={coverPreview || draft.cover_url} alt="文章封面预览" />}<label><UploadCloud size={16} /> 上传图片<input type="file" accept="image/png,image/jpeg" onChange={(event) => { const file = event.target.files?.[0]; if (!file) return; if (file.size > 8 * 1024 * 1024) { setMessage('封面不能超过 8 MiB'); return } setCover(file); setCoverPreview(URL.createObjectURL(file)) }} /></label><button type="button" onClick={() => { setCover(null); setCoverPreview(''); setDraft({ ...draft, cover_url: '' }) }}>移除封面</button></div></div>
        <div className="studio-body-head"><span>正文 / MARKDOWN</span><div><button type="button" className={!preview ? 'active' : ''} onClick={() => setPreview(false)}>编辑</button><button type="button" className={preview ? 'active' : ''} onClick={() => setPreview(true)}>预览</button></div></div>
        {preview ? <div className="studio-markdown markdown"><ReactMarkdown remarkPlugins={[remarkGfm]}>{draft.body_md || '*还没有正文*'}</ReactMarkdown></div> : <textarea className="studio-body" required value={draft.body_md} onChange={(event) => setDraft({ ...draft, body_md: event.target.value })} placeholder="从这里开始写…" />}
        <button className="studio-save" type="submit" disabled={busy}><Check size={17} /> {busy ? '保存中…' : '保存文章'}</button>
      </form> : <section className="studio-list"><div className="studio-list-head"><strong>文章列表</strong><span>共 {total} 篇</span></div>{items.length ? items.map((item) => <article className="studio-row" key={item.id}><div><span>{item.status === 'published' ? '已发布' : '草稿'} · {categories[item.category as keyof typeof categories] || item.category}</span><h2>{item.title}</h2><p>{item.excerpt || '还没有摘要'}</p><small>{item.created_at ? formatDate(item.created_at) : ''} · /{item.slug}</small></div><div className="studio-row-actions">{item.status === 'published' && <Link to={`/articles/${item.slug}`} title="查看文章"><ExternalLink size={16} /></Link>}<button type="button" onClick={() => edit(item)} aria-label={`编辑 ${item.title}`}><Edit3 size={17} /></button><button type="button" disabled={busy} onClick={() => void remove(item)} aria-label={`删除 ${item.title}`}><Trash2 size={17} /></button></div></article>) : <div className="studio-empty"><BookOpen size={34} /><strong>还没有文章</strong><p>点击右上角，写下第一篇。</p></div>}{total > 12 && <div className="studio-pages"><button disabled={page === 1} onClick={() => setPage(page - 1)}><ChevronLeft size={16} /> 上一页</button><span>{page} / {Math.ceil(total / 12)}</span><button disabled={page >= Math.ceil(total / 12)} onClick={() => setPage(page + 1)}>下一页 <ChevronRight size={16} /></button></div>}</section>}
      <button className="studio-logout" type="button" onClick={() => { clearSession(); window.location.href = '/login' }}><LogOut size={15} /> 退出登录</button>
    </main>
  </div>
}
