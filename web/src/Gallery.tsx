import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, Camera, MapPin, X } from 'lucide-react'
import { api, formatDate, type GalleryItem, type Page } from './api'
import './gallery.css'

export default function Gallery() {
  const [items, setItems] = useState<GalleryItem[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selected, setSelected] = useState<GalleryItem | null>(null)
  const dialog = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    let active = true
    setLoading(true)
    api<Page<GalleryItem>>(`/gallery?page=${page}&page_size=12`)
      .then((data) => {
        if (!active) return
        setItems((previous) => page === 1 ? data.items : [...previous, ...data.items])
        setTotal(data.total)
        setError('')
      })
      .catch((reason) => {
        if (active) setError(reason instanceof Error ? reason.message : '相册暂时无法加载')
      })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [page])

  useEffect(() => {
    if (selected && dialog.current && !dialog.current.open) dialog.current.showModal()
    if (!selected && dialog.current?.open) dialog.current.close()
  }, [selected])

  return <div className="gallery-page container">
    <header className="gallery-hero">
      <span>ALBUM / 影像手记</span>
      <h1>留住路过的光。</h1>
      <p>相片会慢慢放进来。先把这一页留给未来的风景。</p>
      <div className="gallery-hero-line"><i /><span>{total} 张照片</span><i /></div>
    </header>
    {error && <p className="gallery-error" role="status">{error}</p>}
    {!loading && items.length === 0 && !error && <section className="gallery-empty">
      <div className="gallery-empty-frames" aria-hidden="true"><span /><span /><span><Camera size={52} strokeWidth={1} /></span></div>
      <strong>相册还没有照片</strong>
      <p>等我整理好照片，再把故事放在这里。</p>
      <Link to="/travel">先看看游记 <ArrowRight size={16} /></Link>
    </section>}
    {items.length > 0 && <div className="gallery-grid">
      {items.map((item, index) => <button className="gallery-card" key={item.id} type="button" onClick={() => setSelected(item)} style={{ animationDelay: `${Math.min(index % 12, 8) * 55}ms` }}>
        <img src={item.image_url} alt={item.title} loading="lazy" />
        <span className="gallery-card-caption"><strong>{item.title}</strong><small>{item.location || formatDate(item.taken_at || item.created_at)}</small></span>
      </button>)}
    </div>}
    {loading && <p className="gallery-loading">正在翻开相册…</p>}
    {items.length < total && !loading && <button className="gallery-more" type="button" onClick={() => setPage((value) => value + 1)}>再看一些 <ArrowRight size={16} /></button>}
    <dialog className="gallery-lightbox" ref={dialog} onClose={() => setSelected(null)} onClick={(event) => { if (event.target === dialog.current) setSelected(null) }}>
      {selected && <div className="gallery-lightbox-content"><button type="button" aria-label="关闭照片" onClick={() => setSelected(null)}><X size={22} /></button><img src={selected.image_url} alt={selected.title} /><div><span>{formatDate(selected.taken_at || selected.created_at)}</span><h2>{selected.title}</h2>{selected.location && <small><MapPin size={14} />{selected.location}</small>}{selected.description && <p>{selected.description}</p>}</div></div>}
    </dialog>
  </div>
}
