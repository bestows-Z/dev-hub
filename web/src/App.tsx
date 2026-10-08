import { toast } from 'sonner'
import {
  useEffect,
  lazy,
  Suspense,
  useRef,
  useState,
  type FormEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react'
import { Link, NavLink, Route, Routes, useLocation, useParams } from 'react-router-dom'
import { Account, Login, Register } from './Auth'
import { Comments, LinkApplicationForm } from './Engagement'
import { authChanged, readToken, readUser } from './session'
import SiteEffects from './SiteEffects'

import {
  ArrowDown,
  ArrowRight,
  ArrowUpRight,
  Check,
  ChevronLeft,
  ChevronRight,
  Code2,
  ExternalLink,
  Github,
  Menu,
  Minus,
  Package,
  PenLine,
  Send,
  UserRound,
  Volume2,
  VolumeX,
  X,
} from 'lucide-react'
import {
  api,
  formatDate,
  formatPrice,
  type Article,
  type ArticleSummary,
  type FriendLink,
  type Page,
  type Product,
  type Project,
} from './api'

const ArticleReading = lazy(() => import('./ArticleReading'))
const Admin = lazy(() => import('./Admin'))
const Studio = lazy(() => import('./Studio'))
const Gallery = lazy(() => import('./Gallery'))
type Remote<T> = { data: T | null; loading: boolean; error: string }
const contactEmail = import.meta.env.VITE_CONTACT_EMAIL as string | undefined
const contactHref = contactEmail ? `mailto:${contactEmail}` : 'https://github.com/bestows-Z'
const contactLabel = contactEmail ? '邮件联系站长' : '在 GitHub 联系站长'

function useRemote<T>(path: string): Remote<T> {
  const [state, setState] = useState<Remote<T>>({ data: null, loading: true, error: '' })
  useEffect(() => {
    let active = true
    setState({ data: null, loading: true, error: '' })
    api<T>(path)
      .then((data) => {
        if (active) setState({ data, loading: false, error: '' })
      })
      .catch((error) => {
        if (active)
          setState({
            data: null,
            loading: false,
            error: error instanceof Error ? error.message : '加载失败',
          })
      })
    return () => {
      active = false
    }
  }, [path])
  return state
}

function Layout({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [visitor, setVisitor] = useState(readUser)
  const location = useLocation()
  useEffect(() => {
    const refresh = () => setVisitor(readUser())
    window.addEventListener(authChanged, refresh)
    window.addEventListener('storage', refresh)
    return () => {
      window.removeEventListener(authChanged, refresh)
      window.removeEventListener('storage', refresh)
    }
  }, [])
  useEffect(() => {
    setMenuOpen(false)
    window.scrollTo({ top: 0, behavior: 'instant' })
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    const targets = document.querySelectorAll(
      '.home-main .section, .home-sidebar .sidebar-card, .page-intro',
    )
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('in-view')
            observer.unobserve(entry.target)
          }
        })
      },
      { rootMargin: '0px 0px -45px 0px', threshold: 0.08 },
    )
    targets.forEach((target) => {
      target.classList.add('section-motion')
      observer.observe(target)
    })
    return () => observer.disconnect()
  }, [location.pathname])
  const links = [
    ['/articles', '文章'],
    ['/projects', '项目'],
    ['/shop', '商店'],
    ['/gallery', '相册'],
    ['/links', '友链'],
  ]
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main">
        跳转到正文
      </a>
      <header className={location.pathname === '/' ? 'site-header home-header' : 'site-header'}>
        <div className="header-inner container">
          <Link className="brand" to="/" aria-label="DevHub 首页">
            <img className="brand-mark" src="/logo.svg" alt="" />
            <span>DevHub</span>
          </Link>
          <nav className={menuOpen ? 'primary-nav open' : 'primary-nav'} aria-label="主导航">
            {links.map(([href, label]) => (
              <NavLink
                key={href}
                to={href}
                className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
              >
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="header-actions">
            <Link
              className="header-login"
              to={visitor ? '/account' : '/login'}
              aria-label={visitor ? `我的账户：${visitor.username}` : '登录或注册'}
            >
              <UserRound size={16} /> <span>{visitor ? visitor.username : '登录'}</span>
            </Link>
            <a className="header-contact" href={contactHref} aria-label={contactLabel}>
              联系我 <ArrowUpRight size={16} />
            </a>
            <button
              className="menu-button"
              type="button"
              onClick={() => setMenuOpen(!menuOpen)}
              aria-expanded={menuOpen}
              aria-label={menuOpen ? '关闭菜单' : '打开菜单'}
            >
              {menuOpen ? <X size={23} /> : <Menu size={23} />}
            </button>
          </div>
        </div>
      </header>
      <main id="main" tabIndex={-1} aria-label="正文">{children}</main>
      <footer className="site-footer">
        <div className="container footer-inner">
          <div>
            <Link className="brand footer-brand" to="/">
              <img className="brand-mark" src="/logo.svg" alt="" />
              <span>DevHub</span>
            </Link>
            <p>一个人的博客，记录正在发生的事。</p>
          </div>
          <div className="footer-links">
            <Link to="/articles">文章</Link>
            <Link to="/projects">项目</Link>
            <Link to="/shop">商店</Link>
            <Link to="/gallery">相册</Link>
            <Link to="/links">友链</Link>
          </div>
          <span className="copyright">© {new Date().getFullYear()} DevHub</span>
        </div>
      </footer>
      {!['/login', '/register', '/account'].includes(location.pathname) && <Assistant />}
      <SiteEffects />
    </div>
  )
}

function SectionTitle({
  eyebrow,
  title,
  description,
  to,
  linkText,
  number,
}: {
  eyebrow: string
  title: string
  description?: string
  to?: string
  linkText?: string
  number?: string
}) {
  return (
    <div className="section-heading">
      <div>
        <span className="eyebrow">
          <span className="eyebrow-line" />
          {eyebrow}
        </span>
        <h2>
          {number && <span className="section-number">{number}</span>}
          {title}
        </h2>
        {description && <p>{description}</p>}
      </div>
      {to && (
        <Link className="text-link" to={to}>
          {linkText || '查看全部'} <ArrowUpRight size={17} />
        </Link>
      )}
    </div>
  )
}

function Status({
  loading,
  error,
  empty,
  children,
  emptyTitle = '暂时没有内容',
  emptyText = '发布后会显示在这里。',
}: {
  loading: boolean
  error: string
  empty: boolean
  children: ReactNode
  emptyTitle?: string
  emptyText?: string
}) {
  if (loading)
    return (
      <div className="status-card loading">
        <span className="loader" />
        正在加载内容…
      </div>
    )
  if (error)
    return (
      <div className="status-card error">
        <Minus size={18} />
        {error}
      </div>
    )
  if (empty)
    return (
      <div className="status-card empty">
        <PenLine size={24} />
        <strong>{emptyTitle}</strong>
        <span>{emptyText}</span>
      </div>
    )
  return <>{children}</>
}

function ArticleCard({ article, index = 0 }: { article: ArticleSummary; index?: number }) {
  return (
    <Link
      to={`/articles/${article.slug}`}
      className={`article-card card-enter${index === 0 ? ' article-featured' : ''}`}
      style={{ animationDelay: `${index * 70}ms` }}
    >
      <div
        className="article-visual"
        style={article.cover_url ? { backgroundImage: `url(${article.cover_url})` } : undefined}
      >
        {!article.cover_url && <PenLine size={48} strokeWidth={1} aria-hidden="true" />}
      </div>
      <div className="article-info">
        {index === 0 && <span className="featured-kicker">本期新文</span>}
        <div className="meta-row">
          <span>{formatDate(article.published_at)}</span>
          {article.tags?.[0] && (
            <>
              <span className="meta-divider" />
              <span>{article.tags[0]}</span>
            </>
          )}
        </div>
        <h3>{article.title}</h3>
        <p>{article.excerpt}</p>
        <span className="card-read">
          阅读全文 <ArrowRight size={15} />
        </span>
      </div>
    </Link>
  )
}

function Home() {
  const visitor = readUser()
  const articles = useRemote<Page<ArticleSummary>>('/articles?page_size=5')
  const projects = useRemote<Page<Project>>('/projects?page_size=2')
  const products = useRemote<Page<Product>>('/products?page_size=2')
  const links = useRemote<FriendLink[]>('/links')
  return (
    <>
      <section className="hero blog-hero">
        <div className="hero-cover" />
        <div className="hero-center">
          <span>欢迎来到这里</span>
          <h1>DevHub · 个人博客</h1>
          <p>写点东西，做点项目，记下路上的发现。</p>
        </div>
        <a className="hero-down" href="#latest" aria-label="查看最新内容">
          <ArrowDown size={22} />
        </a>
        <div className="hero-wave" aria-hidden="true" />
      </section>
      <nav className="home-index container" aria-label="快速浏览">
        <span className="home-index-label">从这里开始</span>
        <a href="#home-articles"><PenLine size={19} /><strong>读几篇文章</strong><small>{articles.data?.total ?? '—'} 篇</small><ArrowUpRight size={16} /></a>
        <a href="#home-projects"><Code2 size={20} /><strong>看看项目</strong><small>{projects.data?.total ?? '—'} 个</small><ArrowUpRight size={16} /></a>
        <a href="#home-shop"><Package size={19} /><strong>逛逛小店</strong><small>{products.data?.total ?? '—'} 件</small><ArrowUpRight size={16} /></a>
      </nav>
      <div className="home-content container" id="latest">
        <div className="home-main">
          <section className="section home-articles" id="home-articles">
            <SectionTitle
              eyebrow="最近更新"
              title="文章"
              number="01 /"
              description="技术笔记，也有一些日常。"
              to="/articles"
              linkText="更多文章"
            />
            <Status
              loading={articles.loading}
              error={articles.error}
              empty={!articles.data?.items.length}
              emptyTitle="下一篇文章正在路上"
              emptyText="写好的文字会在这里出现。"
            >
              <div className="article-grid">
                {articles.data?.items.map((article, index) => (
                  <ArticleCard key={article.id} article={article} index={index} />
                ))}
              </div>
            </Status>
          </section>
          <section className="section home-projects" id="home-projects">
            <SectionTitle
              eyebrow="动手做的"
              title="项目"
              number="02 /"
              description="一些已经完成和仍在改进的作品。"
              to="/projects"
              linkText="全部项目"
            />
            <Status
              loading={projects.loading}
              error={projects.error}
              empty={!projects.data?.items.length}
              emptyTitle="项目还在打磨"
              emptyText="完成的作品会放在这里，也会附上预览入口。"
            >
              <div className="project-grid">
                {projects.data?.items.map((project, index) => (
                  <ProjectCard key={project.id} project={project} index={index} />
                ))}
              </div>
            </Status>
          </section>
          <section className="section home-shop" id="home-shop">
            <SectionTitle
              eyebrow="小小商店"
              title="数字作品"
              number="03 /"
              description="整理好的工具与资源。"
              to="/shop"
              linkText="进入商店"
            />
            <Status
              loading={products.loading}
              error={products.error}
              empty={!products.data?.items.length}
              emptyTitle="小店正在准备"
              emptyText="工具和资源整理好后会摆上货架。"
            >
              <div className="product-grid">
                {products.data?.items.map((product) => (
                  <ProductCard key={product.id} product={product} />
                ))}
              </div>
            </Status>
          </section>
        </div>
        <aside className="home-sidebar" aria-label="关于本站">
          <div className="sidebar-card visitor-card">
            <span className="visitor-card-kicker">你的空间</span>
            <div className="visitor-card-person">
              <span className="visitor-card-avatar">
                {visitor?.avatar_url ? (
                  <img src={visitor.avatar_url} alt="" />
                ) : (
                  <UserRound size={23} />
                )}
              </span>
              <div>
                <strong>{visitor?.display_name || visitor?.username || '你好，访客'}</strong>
                <small>{visitor ? '欢迎回来' : '登录后参与讨论'}</small>
              </div>
            </div>
            <Link to={visitor ? '/account' : '/login'}>
              {visitor ? '查看我的资料' : '登录 / 注册'} <ArrowRight size={15} />
            </Link>
          </div>
          <div className="sidebar-card profile-card">
            <div className="profile-cover" />
            <div className="profile-avatar">
              <img src="/logo.svg" alt="DevHub" />
            </div>
            <h2>关于 DevHub</h2>
            <p>写代码，也记生活。把折腾过的项目和读过、想过的东西放在这里。</p>
            <div className="profile-stats">
              <span>
                <strong>{articles.data?.total ?? '—'}</strong>文章
              </span>
              <span>
                <strong>{projects.data?.total ?? '—'}</strong>项目
              </span>
              <span>
                <strong>{products.data?.total ?? '—'}</strong>作品
              </span>
            </div>
            <a href={contactHref} className="profile-contact">
              {contactEmail ? '发封邮件' : 'GitHub 主页'} <ArrowUpRight size={15} />
            </a>
          </div>
          <div className="sidebar-card">
            <h3>站点小记</h3>
            <p>这里会慢慢更新文章、项目和一些有用的小东西。欢迎常来看看。</p>
          </div>
          <div className="sidebar-card">
            <h3>友邻</h3>
            {links.data?.length ? (
              <div className="sidebar-links">
                {links.data.slice(0, 5).map((link) => (
                  <a key={link.id} href={link.url} target="_blank" rel="noopener noreferrer">
                    {link.name}
                    <ArrowUpRight size={14} />
                  </a>
                ))}
              </div>
            ) : (
              <p>友链整理中。</p>
            )}
            <Link className="sidebar-more" to="/links">
              查看全部友链 <ArrowRight size={14} />
            </Link>
          </div>
        </aside>
      </div>
    </>
  )
}

function Articles() {
  const location = useLocation()
  const category =
    location.pathname === '/travel'
      ? 'travel'
      : location.pathname === '/essays'
        ? 'essay'
        : location.pathname === '/records'
          ? 'record'
          : ''
  const sections = [
    { path: '/articles', label: '全部', category: '' },
    { path: '/travel', label: '游记', category: 'travel' },
    { path: '/essays', label: '随笔', category: 'essay' },
    { path: '/records', label: '记录', category: 'record' },
  ]
  const sectionCopy: Record<string, [string, string]> = {
    travel: ['路上的风景', '走过的地方，等我慢慢写下来。'],
    essay: ['随笔', '没有固定主题，只记录当时的想法。'],
    record: ['生活记录', '留住一些平常的小事。'],
  }
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState('')
  const [search, setSearch] = useState('')
  const [tag, setTag] = useState('')
  const facets = useRemote<{
    categories: { name: string; count: number }[]
    tags: { name: string; count: number }[]
  }>('/articles/facets')
  const remote = useRemote<Page<ArticleSummary>>(
    `/articles?page=${page}&page_size=9&q=${encodeURIComponent(search)}&category=${category}&tag=${encodeURIComponent(tag)}`,
  )
  function submit(event: FormEvent) {
    event.preventDefault()
    setPage(1)
    setSearch(query.trim())
  }
  return (
    <div className="container page-layout">
      <PageIntro
        eyebrow="文章归档"
        title={sectionCopy[category]?.[0] || '写过的文章'}
        description={sectionCopy[category]?.[1] || '技术笔记和随手记录，都整理在这里。'}
      />
      <div className="article-categories" aria-label="文章栏目">
        {sections.map((section) => {
          const count = facets.data?.categories.find(
            (item) => item.name === section.category,
          )?.count
          return (
            <Link
              key={section.path}
              to={section.path}
              className={location.pathname === section.path ? 'active' : ''}
              onClick={() => {
                setPage(1)
                setTag('')
              }}
            >
              {section.label} {count ? <small>{count}</small> : null}
            </Link>
          )
        })}
      </div>
      <form className="search-form" onSubmit={submit}>
        <label htmlFor="article-search">搜索文章</label>
        <div>
          <input
            id="article-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="输入标题或关键词"
          />
          <button type="submit" aria-label="搜索文章">
            <ArrowRight size={20} />
          </button>
        </div>
      </form>
      {!!facets.data?.tags.length && (
        <div className="article-facets" aria-label="按标签筛选">
          <button
            type="button"
            className={!tag ? 'active' : ''}
            onClick={() => {
              setTag('')
              setPage(1)
            }}
          >
            全部标签
          </button>
          {facets.data.tags.map((item) => (
            <button
              type="button"
              key={item.name}
              className={tag === item.name ? 'active' : ''}
              onClick={() => {
                setTag(item.name)
                setPage(1)
              }}
            >
              # {item.name} <small>{item.count}</small>
            </button>
          ))}
        </div>
      )}
      <Status
        loading={remote.loading}
        error={remote.error}
        empty={!remote.data?.items.length}
        emptyTitle={search || tag ? '没找到相关文章' : category ? '这里还没有内容' : '文章正在整理'}
        emptyText={
          search || tag
            ? '换个关键词或标签试试。'
            : category
              ? '以后发布的内容会出现在这个栏目。'
              : '新的笔记发布后会出现在这里。'
        }
      >
        <div className="article-grid">
          {remote.data?.items.map((article, index) => (
            <ArticleCard key={article.id} article={article} index={index} />
          ))}
        </div>
      </Status>
      {remote.data && remote.data.total > 9 && (
        <div className="pagination">
          <button type="button" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            <ChevronLeft size={18} /> 上一页
          </button>
          <span>
            {page} / {Math.ceil(remote.data.total / 9)}
          </span>
          <button
            type="button"
            disabled={page * 9 >= remote.data.total}
            onClick={() => setPage(page + 1)}
          >
            下一页 <ChevronRight size={18} />
          </button>
        </div>
      )}
    </div>
  )
}

function ArticleDetail() {
  const { slug } = useParams()
  const remote = useRemote<Article>(`/articles/${encodeURIComponent(slug || '')}`)
  return (
    <div className="container article-detail reading-detail">
      <Link className="back-link" to="/articles">
        <ChevronLeft size={17} /> 返回文章
      </Link>
      <Status loading={remote.loading} error={remote.error} empty={!remote.data}>
        {remote.data && (
          <article>
            <Suspense fallback={<div className="loading-state">正在整理文章…</div>}>
              <ArticleReading article={remote.data} />
            </Suspense>
          </article>
        )}
      </Status>
      {remote.data && <Comments slug={remote.data.slug} />}
    </div>
  )
}

function PageIntro({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string
  title: string
  description: string
}) {
  return (
    <div className="page-intro">
      <span className="eyebrow">
        <span className="eyebrow-line" />
        {eyebrow}
      </span>
      <h1>{title}</h1>
      <p>{description}</p>
    </div>
  )
}

function ProjectCard({ project, index }: { project: Project; index: number }) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewState, setPreviewState] = useState<'checking' | 'ready' | 'unavailable'>('checking')
  const dialogRef = useRef<HTMLDialogElement>(null)
  const previewRequest = useRef<AbortController | null>(null)
  const localPreview = project.preview_url?.startsWith('/api/v1/project-runtimes/') || project.preview_url?.startsWith('/api/v1/project-previews/')
  function showPreview() {
    setPreviewOpen(true)
    setPreviewState('checking')
    if (!dialogRef.current?.open) dialogRef.current?.showModal()
    const controller = new AbortController()
    previewRequest.current = controller
    fetch(project.preview_url, { signal: controller.signal }).then((response) => {
      void response.body?.cancel().catch(() => {})
      if (!controller.signal.aborted) setPreviewState(response.ok ? 'ready' : 'unavailable')
    }).catch(() => { if (!controller.signal.aborted) setPreviewState('unavailable') })
  }
  function closePreview() { previewRequest.current?.abort(); dialogRef.current?.close() }
  return (
    <article className="project-card card-enter" style={{ animationDelay: `${index * 65}ms` }}>
      <div
        className="project-cover"
        style={project.cover_url ? { backgroundImage: `url(${project.cover_url})` } : undefined}
      >
        {!project.cover_url && <Code2 size={46} strokeWidth={1} />}
        <span className="project-index">项目 {String(index + 1).padStart(2, '0')}</span>
        <span className="project-cover-arrow">
          <ArrowUpRight size={19} />
        </span>
        {project.runtime_status === 'running' && <span className="project-live-badge">已部署</span>}
      </div>
      <div className="project-body">
        <div className="tag-row">
          {project.tags?.slice(0, 3).map((tag) => (
            <span key={tag}>{tag}</span>
          ))}
        </div>
        <h3>{project.title}</h3>
        <p>{project.description}</p>
        <div className="card-actions">
          {project.preview_url && localPreview && (
            <button type="button" onClick={showPreview}>
              站内预览 <ExternalLink size={16} />
            </button>
          )}
          {project.preview_url && !localPreview && (
            <a href={project.preview_url} target="_blank" rel="noopener noreferrer">
              外部预览 <ExternalLink size={16} />
            </a>
          )}
          {project.backend_url && (
            <a href={project.backend_url} target="_blank" rel="noopener noreferrer">
              API 预览 <ExternalLink size={16} />
            </a>
          )}
          {project.source_url && (
            <a
              href={project.source_url}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={`${project.title} 源码`}
            >
              <Github size={17} />
            </a>
          )}
        </div>
      </div>
      {localPreview && <dialog ref={dialogRef} className="project-preview-dialog" onClose={() => { previewRequest.current?.abort(); setPreviewOpen(false) }} onClick={(event) => { if (event.target === event.currentTarget) closePreview() }} aria-label={`${project.title} 站内预览`}>
        <div className="project-preview-topbar">
          <div><span className="project-preview-dot" /><strong>{project.title}</strong><small>站内隔离预览</small></div>
          <button type="button" onClick={closePreview} aria-label="关闭项目预览"><X size={20} /></button>
        </div>
        {previewOpen && previewState === 'checking' && <div className="project-preview-message">正在连接项目…</div>}
        {previewOpen && previewState === 'unavailable' && <div className="project-preview-message"><strong>项目暂时无法访问</strong><span>运行容器可能已经停止，请稍后再试。</span><button type="button" onClick={showPreview}>重新连接</button></div>}
        {previewOpen && previewState === 'ready' && <iframe title={`${project.title} 预览`} src={project.preview_url} sandbox="allow-scripts allow-forms allow-downloads" referrerPolicy="no-referrer" />}
      </dialog>}
    </article>
  )
}

function Projects() {
  const remote = useRemote<Page<Project>>('/projects?page_size=50')
  const [activeTag, setActiveTag] = useState('全部')
  const tags = [...new Set(remote.data?.items.flatMap((project) => project.tags || []) || [])]
  const visibleProjects = remote.data?.items.filter(
    (project) => activeTag === '全部' || project.tags?.includes(activeTag),
  )
  return (
    <div className="container page-layout">
      <PageIntro eyebrow="项目记录" title="做过的项目" description="已经上线和正在维护的项目。" />
      {tags.length > 0 && (
        <div className="catalog-toolbar" aria-label="按标签筛选项目">
          {['全部', ...tags].map((tag) => (
            <button
              key={tag}
              type="button"
              className={tag === activeTag ? 'catalog-chip active' : 'catalog-chip'}
              onClick={() => setActiveTag(tag)}
              aria-pressed={tag === activeTag}
            >
              {tag}
            </button>
          ))}
          <span className="catalog-count">{visibleProjects?.length || 0} 个项目</span>
        </div>
      )}
      <Status
        loading={remote.loading}
        error={remote.error}
        empty={!remote.data?.items.length}
        emptyTitle="项目还在打磨"
        emptyText="作品完成后会放在这里，附上预览与源码入口。"
      >
        <div className="project-grid">
          {visibleProjects?.map((project, index) => (
            <ProjectCard key={project.id} project={project} index={index} />
          ))}
        </div>
      </Status>
    </div>
  )
}

function ProductCard({ product, index = 0 }: { product: Product; index?: number }) {
  return (
    <Link
      to={`/shop/${product.slug}`}
      className="product-card card-enter"
      style={{ animationDelay: `${index * 65}ms` }}
    >
      <div
        className="product-cover"
        style={product.cover_url ? { backgroundImage: `url(${product.cover_url})` } : undefined}
      >
        {!product.cover_url && <Package size={42} strokeWidth={1} />}
        <span className="product-badge">数字商品</span>
      </div>
      <div className="product-info">
        <h3>{product.name}</h3>
        <p>{product.description}</p>
        <div>
          <strong>{formatPrice(product.price_cents)}</strong>
          <span>
            了解详情 <ArrowUpRight size={16} />
          </span>
        </div>
      </div>
    </Link>
  )
}

function Shop() {
  const remote = useRemote<Page<Product>>('/products?page_size=50')
  const [sort, setSort] = useState('newest')
  const visibleProducts = [...(remote.data?.items || [])].sort((a, b) =>
    sort === 'low'
      ? a.price_cents - b.price_cents
      : sort === 'high'
        ? b.price_cents - a.price_cents
        : 0,
  )
  return (
    <div className="container page-layout">
      <PageIntro eyebrow="小商店" title="数字作品" description="整理好的资源和小工具。" />
      {!!remote.data?.items.length && (
        <div className="catalog-toolbar shop-toolbar">
          <span className="catalog-count">共 {remote.data.total} 件作品</span>
          <label htmlFor="shop-sort">排序</label>
          <select id="shop-sort" value={sort} onChange={(event) => setSort(event.target.value)}>
            <option value="newest">最新上架</option>
            <option value="low">价格从低到高</option>
            <option value="high">价格从高到低</option>
          </select>
        </div>
      )}
      <Status
        loading={remote.loading}
        error={remote.error}
        empty={!remote.data?.items.length}
        emptyTitle="小店正在准备"
        emptyText="工具和资源整理好后会摆上货架。"
      >
        <div className="product-grid">
          {visibleProducts.map((product, index) => (
            <ProductCard key={product.id} product={product} index={index} />
          ))}
        </div>
      </Status>
    </div>
  )
}

function ProductDetail() {
  const { slug } = useParams()
  const remote = useRemote<Product>(`/products/${encodeURIComponent(slug || '')}`)
  const [customer, setCustomer] = useState(readUser)
  const [quantity, setQuantity] = useState(1)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [createdOrder, setCreatedOrder] = useState(false)
  useEffect(() => {
    const refresh = () => setCustomer(readUser())
    window.addEventListener(authChanged, refresh)
    window.addEventListener('storage', refresh)
    return () => {
      window.removeEventListener(authChanged, refresh)
      window.removeEventListener('storage', refresh)
    }
  }, [])
  async function order(event: FormEvent) {
    event.preventDefault()
    if (!remote.data || !customer || !readToken()) return
    setBusy(true)
    setNotice('')
    setCreatedOrder(false)
    try {
      const result = await api<{ order_no: string }>('/orders', {
        method: 'POST',
        headers: { Authorization: `Bearer ${readToken()}` },
        body: JSON.stringify({ product_id: remote.data.id, quantity }),
      })
      setNotice(`订单 ${result.order_no} 已创建，当前状态为待付款。`)
      setCreatedOrder(true)
      toast.success('订单已创建', { description: `订单号 ${result.order_no}，请在个人中心查看。` })
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '下单失败')
      toast.error(error instanceof Error ? error.message : '下单失败')
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="container page-layout">
      <Link className="back-link" to="/shop">
        <ChevronLeft size={17} /> 返回商店
      </Link>
      <Status loading={remote.loading} error={remote.error} empty={!remote.data}>
        {remote.data && (
          <div className="product-detail">
            <div
              className="product-detail-cover"
              style={
                remote.data.cover_url
                  ? { backgroundImage: `url(${remote.data.cover_url})` }
                  : undefined
              }
            >
              <Package size={82} strokeWidth={0.8} />
            </div>
            <div>
              <span className="eyebrow">数字商品</span>
              <h1>{remote.data.name}</h1>
              <p>{remote.data.description}</p>
              <strong className="detail-price">{formatPrice(remote.data.price_cents)}</strong>
              {customer ? <form className="order-form" onSubmit={order}>
                <p className="order-account">购买账号：<strong>{customer.username}</strong><br />订单通知将发送至 {customer.email}</p>
                <label htmlFor="order-quantity">数量</label>
                <input
                  id="order-quantity"
                  type="number"
                  min="1"
                  max={Math.min(remote.data.stock, 10)}
                  required
                  value={quantity}
                  onChange={(event) => setQuantity(Number(event.target.value))}
                />
                <button
                  className="button button-primary"
                  type="submit"
                  disabled={busy || remote.data.stock < 1}
                >
                  {busy ? '正在提交…' : remote.data.stock < 1 ? '暂时售罄' : '提交订单'}{' '}
                  <ArrowRight size={18} />
                </button>
                {notice && (
                  <p className="form-notice" role="status">
                    {notice} {createdOrder && <Link to="/account">在个人中心查看订单</Link>}
                  </p>
                )}
              </form> : <div className="order-signin">
                <p>登录后才能提交订单。商品价格和库存会在提交时再次确认。</p>
                <Link className="button button-primary" to={`/login?next=${encodeURIComponent(`/shop/${slug || ''}`)}`}>
                  登录后继续 <ArrowRight size={18} />
                </Link>
              </div>}
              <p className="purchase-note">
                <Check size={15} /> 订单提交后由站长通过邮件确认付款与交付。
              </p>
            </div>
          </div>
        )}
      </Status>
    </div>
  )
}

function Links() {
  const remote = useRemote<FriendLink[]>('/links')
  return (
    <div className="container page-layout">
      <PageIntro eyebrow="友链" title="朋友们的站点" description="看看朋友们最近都在写什么。" />
      <Status loading={remote.loading} error={remote.error} empty={!remote.data?.length}>
        <div className="links-grid">
          {remote.data?.map((link) => (
            <a
              className="friend-card"
              key={link.id}
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
            >
              <div className="friend-avatar">
                {link.avatar_url ? (
                  <img src={link.avatar_url} alt="" />
                ) : (
                  <span>{link.name.slice(0, 1)}</span>
                )}
              </div>
              <div>
                <h3>{link.name}</h3>
                <p>{link.description}</p>
              </div>
              <ArrowUpRight size={20} />
            </a>
          ))}
        </div>
      </Status>
      <LinkApplicationForm />
    </div>
  )
}

type Message = {
  role: 'user' | 'assistant'
  content: string
  sources?: { title: string; url: string }[]
}
type KeeperPoint = { x: number; y: number }
const keeperPositionKey = 'devhub_keeper_position'
function clampKeeper(point: KeeperPoint): KeeperPoint {
  return {
    x: Math.max(12, Math.min(point.x, window.innerWidth - 124)),
    y: Math.max(12, Math.min(point.y, window.innerHeight - 140)),
  }
}
function readKeeperPosition(): KeeperPoint | null {
  try {
    const saved = JSON.parse(
      localStorage.getItem(keeperPositionKey) || 'null',
    ) as KeeperPoint | null
    return saved && Number.isFinite(saved.x) && Number.isFinite(saved.y) ? clampKeeper(saved) : null
  } catch {
    return null
  }
}
function Assistant() {
  const [open, setOpen] = useState(false)
  const [walking, setWalking] = useState(false)
  const [greeting, setGreeting] = useState(false)
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [speakingIndex, setSpeakingIndex] = useState<number | null>(null)
  const [position, setPosition] = useState<KeeperPoint | null>(readKeeperPosition)
  const [viewport, setViewport] = useState({ width: window.innerWidth, height: window.innerHeight })
  const dragRef = useRef<{
    pointerId: number
    startX: number
    startY: number
    origin: KeeperPoint
    moved: boolean
  } | null>(null)
  const latestPositionRef = useRef<KeeperPoint | null>(position)
  const suppressClickRef = useRef(false)
  const dragCleanupRef = useRef<(() => void) | null>(null)
  const speechRef = useRef<SpeechSynthesisUtterance | null>(null)
  const [messages, setMessages] = useState<Message[]>([
    { role: 'assistant', content: '你好，有关于博客文章的问题可以问我。我会尽量附上原文出处。' },
  ])
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])
  useEffect(() => {
    return () => {
      if ('speechSynthesis' in window) window.speechSynthesis.cancel()
      dragCleanupRef.current?.()
    }
  }, [])
  useEffect(() => {
    const resize = () => {
      setViewport({ width: window.innerWidth, height: window.innerHeight })
      setPosition((current) => {
        if (!current) return null
        const next = clampKeeper(current)
        latestPositionRef.current = next
        try {
          localStorage.setItem(keeperPositionKey, JSON.stringify(next))
        } catch {
          // The character can still be moved when browser storage is unavailable.
        }
        return next
      })
    }
    window.addEventListener('resize', resize)
    return () => window.removeEventListener('resize', resize)
  }, [])
  useEffect(() => {
    const preload = window.setTimeout(() => {
      ;['/images/blog-keeper-walk.png', '/images/blog-keeper-wave.png'].forEach((src) => {
        const image = new Image()
        image.src = src
      })
    }, 1200)
    return () => window.clearTimeout(preload)
  }, [])
  useEffect(() => {
    if (
      open ||
      position !== null ||
      busy ||
      speakingIndex !== null ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      setWalking(false)
      return
    }
    if (greeting) return
    const start = window.setTimeout(() => setWalking(true), 5000)
    const repeat = window.setInterval(() => setWalking(true), 18000)
    return () => {
      window.clearTimeout(start)
      window.clearInterval(repeat)
    }
  }, [open, busy, speakingIndex, greeting, position])
  useEffect(() => {
    if (!walking || greeting) return
    const stop = window.setTimeout(() => setWalking(false), 6200)
    return () => window.clearTimeout(stop)
  }, [walking, greeting])
  function readAloud(index: number, content: string) {
    if (!('speechSynthesis' in window)) return
    if (speakingIndex === index) {
      window.speechSynthesis.cancel()
      setSpeakingIndex(null)
      return
    }
    window.speechSynthesis.cancel()
    const utterance = new SpeechSynthesisUtterance(content)
    utterance.lang = 'zh-CN'
    utterance.rate = 1
    utterance.onend = () => setSpeakingIndex(null)
    utterance.onerror = () => setSpeakingIndex(null)
    speechRef.current = utterance
    setSpeakingIndex(index)
    window.speechSynthesis.speak(utterance)
  }
  async function send(event: FormEvent) {
    event.preventDefault()
    const question = input.trim()
    if (!question || busy) return
    setMessages((current) => [...current, { role: 'user', content: question }])
    setInput('')
    setBusy(true)
    try {
      const result = await api<{ answer: string; sources: { title: string; url: string }[] }>(
        '/assistant/chat',
        { method: 'POST', body: JSON.stringify({ question }) },
      )
      setMessages((current) => [
        ...current,
        { role: 'assistant', content: result.answer, sources: result.sources },
      ])
    } catch (error) {
      setMessages((current) => [
        ...current,
        {
          role: 'assistant',
          content: error instanceof Error ? error.message : '暂时无法回答，请稍后再试。',
        },
      ])
    } finally {
      setBusy(false)
    }
  }
  function startDrag(event: ReactPointerEvent<HTMLButtonElement>) {
    if (event.pointerType === 'mouse' && event.button !== 0) return
    dragCleanupRef.current?.()
    const rect = event.currentTarget.getBoundingClientRect()
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      origin: { x: rect.left, y: rect.top },
      moved: false,
    }
    event.currentTarget.setPointerCapture(event.pointerId)
    const move = (pointer: globalThis.PointerEvent) => {
      const drag = dragRef.current
      if (!drag || drag.pointerId !== pointer.pointerId) return
      const dx = pointer.clientX - drag.startX
      const dy = pointer.clientY - drag.startY
      if (!drag.moved && Math.hypot(dx, dy) < 6) return
      drag.moved = true
      const next = clampKeeper({ x: drag.origin.x + dx, y: drag.origin.y + dy })
      latestPositionRef.current = next
      setPosition(next)
      setWalking(false)
      setGreeting(false)
    }
    const finish = (pointer: globalThis.PointerEvent) => {
      const drag = dragRef.current
      if (!drag || drag.pointerId !== pointer.pointerId) return
      if (drag.moved) {
        suppressClickRef.current = true
        window.setTimeout(() => {
          suppressClickRef.current = false
        }, 0)
        try {
          localStorage.setItem(keeperPositionKey, JSON.stringify(latestPositionRef.current))
        } catch {
          // The current position remains usable until the page reloads.
        }
      }
      dragRef.current = null
      dragCleanupRef.current?.()
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', finish)
    window.addEventListener('pointercancel', finish)
    dragCleanupRef.current = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', finish)
      window.removeEventListener('pointercancel', finish)
      dragCleanupRef.current = null
    }
  }
  const panelWidth = Math.min(375, viewport.width - 24)
  const panelHeight = Math.min(510, Math.max(160, viewport.height - 160))
  const panelStyle = position
    ? {
        left: Math.max(
          12,
          Math.min(position.x + 112 - panelWidth, viewport.width - panelWidth - 12),
        ),
        top:
          position.y - panelHeight - 10 >= 12
            ? position.y - panelHeight - 10
            : Math.min(position.y + 138, viewport.height - panelHeight - 12),
        width: panelWidth,
        height: panelHeight,
      }
    : undefined
  return (
    <div
      className={`assistant-root${walking ? ' strolling' : ''}${greeting ? ' greeting' : ''}${position ? ' positioned' : ''}`}
      style={position ? { left: position.x, top: position.y } : undefined}
    >
      <button
        type="button"
        className={`assistant-trigger${open ? ' active' : ''}${busy ? ' waiting' : ''}${speakingIndex !== null ? ' speaking' : ''}${greeting ? ' greeting' : ''}${walking ? ' walking' : ''}`}
        onMouseEnter={() => setGreeting(true)}
        onMouseLeave={() => setGreeting(false)}
        onFocus={() => setGreeting(true)}
        onBlur={() => setGreeting(false)}
        onPointerDown={startDrag}
        onClick={() => {
          if (suppressClickRef.current) return
          if (open && 'speechSynthesis' in window) window.speechSynthesis.cancel()
          setSpeakingIndex(null)
          setOpen(!open)
        }}
        aria-label={open ? '关闭数字助手' : '打开数字助手'}
        aria-description="按住并拖动可以调整位置"
        aria-expanded={open}
        title="按住拖动我"
      >
        <img
          className="assistant-character"
          draggable={false}
          src={
            greeting
              ? '/images/blog-keeper-wave.png'
              : walking
                ? '/images/blog-keeper-walk.png'
                : '/images/blog-keeper.png'
          }
          alt=""
        />
        <span className="assistant-trigger-label">
          {busy
            ? '正在找文章…'
            : speakingIndex !== null
              ? '正在说话…'
              : greeting
                ? '你好呀！'
                : walking
                  ? '到处看看…'
                  : '来聊聊？'}
        </span>
        {open && (
          <span className="assistant-close-mark">
            <X size={14} />
          </span>
        )}
      </button>
      {open && (
        <section
          className={`assistant-panel${position ? ' positioned' : ''}`}
          style={panelStyle}
          aria-label="数字助手对话"
        >
          <div className="assistant-head">
            <div className="assistant-mini">
              <img src="/images/blog-keeper.png" alt="" />
            </div>
            <div>
              <strong>博客小助手</strong>
              <span>
                <i /> 从文章里找答案
              </span>
            </div>
            {position && (
              <button
                type="button"
                className="assistant-reset"
                onClick={() => {
                  setPosition(null)
                  latestPositionRef.current = null
                  try {
                    localStorage.removeItem(keeperPositionKey)
                  } catch {
                    // Reset still works for the current page.
                  }
                }}
              >
                归位
              </button>
            )}
            <button type="button" aria-label="关闭对话" onClick={() => setOpen(false)}>
              <X size={19} />
            </button>
          </div>
          <div className="assistant-messages">
            {messages.map((message, index) => (
              <div className={`message ${message.role}`} key={index}>
                <p>{message.content}</p>
                {message.role === 'assistant' && 'speechSynthesis' in window && (
                  <button
                    type="button"
                    className="message-read"
                    onClick={() => readAloud(index, message.content)}
                    aria-label={speakingIndex === index ? '停止朗读' : '朗读回答'}
                  >
                    {speakingIndex === index ? <VolumeX size={15} /> : <Volume2 size={15} />}
                    {speakingIndex === index ? '停止' : '朗读'}
                  </button>
                )}
                {message.sources?.length ? (
                  <div className="message-sources">
                    参考：
                    {message.sources.map((source) => (
                      <a href={source.url} key={source.url}>
                        {source.title}
                      </a>
                    ))}
                  </div>
                ) : null}
              </div>
            ))}
            {busy && (
              <div className="message assistant typing">
                <span />
                <span />
                <span />
              </div>
            )}
            <div ref={endRef} />
          </div>
          <form className="assistant-compose" onSubmit={send}>
            <label className="sr-only" htmlFor="assistant-question">
              向数字助手提问
            </label>
            <input
              id="assistant-question"
              value={input}
              onChange={(event) => setInput(event.target.value)}
              placeholder="问我关于文章的内容…"
              maxLength={500}
            />
            <button type="submit" disabled={!input.trim() || busy} aria-label="发送消息">
              <Send size={18} />
            </button>
          </form>
        </section>
      )}
    </div>
  )
}

function NotFound() {
  return (
    <div className="container not-found">
      <span>404 / LOST IN SPACE</span>
      <h1>这页走丢了。</h1>
      <p>回到起点，继续探索吧。</p>
      <Link className="button button-primary" to="/">
        返回首页 <ArrowRight size={18} />
      </Link>
    </div>
  )
}

export default function App() {
  const location = useLocation()
  if (location.pathname.startsWith('/admin')) {
    return (
      <Suspense fallback={<div className="loading-state">正在打开管理台…</div>}>
        <Admin />
      </Suspense>
    )
  }
  if (location.pathname.startsWith('/studio')) {
    return <Suspense fallback={<div className="loading-state">正在打开写作台…</div>}><Studio /></Suspense>
  }
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/articles" element={<Articles />} />
        <Route path="/travel" element={<Articles />} />
        <Route path="/essays" element={<Articles />} />
        <Route path="/records" element={<Articles />} />
        <Route path="/articles/:slug" element={<ArticleDetail />} />
        <Route path="/projects" element={<Projects />} />
        <Route path="/gallery" element={<Suspense fallback={<div className="loading-state">正在翻开相册…</div>}><Gallery /></Suspense>} />
        <Route path="/shop" element={<Shop />} />
        <Route path="/shop/:slug" element={<ProductDetail />} />
        <Route path="/links" element={<Links />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/account" element={<Account />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Layout>
  )
}
