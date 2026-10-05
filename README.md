# DevHub

一个个人博客、作品展示与数字商店。后端沿用仓库现有的 Go/Gin，数据存 PostgreSQL；前端使用 React、TypeScript 与 Vite。接口契约位于 [`api/openapi.yaml`](api/openapi.yaml)，可直接导入 Apifox。

## 状态

项目按功能逐步交付。当前仓库已有用户注册基础代码，完整模块清单与交付边界见 [`docs/architecture.md`](docs/architecture.md)。

## 本地准备

1. 安装 Go 1.25、Node.js 24、Docker Desktop。
2. 复制 `.env.example` 为 `.env`，设置非示例密码；不要提交 `.env`。
3. 在仓库根目录执行 `docker compose --env-file .env -f deployments/docker-compose.yml up -d postgres redis minio`。
4. 执行数据库迁移（SQL 文件位于 `migrations/`，按文件名顺序执行）。
5. 在 `backend/` 执行 `go run ./cmd/api`；在 `web/` 执行 `npm install && npm run dev`。

首次使用管理功能：先调用注册接口创建自己的账户，再在 `backend/` 执行 `go run ./cmd/admin promote <用户名>`。随后登录获取访问令牌，进入 `/admin`。管理员权限只通过本地命令授予，公开注册不会成为管理员。

开发时前端由 Vite 代理 `/api` 到 `http://localhost:8080`。API 健康检查：`GET /api/v1/health`。

文章助手默认从已发布文章中检索并返回相关段落与原文链接。若要让它生成自然语言回答，在 `.env` 中同时设置 `ASSISTANT_API_BASE_URL`、`ASSISTANT_API_KEY`、`ASSISTANT_MODEL`，服务端会调用兼容 Chat Completions 的模型接口。密钥只保存在服务端；模型不可用时仍返回文章摘录。

## 接口文档

在 Apifox 选择“项目设置 → 导入数据 → OpenAPI/Swagger”，导入 `api/openapi.yaml`。每次接口行为变更应与该文件一起提交。

## 安全边界

上传的项目代码属于不可信内容。公共项目先提供静态预览；前后端动态运行必须由管理员在独立、无宿主 Docker Socket、受限资源的运行环境中发布。不得让普通上传接口直接执行代码。支付采用线下确认订单的首版流程，不在网站收集银行卡信息。
