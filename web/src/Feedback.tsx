import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useLocation } from 'react-router-dom'
import * as AlertDialog from '@radix-ui/react-alert-dialog'
import { AlertTriangle, ArrowRight } from 'lucide-react'
import { Toaster } from 'sonner'
import './feedback.css'

type Confirmation = { title: string; description: string; confirmLabel?: string; danger?: boolean }
type Confirm = (options: Confirmation) => Promise<boolean>
const ConfirmationContext = createContext<Confirm | null>(null)

export function useConfirm() {
  const confirm = useContext(ConfirmationContext)
  if (!confirm) throw new Error('useConfirm requires FeedbackProvider')
  return confirm
}

export function FeedbackProvider({ children }: { children: ReactNode }) {
  const [request, setRequest] = useState<Confirmation | null>(null)
  const [theme, setTheme] = useState<'light' | 'dark'>(() =>
    document.documentElement.dataset.theme === 'night' ? 'dark' : 'light',
  )
  const pending = useRef<((value: boolean) => void) | null>(null)
  const returnFocus = useRef<HTMLElement | null>(null)
  const location = useLocation()
  const finish = useCallback((result: boolean) => {
    pending.current?.(result)
    pending.current = null
    setRequest(null)
  }, [])
  const confirm = useCallback<Confirm>((options) => {
    pending.current?.(false)
    returnFocus.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null
    setRequest(options)
    return new Promise<boolean>((resolve) => {
      pending.current = resolve
    })
  }, [])
  useEffect(() => {
    finish(false)
  }, [location.pathname, finish])
  useEffect(
    () => () => {
      pending.current?.(false)
    },
    [],
  )
  useEffect(() => {
    const update = () =>
      setTheme(document.documentElement.dataset.theme === 'night' ? 'dark' : 'light')
    const observer = new MutationObserver(update)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    })
    update()
    return () => observer.disconnect()
  }, [])
  return (
    <ConfirmationContext.Provider value={confirm}>
      {children}
      <Toaster
        theme={theme}
        position="top-right"
        offset={80}
        mobileOffset={{ top: 76, left: 16, right: 16 }}
        closeButton
        richColors
        visibleToasts={3}
        duration={4500}
        toastOptions={{ closeButtonAriaLabel: '关闭通知', className: 'site-toast' }}
      />
      <AlertDialog.Root
        open={request !== null}
        onOpenChange={(open) => {
          if (!open) finish(false)
        }}
      >
        <AlertDialog.Portal>
          <AlertDialog.Overlay className="feedback-overlay" />
          <AlertDialog.Content
            className="feedback-dialog"
            onCloseAutoFocus={(event) => {
              event.preventDefault()
              const target = returnFocus.current
              if (target?.isConnected) target.focus({ preventScroll: true })
              if (!target || document.activeElement !== target)
                document
                  .querySelector<HTMLElement>('main[tabindex="-1"]')
                  ?.focus({ preventScroll: true })
            }}
          >
            <div
              className={`feedback-symbol${request?.danger ? ' danger' : ''}`}
              aria-hidden="true"
            >
              <AlertTriangle size={23} />
            </div>
            <AlertDialog.Title className="feedback-title">{request?.title}</AlertDialog.Title>
            <AlertDialog.Description className="feedback-description">
              {request?.description}
            </AlertDialog.Description>
            <div className="feedback-actions">
              <AlertDialog.Cancel asChild>
                <button type="button" className="feedback-cancel" onClick={() => finish(false)}>
                  取消
                </button>
              </AlertDialog.Cancel>
              <AlertDialog.Action asChild>
                <button
                  type="button"
                  className={`feedback-confirm${request?.danger ? ' danger' : ''}`}
                  onClick={() => finish(true)}
                >
                  {request?.confirmLabel || '确认'}
                  <ArrowRight size={16} />
                </button>
              </AlertDialog.Action>
            </div>
          </AlertDialog.Content>
        </AlertDialog.Portal>
      </AlertDialog.Root>
    </ConfirmationContext.Provider>
  )
}
