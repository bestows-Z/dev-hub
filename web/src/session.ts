export type AuthUser = {
  id: number
  username: string
  email: string
  display_name: string
  bio: string
  website_url: string
  avatar_url: string
  ip_region: string
  role: number
  status: number
  created_at: string
}

export type LoginResult = {
  access_token: string
  token_type: string
  expires_in: number
  user: AuthUser
}

const tokenKey = 'devhub_access_token'
const userKey = 'devhub_user'
export const authChanged = 'devhub-auth-change'

export function readToken(): string {
  return sessionStorage.getItem(tokenKey) || sessionStorage.getItem('devhub_admin_token') || ''
}

export function readUser(): AuthUser | null {
  try {
    const raw = sessionStorage.getItem(userKey)
    return raw ? (JSON.parse(raw) as AuthUser) : null
  } catch {
    return null
  }
}

export function saveSession(result: LoginResult): void {
  sessionStorage.setItem(tokenKey, result.access_token)
  refreshSessionUser(result.user)
}

export function refreshSessionUser(user: AuthUser): void {
  const token = readToken()
  if (token) sessionStorage.setItem(tokenKey, token)
  sessionStorage.setItem(userKey, JSON.stringify(user))
  if (user.role === 1 && token) sessionStorage.setItem('devhub_admin_token', token)
  else sessionStorage.removeItem('devhub_admin_token')
  window.dispatchEvent(new Event(authChanged))
}

export function clearSession(): void {
  sessionStorage.removeItem(tokenKey)
  sessionStorage.removeItem(userKey)
  sessionStorage.removeItem('devhub_admin_token')
  window.dispatchEvent(new Event(authChanged))
}
