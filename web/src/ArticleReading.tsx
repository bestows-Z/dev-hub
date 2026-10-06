import { useMemo, useRef, useState, type ReactNode, isValidElement } from 'react'
import { Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeSlug from 'rehype-slug'
import remarkParse from 'remark-parse'
import remarkRehype from 'remark-rehype'
import { unified } from 'unified'
import { toString } from 'hast-util-to-string'
import { visit } from 'unist-util-visit'
import { ArrowRight, Check, Clock3, Copy, List, UserRound } from 'lucide-react'
import { formatDate, type Article } from './api'
import { readUser } from './session'
import 'highlight.js/styles/github-dark.css'
import './article-reading.css'

type Heading = { id: string; text: string; depth: 2 | 3 }

function headingsFromMarkdown(source: string): Heading[] {
  const result: Heading[] = []
  const processor = unified().use(remarkParse).use(remarkGfm).use(remarkRehype).use(rehypeSlug)
  const tree = processor.runSync(processor.parse(source))
  visit(tree, 'element', (node) => {
    if (node.tagName !== 'h2' && node.tagName !== 'h3') return
    result.push({
      id: String(node.properties.id),
      text: toString(node),
      depth: node.tagName === 'h2' ? 2 : 3,
    })
  })
  return result
}

function CodeBlock({ children }: { children?: ReactNode }) {
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState(false)
  const preRef = useRef<HTMLPreElement>(null)
  const className = isValidElement<{ className?: string }>(children)
    ? children.props.className || ''
    : ''
  const language = /language-([\w-]+)/.exec(className)?.[1] || 'TEXT'
  async function copy() {
    const value = preRef.current?.querySelector('code')?.textContent || ''
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setCopyError(false)
      window.setTimeout(() => setCopied(false), 1800)
    } catch {
      setCopied(false)
      setCopyError(true)
    }
  }
  return (
    <div className="reading-code">
      <div className="reading-code-head">
        <span>{language.toUpperCase()}</span>
        <button type="button" onClick={() => void copy()} aria-label="复制代码">
          {copied ? <Check size={14} /> : <Copy size={14} />}
          {copied ? '已复制' : copyError ? '复制失败' : '复制'}
        </button>
      </div>
      <span className="sr-only" role="status">
        {copyError ? '复制失败，请手动选择代码。' : copied ? '代码已复制。' : ''}
      </span>
      <pre ref={preRef}>{children}</pre>
    </div>
  )
}

export default function ArticleReading({ article }: { article: Article }) {
  const headings = useMemo(() => headingsFromMarkdown(article.body_md), [article.body_md])
  const minutes = Math.max(1, Math.ceil(article.body_md.length / 500))
  const visitor = readUser()
  return (
    <>
      <div className="reading-hero">
        <span className="reading-kicker">
          文章 /{' '}
          {article.category === 'travel'
            ? '游记'
            : article.category === 'essay'
              ? '随笔'
              : article.category === 'record'
                ? '记录'
                : '技术'}
        </span>
        <h1>{article.title}</h1>
        {article.excerpt && <p>{article.excerpt}</p>}
        <div className="reading-meta">
          <span className="reading-author-avatar">
            <img src="/logo.svg" alt="" />
          </span>
          <strong>DevHub</strong>
          <i /> <time>{formatDate(article.published_at)}</time>
          <i />{' '}
          <span>
            <Clock3 size={14} /> 约 {minutes} 分钟阅读
          </span>
        </div>
        {!!article.tags?.length && (
          <div className="reading-tags">
            {article.tags.map((tag) => (
              <span key={tag}># {tag}</span>
            ))}
          </div>
        )}
      </div>
      {article.cover_url && (
        <img className="reading-cover" src={article.cover_url} alt="文章封面" />
      )}
      <div className="reading-layout">
        <div className="reading-main">
          <div className="markdown reading-markdown">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeSlug, [rehypeHighlight, { detect: false }]]}
              components={{ pre: CodeBlock }}
            >
              {article.body_md}
            </ReactMarkdown>
          </div>
          <div className="reading-finish">
            <span>感谢读到这里。欢迎在下方留下想法。</span>
            <Link to="/articles">
              继续看文章 <ArrowRight size={16} />
            </Link>
          </div>
        </div>
        <aside className="reading-aside" aria-label="文章侧边栏">
          {headings.length > 0 && (
            <nav className="reading-toc" aria-label="文章目录">
              <h2>
                <List size={15} /> 本文目录
              </h2>
              {headings.map((heading) => (
                <a
                  className={heading.depth === 3 ? 'sub' : ''}
                  key={heading.id}
                  href={`#${heading.id}`}
                >
                  {heading.text}
                </a>
              ))}
            </nav>
          )}
          <div className="reading-person">
            <span className="reading-person-avatar">
              {visitor?.avatar_url ? (
                <img src={visitor.avatar_url} alt="" />
              ) : (
                <UserRound size={25} />
              )}
            </span>
            <strong>{visitor ? visitor.display_name || visitor.username : '你好，访客'}</strong>
            <p>
              {visitor ? '欢迎回来，可以在文章下交流。' : '登录后可参与文章讨论，也可以申请友链。'}
            </p>
            <Link to={visitor ? '/account' : '/login'}>
              {visitor ? '查看我的账户' : '登录后交流'} <ArrowRight size={14} />
            </Link>
          </div>
        </aside>
      </div>
    </>
  )
}
