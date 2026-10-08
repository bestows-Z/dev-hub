import { useEffect, useState } from 'react'
import { api } from './api'
import { readToken } from './session'
import './email-code.css'

export function EmailCodeField({ email, purpose, value, onChange }: { email: string; purpose: 'register' | 'login' | 'change_email'; value: string; onChange: (value: string) => void }) {
  const [cooldown, setCooldown] = useState(0)
  const [sending, setSending] = useState(false)
  const [notice, setNotice] = useState('')
  useEffect(() => {
    if (cooldown <= 0) return
    const timer = window.setTimeout(() => setCooldown((seconds) => seconds - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [cooldown])
  async function send() {
    if (!email.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) { setNotice('先填写有效邮箱。'); return }
    setSending(true)
    setNotice('')
    try {
      await api(purpose === 'change_email' ? '/auth/me/email-code' : '/auth/email-codes', { method: 'POST', headers: purpose === 'change_email' ? { Authorization: `Bearer ${readToken()}` } : undefined, body: JSON.stringify({ email: email.trim(), purpose }) })
      setCooldown(60)
      setNotice('验证码已发送，10 分钟内有效。')
    } catch (failure) { setNotice(failure instanceof Error ? failure.message : '发送失败，请稍后再试。') }
    finally { setSending(false) }
  }
  return <><div className="auth-code-row"><input id={`auth-code-${purpose}`} inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} minLength={6} required value={value} onChange={(event) => onChange(event.target.value.replace(/\D/g, '').slice(0, 6))} placeholder="输入 6 位验证码" /><button type="button" onClick={() => void send()} disabled={sending || cooldown > 0}>{sending ? '发送中…' : cooldown > 0 ? `${cooldown}s 后重发` : '获取验证码'}</button></div><p className="auth-code-notice" role="status">{notice || '\u00a0'}</p></>
}

