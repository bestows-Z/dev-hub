import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, MessageCircle, Send } from 'lucide-react'
import { ApiError, api, formatDate, type Page } from './api'
import { authChanged, clearSession, readToken, readUser, type AuthUser } from './session'
import './engagement.css'

type Comment = { id: number; body: string; username: string; reply_to_id: number | null; reply_to_username: string; created_at: string }

function useVisitor() {
  const [user, setUser] = useState<AuthUser | null>(readUser)
  useEffect(() => {
    const update = () => setUser(readUser())
    window.addEventListener(authChanged, update)
    return () => window.removeEventListener(authChanged, update)
  }, [])
  return user
}

export function Comments({ slug }: { slug: string }) {
  const user = useVisitor()
  const [page, setPage] = useState(1)
  const [comments, setComments] = useState<Page<Comment> | null>(null)
  const [body, setBody] = useState('')
  const [replyTo, setReplyTo] = useState<Comment | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  useEffect(() => {
    let active = true
    api<Page<Comment>>(`/articles/${encodeURIComponent(slug)}/comments?page=${page}`)
      .then((result) => {
        if (active) {
          setComments(result)
          setError('')
        }
      })
      .catch((failure) => {
        if (active) setError(failure instanceof Error ? failure.message : '评论暂时无法读取')
      })
    return () => {
      active = false
    }
  }, [slug, page])
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!readToken()) return
    setBusy(true)
    setNotice('')
    try {
      await api(`/articles/${encodeURIComponent(slug)}/comments`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${readToken()}` },
        body: JSON.stringify({ body: body.trim(), reply_to_id: replyTo?.id || null }),
      })
      setBody('')
      setReplyTo(null)
      setNotice('评论已提交，审核通过后会显示在这里。')
    } catch (failure) {
      if (failure instanceof ApiError && failure.status === 401) clearSession()
      setNotice(failure instanceof Error ? failure.message : '提交失败，请重试。')
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="comments-section" aria-labelledby="comments-title">
      <div className="comments-heading">
        <span className="comments-icon">
          <MessageCircle size={20} />
        </span>
        <div>
          <h2 id="comments-title">聊聊这篇文章</h2>
          <p>{comments?.total ?? 0} 条已公开的评论</p>
        </div>
      </div>
      {user ? (
        <form className="comment-form" onSubmit={submit}>
          <div className="comment-identity">
            <span>{user.username.slice(0, 1).toUpperCase()}</span> 以 {user.username} 的身份留言
          </div>
          <label className="sr-only" htmlFor="comment-body">
            评论内容
          </label>
          {replyTo && <div className="comment-reply-target">正在回复 <strong>{replyTo.username}</strong><button type="button" onClick={() => setReplyTo(null)}>取消回复</button></div>}
          <textarea
            id="comment-body"
            ref={textareaRef}
            value={body}
            onChange={(event) => setBody(event.target.value)}
            minLength={3}
            maxLength={2000}
            required
            placeholder="说说你的想法，或补充一个有用的细节…"
          />
          <div className="comment-form-footer">
            <small>评论经站长审核后公开 · {body.length}/2000</small>
            <button type="submit" disabled={busy || body.trim().length < 3}>
              {busy ? '提交中…' : '发表留言'} <Send size={15} />
            </button>
          </div>
          {notice && (
            <p role="status" className="comment-notice">
              {notice}
            </p>
          )}
        </form>
      ) : (
        <div className="comment-login">
          登录后就可以参与讨论。
          <Link to="/login">
            去登录 <ArrowRight size={15} />
          </Link>
        </div>
      )}
      {error && (
        <p role="status" className="comment-notice">
          {error}
        </p>
      )}
      {comments?.items.length ? (
        <div className="comment-list">
          {comments.items.map((comment) => (
            <article className="comment-item" key={comment.id}>
              <span className="comment-avatar">{comment.username.slice(0, 1).toUpperCase()}</span>
              <div>
                <div className="comment-meta">
                  <strong>{comment.username}</strong>
                  <time>{formatDate(comment.created_at)}</time>
                </div>
                {comment.reply_to_id && <span className="comment-reply-context">回复 {comment.reply_to_username || '已删除的评论'}</span>}
                <p>{comment.body}</p>
                {user && <button className="comment-reply-button" type="button" onClick={() => { setReplyTo(comment); textareaRef.current?.focus(); textareaRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' }) }}>回复</button>}
              </div>
            </article>
          ))}
        </div>
      ) : !error ? (
        <p className="comments-empty">还没有评论。写下第一条吧。</p>
      ) : null}
      {comments && comments.total > 20 && (
        <div className="comments-pages">
          <button type="button" disabled={page === 1} onClick={() => setPage(page - 1)}>
            上一页
          </button>
          <span>
            {page} / {Math.ceil(comments.total / 20)}
          </span>
          <button
            type="button"
            disabled={page * 20 >= comments.total}
            onClick={() => setPage(page + 1)}
          >
            下一页
          </button>
        </div>
      )}
    </section>
  )
}

export function LinkApplicationForm() {
  const user = useVisitor()
  const [name, setName] = useState('')
  const [url, setURL] = useState('')
  const [avatarURL, setAvatarURL] = useState('')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!readToken()) return
    setBusy(true)
    setNotice('')
    try {
      await api('/link-applications', {
        method: 'POST',
        headers: { Authorization: `Bearer ${readToken()}` },
        body: JSON.stringify({
          name: name.trim(),
          url: url.trim(),
          avatar_url: avatarURL.trim(),
          description: description.trim(),
        }),
      })
      setNotice('申请已收到，站长审核后会出现在友链列表。')
      setName('')
      setURL('')
      setAvatarURL('')
      setDescription('')
    } catch (failure) {
      if (failure instanceof ApiError && failure.status === 401) clearSession()
      setNotice(failure instanceof Error ? failure.message : '申请暂时无法提交')
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="link-application">
      <div>
        <span className="eyebrow">加入友邻</span>
        <h2>申请友链</h2>
        <p>有自己的站点？把地址留在这里，审核后会展示。</p>
      </div>
      {user ? (
        <form onSubmit={submit}>
          <label>
            站点名称
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={100}
              required
              placeholder="你的博客名称"
            />
          </label>
          <label>
            站点地址
            <input
              type="url"
              value={url}
              onChange={(event) => setURL(event.target.value)}
              required
              placeholder="https://example.com"
            />
          </label>
          <label>
            头像地址（可选）
            <input
              type="url"
              value={avatarURL}
              onChange={(event) => setAvatarURL(event.target.value)}
              placeholder="https://example.com/avatar.png"
            />
          </label>
          <label>
            一句介绍
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              maxLength={280}
              rows={3}
              placeholder="你在写些什么？"
            />
          </label>
          <button type="submit" disabled={busy}>
            {busy ? '提交中…' : '提交申请'} <ArrowRight size={16} />
          </button>
          {notice && (
            <p role="status" className="comment-notice">
              {notice}
            </p>
          )}
        </form>
      ) : (
        <div className="comment-login">
          先登录，再提交你的站点。
          <Link to="/login">
            去登录 <ArrowRight size={15} />
          </Link>
        </div>
      )}
    </section>
  )
}
