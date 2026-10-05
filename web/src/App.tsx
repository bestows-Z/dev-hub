import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { Link, NavLink, Route, Routes, useLocation, useParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowDownRight, ArrowRight, ArrowUpRight, Bot, Check, ChevronLeft, ChevronRight, Code2, ExternalLink, Github, Menu, Minus, Package, Send, Sparkles, X } from 'lucide-react'
import { api, formatDate, formatPrice, type Article, type ArticleSummary, type FriendLink, type Page, type Product, type Project } from './api'

type Remote<T> = { data: T | null; loading: boolean; error: string }
const contactEmail = import.meta.env.VITE_CONTACT_EMAIL as string | undefined
const contactHref = contactEmail ? `mailto:${contactEmail}` : 'https://github.com/bestows-Z'
const contactLabel = contactEmail ? '邮件联系站长' : '在 GitHub 联系站长'

function useRemote<T>(path: string): Remote<T> {
  const [state, setState] = useState<Remote<T>>({ data: null, loading: true, error: '' })
  useEffect(() => {
    let active = true
    setState({ data: null, loading: true, error: '' })
    api<T>(path).then(data => { if (active) setState({ data, loading: false, error: '' }) })
      .catch(error => { if (active) setState({ data: null, loading: false, error: error instanceof Error ? error.message : '加载失败' }) })
    return () => { active = false }
  }, [path])
  return state
}

function Layout({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()
  useEffect(() => { setMenuOpen(false); window.scrollTo({ top: 0, behavior: 'instant' }) }, [location.pathname])
  const links = [['/articles', '文章'], ['/projects', '项目'], ['/shop', '商店'], ['/links', '友链']]
  return <div className="app-shell">
    <a className="skip-link" href="#main">跳转到正文</a>
    <div className="ambient ambient-one" aria-hidden="true" /><div className="ambient ambient-two" aria-hidden="true" />
    <header className="site-header">
      <div className="header-inner container">
        <Link className="brand" to="/" aria-label="DevHub 首页"><span className="brand-mark"><span /></span><span>dev<span className="brand-accent">hub</span><span className="brand-dot">.</span></span></Link>
        <nav className={menuOpen ? 'primary-nav open' : 'primary-nav'} aria-label="主导航">
          {links.map(([href, label]) => <NavLink key={href} to={href} className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>{label}</NavLink>)}
        </nav>
        <div className="header-actions"><a className="header-contact" href={contactHref} aria-label={contactLabel}>联系我 <ArrowUpRight size={16} /></a><button className="menu-button" type="button" onClick={() => setMenuOpen(!menuOpen)} aria-expanded={menuOpen} aria-label={menuOpen ? '关闭菜单' : '打开菜单'}>{menuOpen ? <X size={23} /> : <Menu size={23} />}</button></div>
      </div>
    </header>
    <main id="main">{children}</main>
    <footer className="site-footer"><div className="container footer-inner"><div><Link className="brand footer-brand" to="/"><span className="brand-mark"><span /></span><span>dev<span className="brand-accent">hub</span><span className="brand-dot">.</span></span></Link><p>把想法写下来，让作品发光。</p></div><div className="footer-links"><Link to="/articles">文章</Link><Link to="/projects">项目</Link><Link to="/shop">商店</Link><Link to="/links">友链</Link></div><span className="copyright">© {new Date().getFullYear()} DevHub</span></div></footer>
    <Assistant />
  </div>
}

function SectionTitle({ eyebrow, title, description, to, linkText }: { eyebrow: string; title: string; description?: string; to?: string; linkText?: string }) {
  return <div className="section-heading"><div><span className="eyebrow"><span className="eyebrow-line" />{eyebrow}</span><h2>{title}</h2>{description && <p>{description}</p>}</div>{to && <Link className="text-link" to={to}>{linkText || '查看全部'} <ArrowUpRight size={17} /></Link>}</div>
}

function Status({ loading, error, empty, children }: { loading: boolean; error: string; empty: boolean; children: ReactNode }) {
  if (loading) return <div className="status-card loading"><span className="loader" />正在加载内容…</div>
  if (error) return <div className="status-card error"><Minus size={18} />{error}</div>
  if (empty) return <div className="status-card empty"><Sparkles size={24} /><strong>这里还在酝酿中</strong><span>新内容发布后会第一时间出现在这里。</span></div>
  return <>{children}</>
}

function ArticleCard({ article, index = 0 }: { article: ArticleSummary; index?: number }) {
  return <Link to={`/articles/${article.slug}`} className="article-card reveal" style={{ animationDelay: `${index * 70}ms` }}><div className="article-visual" style={article.cover_url ? { backgroundImage: `url(${article.cover_url})` } : undefined}><span className="article-visual-grid" aria-hidden="true" /><span className="article-number">0{index + 1}</span><ArrowUpRight size={22} className="article-arrow" /></div><div className="article-info"><div className="meta-row"><span>{formatDate(article.published_at)}</span><span className="meta-divider" />{article.tags?.[0] && <span>{article.tags[0]}</span>}</div><h3>{article.title}</h3><p>{article.excerpt}</p><span className="card-read">阅读文章 <ArrowRight size={15} /></span></div></Link>
}

function Home() {
  const articles = useRemote<Page<ArticleSummary>>('/articles?page_size=3')
  const projects = useRemote<Page<Project>>('/projects?page_size=3')
  const products = useRemote<Page<Product>>('/products?page_size=3')
  return <>
    <section className="hero container"><div className="hero-copy"><div className="availability"><span className="availability-dot" />欢迎来到我的数字空间 <span className="availability-spark">✦</span></div><h1>把好奇心<br />变成<span className="headline-gradient">作品</span><span className="headline-star">✳</span></h1><p>记录思考，展示正在生长的项目，也分享那些值得被拥有的数字产品。这里是我的博客，也是一个持续迭代的实验场。</p><div className="hero-actions"><Link className="button button-primary" to="/projects">探索项目 <ArrowUpRight size={19} /></Link><Link className="button button-ghost" to="/articles">阅读文章 <ArrowRight size={18} /></Link></div><div className="hero-caption"><span className="caption-line" /><span>SCROLL TO EXPLORE</span><ArrowDownRight size={15} /></div></div><div className="hero-art" aria-label="抽象的开发者创作空间"><div className="hero-orbit orbit-one" /><div className="hero-orbit orbit-two" /><div className="hero-core"><div className="core-ring"><Code2 size={70} strokeWidth={1.1} /></div><span className="core-label">IDEAS<br />IN MOTION</span></div><div className="float-chip chip-top"><span className="chip-dot purple" /> CREATIVE CODE</div><div className="float-chip chip-bottom"><span className="chip-dot orange" /> ALWAYS BUILDING</div><div className="hero-cross cross-one">+</div><div className="hero-cross cross-two">+</div></div></section>
    <div className="ticker" aria-hidden="true"><div className="ticker-track">DESIGN <span>✦</span> ENGINEERING <span>✦</span> WRITING <span>✦</span> CURIOSITY <span>✦</span> DESIGN <span>✦</span> ENGINEERING <span>✦</span> WRITING <span>✦</span> CURIOSITY <span>✦</span></div></div>
    <section className="section container"><SectionTitle eyebrow="01 / THE JOURNAL" title="最新思考" description="关于技术、创造和日常观察的笔记。" to="/articles" linkText="全部文章" /><Status loading={articles.loading} error={articles.error} empty={!articles.data?.items.length}><div className="article-grid">{articles.data?.items.map((article, index) => <ArticleCard key={article.id} article={article} index={index} />)}</div></Status></section>
    <section className="feature-band"><div className="container feature-inner"><div className="feature-kicker"><span className="pulse-mark" /> INDEPENDENT BY DESIGN</div><h2>保持探索，<br /><em>永远在路上。</em></h2><p>从一个问题出发，写代码、做设计、不断尝试。好的作品总是在认真解决问题的过程中出现。</p><Link to="/projects" className="round-link" aria-label="探索全部项目"><ArrowUpRight size={25} /></Link></div></section>
    <section className="section container"><SectionTitle eyebrow="02 / SELECTED WORK" title="项目实验室" description="正在构建的工具、交互和有趣的尝试。" to="/projects" linkText="浏览作品" /><Status loading={projects.loading} error={projects.error} empty={!projects.data?.items.length}><div className="project-grid">{projects.data?.items.map((project, index) => <ProjectCard key={project.id} project={project} index={index} />)}</div></Status></section>
    <section className="section container shop-preview"><SectionTitle eyebrow="03 / DIGITAL STORE" title="数字商店" description="可下载的资源、工具和作品。" to="/shop" linkText="进入商店" /><Status loading={products.loading} error={products.error} empty={!products.data?.items.length}><div className="product-grid">{products.data?.items.map(product => <ProductCard key={product.id} product={product} />)}</div></Status></section>
    <section className="closing-cta container"><div className="closing-glow" /><span className="eyebrow">LET'S CONNECT</span><h2>下一段故事，<br /><span>或许从一句你好开始。</span></h2><a href={contactHref} className="button button-light">{contactEmail ? '给我发邮件' : '在 GitHub 联系'} <ArrowUpRight size={18} /></a></section>
  </>
}

function Articles() {
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState('')
  const [search, setSearch] = useState('')
  const remote = useRemote<Page<ArticleSummary>>(`/articles?page=${page}&page_size=9&q=${encodeURIComponent(search)}`)
  function submit(event: FormEvent) { event.preventDefault(); setPage(1); setSearch(query.trim()) }
  return <div className="container page-layout"><PageIntro eyebrow="THE JOURNAL" title="文字与思考。" description="记录那些值得反复思考的技术、设计和生活片段。" /><form className="search-form" onSubmit={submit}><label htmlFor="article-search">搜索文章</label><div><input id="article-search" value={query} onChange={event => setQuery(event.target.value)} placeholder="输入标题或关键词" /><button type="submit" aria-label="搜索文章"><ArrowRight size={20} /></button></div></form><Status loading={remote.loading} error={remote.error} empty={!remote.data?.items.length}><div className="article-grid">{remote.data?.items.map((article, index) => <ArticleCard key={article.id} article={article} index={index} />)}</div></Status>{remote.data && remote.data.total > 9 && <div className="pagination"><button type="button" disabled={page <= 1} onClick={() => setPage(page - 1)}><ChevronLeft size={18} /> 上一页</button><span>{page} / {Math.ceil(remote.data.total / 9)}</span><button type="button" disabled={page * 9 >= remote.data.total} onClick={() => setPage(page + 1)}>下一页 <ChevronRight size={18} /></button></div>}</div>
}

function ArticleDetail() {
  const { slug } = useParams()
  const remote = useRemote<Article>(`/articles/${encodeURIComponent(slug || '')}`)
  return <div className="container article-detail"><Link className="back-link" to="/articles"><ChevronLeft size={17} /> 返回文章</Link><Status loading={remote.loading} error={remote.error} empty={!remote.data}>{remote.data && <article><div className="article-head"><span className="eyebrow">THE JOURNAL / {formatDate(remote.data.published_at)}</span><h1>{remote.data.title}</h1><p>{remote.data.excerpt}</p><div className="tag-row">{remote.data.tags?.map(tag => <span key={tag}>{tag}</span>)}</div></div>{remote.data.cover_url && <img className="detail-cover" src={remote.data.cover_url} alt="文章封面" />}<div className="markdown"><ReactMarkdown remarkPlugins={[remarkGfm]}>{remote.data.body_md}</ReactMarkdown></div><div className="article-bottom"><span>感谢阅读。</span><Link to="/articles">继续探索 <ArrowRight size={17} /></Link></div></article>}</Status></div>
}

function PageIntro({ eyebrow, title, description }: { eyebrow: string; title: string; description: string }) { return <div className="page-intro"><span className="eyebrow"><span className="eyebrow-line" />{eyebrow}</span><h1>{title}</h1><p>{description}</p></div> }

function ProjectCard({ project, index }: { project: Project; index: number }) { return <article className="project-card reveal" style={{ animationDelay: `${index * 65}ms` }}><div className="project-cover" style={project.cover_url ? { backgroundImage: `url(${project.cover_url})` } : undefined}><Code2 size={46} strokeWidth={1} /><span>0{index + 1} / PROJECT</span></div><div className="project-body"><div className="tag-row">{project.tags?.slice(0, 3).map(tag => <span key={tag}>{tag}</span>)}</div><h3>{project.title}</h3><p>{project.description}</p><div className="card-actions">{project.preview_url && <a href={project.preview_url} target="_blank" rel="noopener noreferrer">在线预览 <ExternalLink size={16} /></a>}{project.source_url && <a href={project.source_url} target="_blank" rel="noopener noreferrer" aria-label={`${project.title} 源码`}><Github size={17} /></a>}</div></div></article> }

function Projects() { const remote = useRemote<Page<Project>>('/projects?page_size=50'); return <div className="container page-layout"><PageIntro eyebrow="THE LAB" title="作品正在发生。" description="从想法到上线，每一个项目都是一次有价值的探索。" /><Status loading={remote.loading} error={remote.error} empty={!remote.data?.items.length}><div className="project-grid">{remote.data?.items.map((project, index) => <ProjectCard key={project.id} project={project} index={index} />)}</div></Status></div> }

function ProductCard({ product }: { product: Product }) { return <Link to={`/shop/${product.slug}`} className="product-card"><div className="product-cover" style={product.cover_url ? { backgroundImage: `url(${product.cover_url})` } : undefined}><Package size={42} strokeWidth={1} /><span className="product-badge">DIGITAL GOODS</span></div><div className="product-info"><h3>{product.name}</h3><p>{product.description}</p><div><strong>{formatPrice(product.price_cents)}</strong><span>了解详情 <ArrowUpRight size={16} /></span></div></div></Link> }

function Shop() { const remote = useRemote<Page<Product>>('/products?page_size=50'); return <div className="container page-layout"><PageIntro eyebrow="THE STORE" title="好东西，值得分享。" description="精心制作的数字资源和小工具。" /><Status loading={remote.loading} error={remote.error} empty={!remote.data?.items.length}><div className="product-grid">{remote.data?.items.map(product => <ProductCard key={product.id} product={product} />)}</div></Status></div> }

function ProductDetail() {
  const { slug } = useParams()
  const remote = useRemote<Product>(`/products/${encodeURIComponent(slug || '')}`)
  const [email, setEmail] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  async function order(event: FormEvent) { event.preventDefault(); if (!remote.data) return; setBusy(true); setNotice(''); try { const result = await api<{ order_no: string }>('/orders', { method: 'POST', body: JSON.stringify({ product_id: remote.data.id, email, quantity }) }); setNotice(`订单 ${result.order_no} 已创建。站长会通过邮件联系你确认付款。`) } catch (error) { setNotice(error instanceof Error ? error.message : '下单失败') } finally { setBusy(false) } }
  return <div className="container page-layout"><Link className="back-link" to="/shop"><ChevronLeft size={17} /> 返回商店</Link><Status loading={remote.loading} error={remote.error} empty={!remote.data}>{remote.data && <div className="product-detail"><div className="product-detail-cover" style={remote.data.cover_url ? { backgroundImage: `url(${remote.data.cover_url})` } : undefined}><Package size={82} strokeWidth={0.8} /></div><div><span className="eyebrow">DIGITAL GOODS</span><h1>{remote.data.name}</h1><p>{remote.data.description}</p><strong className="detail-price">{formatPrice(remote.data.price_cents)}</strong><form className="order-form" onSubmit={order}><label htmlFor="order-email">接收订单通知的邮箱</label><input id="order-email" type="email" required value={email} onChange={event => setEmail(event.target.value)} placeholder="you@example.com" /><label htmlFor="order-quantity">数量</label><input id="order-quantity" type="number" min="1" max={Math.min(remote.data.stock, 10)} required value={quantity} onChange={event => setQuantity(Number(event.target.value))} /><button className="button button-primary" type="submit" disabled={busy || remote.data.stock < 1}>{busy ? '正在提交…' : remote.data.stock < 1 ? '暂时售罄' : '提交订单'} <ArrowRight size={18} /></button>{notice && <p className="form-notice" role="status">{notice}</p>}</form><p className="purchase-note"><Check size={15} /> 订单提交后由站长通过邮件确认付款与交付。</p></div></div>}</Status></div>
}

function Links() { const remote = useRemote<FriendLink[]>('/links'); return <div className="container page-layout"><PageIntro eyebrow="GOOD COMPANY" title="互联的朋友们。" description="互联网最迷人的地方，是偶然遇见同样热爱创造的人。" /><Status loading={remote.loading} error={remote.error} empty={!remote.data?.length}><div className="links-grid">{remote.data?.map(link => <a className="friend-card" key={link.id} href={link.url} target="_blank" rel="noopener noreferrer"><div className="friend-avatar">{link.avatar_url ? <img src={link.avatar_url} alt="" /> : <span>{link.name.slice(0, 1)}</span>}</div><div><h3>{link.name}</h3><p>{link.description}</p></div><ArrowUpRight size={20} /></a>)}</div></Status></div> }

type Message = { role: 'user' | 'assistant'; content: string; sources?: { title: string; url: string }[] }
function Assistant() {
  const [open, setOpen] = useState(false)
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [messages, setMessages] = useState<Message[]>([{ role: 'assistant', content: '你好！我是这个空间的数字助手。可以问我博客文章里聊过什么。' }])
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages, open])
  async function send(event: FormEvent) { event.preventDefault(); const question = input.trim(); if (!question || busy) return; setMessages(current => [...current, { role: 'user', content: question }]); setInput(''); setBusy(true); try { const result = await api<{ answer: string; sources: { title: string; url: string }[] }>('/assistant/chat', { method: 'POST', body: JSON.stringify({ question }) }); setMessages(current => [...current, { role: 'assistant', content: result.answer, sources: result.sources }]) } catch (error) { setMessages(current => [...current, { role: 'assistant', content: error instanceof Error ? error.message : '暂时无法回答，请稍后再试。' }]) } finally { setBusy(false) } }
  return <div className="assistant-root"><button type="button" className={open ? 'assistant-trigger active' : 'assistant-trigger'} onClick={() => setOpen(!open)} aria-label={open ? '关闭数字助手' : '打开数字助手'} aria-expanded={open}><span className="avatar-face"><span className="avatar-eye" /><span className="avatar-eye" /><span className="avatar-mouth" /></span>{open ? <X size={18} /> : <span className="assistant-trigger-label">问问小助手 <Sparkles size={15} /></span>}</button>{open && <section className="assistant-panel" aria-label="数字助手对话"><div className="assistant-head"><div className="assistant-mini"><Bot size={20} /></div><div><strong>DevHub 助手</strong><span><i /> 在线探索</span></div><button type="button" aria-label="关闭对话" onClick={() => setOpen(false)}><X size={19} /></button></div><div className="assistant-messages">{messages.map((message, index) => <div className={`message ${message.role}`} key={index}><p>{message.content}</p>{message.sources?.length ? <div className="message-sources">参考：{message.sources.map(source => <a href={source.url} key={source.url}>{source.title}</a>)}</div> : null}</div>)}{busy && <div className="message assistant typing"><span /><span /><span /></div>}<div ref={endRef} /></div><form className="assistant-compose" onSubmit={send}><label className="sr-only" htmlFor="assistant-question">向数字助手提问</label><input id="assistant-question" value={input} onChange={event => setInput(event.target.value)} placeholder="问我关于文章的内容…" maxLength={500} /><button type="submit" disabled={!input.trim() || busy} aria-label="发送消息"><Send size={18} /></button></form></section>}</div>
}

function NotFound() { return <div className="container not-found"><span>404 / LOST IN SPACE</span><h1>这页走丢了。</h1><p>回到起点，继续探索吧。</p><Link className="button button-primary" to="/">返回首页 <ArrowRight size={18} /></Link></div> }

export default function App() { return <Layout><Routes><Route path="/" element={<Home />} /><Route path="/articles" element={<Articles />} /><Route path="/articles/:slug" element={<ArticleDetail />} /><Route path="/projects" element={<Projects />} /><Route path="/shop" element={<Shop />} /><Route path="/shop/:slug" element={<ProductDetail />} /><Route path="/links" element={<Links />} /><Route path="*" element={<NotFound />} /></Routes></Layout> }
