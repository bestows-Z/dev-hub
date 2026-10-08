export interface ArticleSummary {
  id: number
  author_name: string
  author_avatar_url: string
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
  public status: number
  public code?: number
  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

function readableError(message: string, code: number, path: string): string {
  const messages: Record<string, string> = {
    'invalid credentials': '用户名、邮箱或密码不正确。',
    'invalid or expired email code': '验证码不正确或已过期，请重新获取。',
    'email already exists': '这个邮箱已被使用，请登录或更换邮箱。',
    'username already exists': '这个用户名已被使用，请换一个。',
    'authentication required': '请先登录后再操作。',
    'invalid access token': '登录已失效，请重新登录。',
    'administrator access required': '当前账号没有管理员权限。',
    'author access required': '需要作者身份，请在个人中心申请。',
    'media belongs to another user': '你只能管理自己上传的文件。',
    'comment is no longer pending': '这条评论已经处理过，请刷新列表。',
    'application is no longer pending': '这条申请已经处理过，请刷新列表。',
    'an application is already under review': '你已经提交了申请，请等待审核。',
    'wait before requesting another code': '验证码发送过于频繁，请稍后重试。',
    'verification email is temporarily unavailable': '验证码邮件暂时无法发送，请稍后再试。',
    'email verification is unavailable': '邮箱验证暂时不可用，请联系站长。',
    'invalid email or purpose': '请填写有效的邮箱地址。',
    'invalid email login details': '请填写邮箱和 6 位验证码。',
    'invalid login details': '请检查登录信息是否填写完整。',
    'invalid profile details': '请检查邮箱、个人网站和资料长度。',
    'invalid article details': '请检查文章标题、网址标识和正文。',
    'invalid product details': '请检查商品名称、网址标识、价格和库存。',
    'invalid project details': '请检查项目标题、网址标识和预览设置。',
    'comment must contain 3 to 2000 characters': '评论需要 3–2000 个字。',
    'comment is required': '请填写评论内容。',
    'invalid reply target': '回复对象无效，请重新选择。',
    'please wait before commenting again': '评论发送过于频繁，请稍后重试。',
    'too many questions; please try again in a minute': '提问有些频繁，请稍等一分钟。',
    'invalid link application': '请检查网站名称、地址和申请内容。',
    'upload an image before publishing': '请先上传图片，再公开发布。',
    'upload a cover image instead of entering an external URL': '请上传本站封面图片。',
    'cover image does not exist': '封面图片不存在，请重新上传。',
  }
  if (messages[message]) return messages[message]
  const codes: Record<number, string> = {
    40002: '用户名需为 3–32 位字母、数字或下划线。',
    40003: '密码不能超过 72 字节。',
    40103: '验证码不正确或已过期，请重新获取。',
    40301: '你没有权限执行这项操作。',
    40302: '需要作者身份，请在个人中心申请。',
    40400: '这条记录不存在，可能已被删除。',
    40401: '文章不存在或尚未发布。',
    40402: '商品不存在或已经下架。',
    40403: '订单不存在。',
    40404: '回复对象已不可用，请刷新评论。',
    40903: '商品库存不足，请调整购买数量。',
    40904: '订单当前状态不允许这项操作，请刷新后重试。',
    40905: '该商品已有订单，请下架商品。',
    40906: '请先停止项目，再修改网址标识或删除。',
    40907: '请检查项目状态和已上传的 ZIP。',
    40908: '项目任务正在执行，请等待完成。',
    40909: '这张封面正在使用，暂时不能删除。',
    40920: '你已经拥有作者权限。',
    40921: '你的作者申请正在审核中。',
    40922: '这条申请已经处理过。',
    50000: '服务暂时出现问题，请稍后重试。',
    50301: '访问统计暂时不可用。',
  }
  if (codes[code]) return codes[code]
  if (code === 40001 && path.includes('bundle'))
    return '项目压缩包不符合要求，请检查大小和模板目录。'
  if (code === 40001) return '提交的信息不完整或格式不正确，请检查后重试。'
  return message || '请求未完成，请稍后重试。'
}

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, {
      ...options,
      headers: {
        ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
        ...options?.headers,
      },
    })
  } catch (failure) {
    if (failure instanceof DOMException && failure.name === 'AbortError') throw failure
    throw new ApiError('无法连接到网站，请检查网络后重试。', 0)
  }
  let envelope: Envelope<T>
  try {
    envelope = (await response.json()) as Envelope<T>
  } catch (failure) {
    if (failure instanceof DOMException && failure.name === 'AbortError') throw failure
    throw new ApiError('服务暂时无法响应，请稍后重试。', response.status)
  }
  if (!response.ok || envelope.code !== 0)
    throw new ApiError(
      readableError(envelope.message, envelope.code, path),
      response.status,
      envelope.code,
    )
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
