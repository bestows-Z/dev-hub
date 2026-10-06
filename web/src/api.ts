export interface ArticleSummary {
  id: number
  slug: string
  title: string
  excerpt: string
  cover_url: string
  category: 'tech' | 'travel' | 'essay' | 'record'
  tags: string[]
  published_at: string | null
}
export interface Article extends ArticleSummary {
  body_md: string
  created_at: string
}
export interface FriendLink {
  id: number
  name: string
  url: string
  avatar_url: string
  description: string
}
export interface GalleryItem {
  id: number
  title: string
  description: string
  location: string
  image_url: string
  taken_at: string | null
  created_at: string
}
export interface Product {
  id: number
  slug: string
  name: string
  description: string
  price_cents: number
  cover_url: string
  stock: number
}
export interface Project {
  id: number
  slug: string
  title: string
  description: string
  cover_url: string
  tags: string[]
  preview_url: string
  backend_url: string
  runtime_status: string
  source_url: string
}
export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers: {
      ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      ...options?.headers,
    },
  })
  let envelope: Envelope<T>
  try {
    envelope = (await response.json()) as Envelope<T>
  } catch {
    throw new ApiError('服务暂时无法响应，请稍后重试。', response.status)
  }
  if (!response.ok || envelope.code !== 0)
    throw new ApiError(envelope.message || '请求失败', response.status, envelope.code)
  return envelope.data
}

export function formatDate(value: string | null): string {
  if (!value) return '近期发布'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(value))
}

export function formatPrice(cents: number): string {
  return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY' }).format(cents / 100)
}
