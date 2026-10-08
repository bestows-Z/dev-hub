import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { api, ApiError } from '../src/api.ts'

const originalFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = originalFetch
})

test('connection errors are readable in Chinese', async () => {
  globalThis.fetch = async () => {
    throw new TypeError('Failed to fetch')
  }
  await assert.rejects(
    api('/articles'),
    (error: unknown) =>
      error instanceof ApiError &&
      error.status === 0 &&
      error.message === '无法连接到网站，请检查网络后重试。',
  )
})

test('intentional cancellation keeps its AbortError', async () => {
  const cancelled = new DOMException('Request was cancelled', 'AbortError')
  globalThis.fetch = async () => {
    throw cancelled
  }
  await assert.rejects(api('/articles'), (error: unknown) => error === cancelled)
})

test('shared conflict codes retain their actual meaning', async () => {
  globalThis.fetch = async () =>
    new Response(JSON.stringify({ code: 40902, message: 'comment is no longer pending' }), {
      status: 409,
    })
  await assert.rejects(api('/admin/comments/1'), { message: '这条评论已经处理过，请刷新列表。' })
  globalThis.fetch = async () =>
    new Response(JSON.stringify({ code: 40902, message: 'email already exists' }), { status: 409 })
  await assert.rejects(api('/auth/register'), { message: '这个邮箱已被使用，请登录或更换邮箱。' })
})
