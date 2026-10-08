# 第三方登录配置

DevHub 支持 GitHub 和 Google 登录，以及在个人中心绑定、解除绑定。每个平台可单独启用。没有配置密钥时，对应按钮不可点击，邮箱登录仍可使用。

## 本地与生产地址

本地开发由 Vite 转发 `/api`，推荐统一使用 `localhost`：

```dotenv
AUTH_PUBLIC_URL=http://localhost:5173
WEB_PUBLIC_URL=http://localhost:5173
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

注册以下回调地址，注意路径大小写和末尾不带斜杠：

| 平台 | 本地回调 |
| --- | --- |
| GitHub | `http://localhost:5173/api/v1/auth/oauth/github/callback` |
| Google | `http://localhost:5173/api/v1/auth/oauth/google/callback` |

生产环境将两个公开地址改为实际站点，例如 `https://blog.example.com`，并将平台回调地址中的本地地址同步替换。生产环境必须使用 HTTPS。两个地址必须采用相同主机名和协议；推荐同一个站点入口，通过反向代理将 `/api` 转发到 API 服务。不要混用 `localhost` 与 `127.0.0.1`，否则浏览器不能将状态 Cookie 发送到回调。

## 申请 GitHub OAuth App

在 GitHub 的 Settings → Developer settings → OAuth Apps 创建应用。填写应用名称、站点首页和上面的 GitHub 回调地址。取得 Client ID，生成 Client Secret，分别配置到 `GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`。步骤见 [GitHub 创建 OAuth App 文档](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app)。

应用请求 `read:user user:email`，用于读取身份和已验证的主邮箱。用户需在 GitHub 验证主邮箱。授权过程使用 PKCE S256，配置依据 [GitHub 授权流程文档](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)。

## 申请 Google OAuth 客户端

在 Google Cloud Console 创建或选择项目，配置 Google Auth Platform 的应用名称、支持邮箱、受众和联系信息。开发阶段将自己的账号加入测试用户。创建类型为「Web application」的 OAuth 客户端，将上面的 Google 回调地址加入 Authorized redirect URIs，取得 Client ID 和 Client Secret，配置到 `GOOGLE_CLIENT_ID`、`GOOGLE_CLIENT_SECRET`。公开上线时按控制台要求完成应用发布和域名设置。具体配置见 [Google 服务端授权文档](https://developers.google.com/identity/protocols/oauth2/web-server)。

应用只请求 `openid email profile`。服务端调用 Google UserInfo 取得稳定身份标识和已验证邮箱，不保存 Google 访问令牌。身份信息依据 [Google OpenID Connect 文档](https://developers.google.com/identity/openid-connect/openid-connect)。

## 启用与账号规则

1. 将密钥放到服务器环境变量或忽略提交的 `.env` 文件，同一平台的 ID 与 Secret 必须同时填写。
2. 执行数据库迁移，包含 `20261008001000_oauth_identities.sql`。
3. 重启 API，确认登录页对应按钮已开放。
4. 用测试账号完成授权，检查个人中心显示绑定；退出后再次登录，确认回到同一个账户。

首次使用第三方账号登录会创建普通用户。若邮箱已经存在，先用邮箱登录，再在个人中心主动绑定。绑定操作要求当前账号已经登录，已属于另一用户的第三方身份不能被覆盖。解绑后仍可使用邮箱验证码登录；更换第三方平台邮箱不会改变本站邮箱。

回调状态有效期为 10 分钟，交换登录凭证有效期为 60 秒且只能使用一次。API 默认不记录 OAuth 回调的完整请求路径；生产反向代理也应避免记录该路径的查询参数。未完成真实平台联调前，请先在自己的测试环境验证回调、邮件投递和账号绑定，再开放给用户。

## 本地验证

```bash
cd backend
go test ./internal/auth
DEVHUB_TEST_OAUTH=1 go test ./internal/auth -run TestOAuthFlowIntegration -v
```

集成验证需要本地 PostgreSQL、Redis 和最新迁移；外部平台用模拟服务替代。测试事务回滚创建的账户与绑定，不需要提升管理权限。
