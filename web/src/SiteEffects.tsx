import { useEffect, useState, type CSSProperties } from 'react'
import { Moon, Palette, Snowflake, Sparkles, Sun, X } from 'lucide-react'
import './site-effects.css'

type Appearance = { theme: 'light' | 'night'; snow: boolean; particles: boolean }
const initial: Appearance = { theme: 'light', snow: false, particles: true }

function loadAppearance(): Appearance {
  try {
    const saved = JSON.parse(localStorage.getItem('devhub_appearance') || '{}') as Partial<Appearance>
    return {
      theme: saved.theme === 'night' ? 'night' : 'light',
      snow: saved.snow === true,
      particles: saved.particles !== false,
    }
  } catch {
    return initial
  }
}

export default function SiteEffects() {
  const [appearance, setAppearance] = useState(loadAppearance)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    document.documentElement.dataset.theme = appearance.theme
    try { localStorage.setItem('devhub_appearance', JSON.stringify(appearance)) } catch { /* storage may be unavailable */ }
  }, [appearance])

  useEffect(() => {
    if (!open) return
    const close = (event: KeyboardEvent) => { if (event.key === 'Escape') setOpen(false) }
    window.addEventListener('keydown', close)
    return () => window.removeEventListener('keydown', close)
  }, [open])

  return (
    <>
      {appearance.particles && (
        <div className="site-particles" aria-hidden="true">
          {Array.from({ length: 23 }, (_, index) => (
            <i key={index} style={{ '--x': `${(index * 47 + 17) % 100}%`, '--y': `${(index * 31 + 11) % 100}%`, '--delay': `${(index % 9) * -0.7}s`, '--size': `${2 + index % 4}px` } as CSSProperties} />
          ))}
        </div>
      )}
      {appearance.snow && (
        <div className="site-snow" aria-hidden="true">
          {Array.from({ length: 38 }, (_, index) => (
            <i key={index} style={{ '--x': `${(index * 53 + 9) % 100}%`, '--delay': `${-((index * 7) % 18)}s`, '--duration': `${11 + index % 9}s`, '--size': `${2 + index % 5}px` } as CSSProperties} />
          ))}
        </div>
      )}
      <div className="appearance-dock">
        {open && (
          <div className="appearance-panel" id="appearance-panel">
            <strong>站点外观</strong>
            <button type="button" onClick={() => setAppearance((old) => ({ ...old, theme: old.theme === 'light' ? 'night' : 'light' }))}>
              {appearance.theme === 'light' ? <Moon size={17} /> : <Sun size={17} />}
              {appearance.theme === 'light' ? '切换到夜间' : '切换到白天'}
            </button>
            <button type="button" aria-pressed={appearance.particles} onClick={() => setAppearance((old) => ({ ...old, particles: !old.particles }))}>
              <Sparkles size={17} /> 粒子效果 <span>{appearance.particles ? '开' : '关'}</span>
            </button>
            <button type="button" aria-pressed={appearance.snow} onClick={() => setAppearance((old) => ({ ...old, snow: !old.snow }))}>
              <Snowflake size={17} /> 飘雪 <span>{appearance.snow ? '开' : '关'}</span>
            </button>
            <small>减少动态效果的系统设置会优先生效。</small>
          </div>
        )}
        <button className="appearance-toggle" type="button" aria-label={open ? '关闭外观设置' : '打开外观设置'} aria-expanded={open} aria-controls="appearance-panel" onClick={() => setOpen(!open)}>
          {open ? <X size={20} /> : <Palette size={20} />}
        </button>
      </div>
    </>
  )
}
