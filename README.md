# DevHub

一个个人博客、作品展示与数字商店。后端沿用仓库现有的 Go/Gin，数据存 PostgreSQL；前端使用 React、TypeScript 与 Vite。接口契约位于 [`api/openapi.yaml`](api/openapi.yaml)，可直接导入 Apifox。

## 状态

项目按功能逐步交付。已包含文章、友链、商品与订单、项目目录、静态与动态项目预览、文章问答和管理页。交付边界见 [`docs/architecture.md`](docs/architecture.md)。

## 本地准备

1. 安装 Go 1.25、Node.js 24、Docker Desktop。
2. 复制 `.env.example` 为 `.env`，设置非示例密码；不要提交 `.env`。
3. 在仓库根目录执行 `docker compose --env-file .env -f deployments/docker-compose.yml up -d postgres redis minio elasticsearch mongodb rabbitmq`。API 启动时会检查 Redis 连接；助手问答按访客 IP 共享每分钟 20 次额度。文章全文搜索使用 Elasticsearch，启动时从 PostgreSQL 重建索引；不可用时自动使用数据库搜索。后台访问统计通过 RabbitMQ 异步写入 MongoDB；两者不可用时公开页面仍可访问。部署在 Docker 网络内时将 `REDIS_ADDR` 设为 `redis:6379`、`ELASTICSEARCH_URL` 设为 `http://elasticsearch:9200`、`MONGO_ADDR` 设为 `mongodb:27017`、`RABBITMQ_ADDR` 设为 `rabbitmq:5672`。
4. 执行数据库迁移（SQL 文件位于 `migrations/`，按文件名顺序执行）。
5. 在 `backend/` 执行 `go run ./cmd/api`；在 `web/` 执行 `npm install && npm run dev`。

首次使用管理功能：先在 `/register` 创建自己的账户，再在 `backend/` 执行 `go run ./cmd/admin promote <用户名>`。随后在 `/login` 登录，进入 `/admin`。管理员权限只通过本地命令授予，公开注册不会成为管理员。

前台提供 `/login`、`/register` 和 `/account` 页面。注册后自动登录；会话令牌保存在当前浏览器标签页的 `sessionStorage`，关闭标签页后需要重新登录。管理员通过同一账户进入 `/admin`。

账户页可以修改显示名称、邮箱、个人简介和网站链接，也可以上传或移除头像。头像仅接收不超过 2 MiB 的 PNG/JPEG；服务端去除原图元数据后保存到 MinIO，并在 API 启动时检查存储桶。升级旧数据库时执行 `migrations/20261006003000_user_profiles.sql` 的 Up 段。

页面左下角可切换白天或夜间主题、粒子与飘雪效果；选择会保存在当前浏览器。系统开启“减少动态效果”时，粒子和飘雪会自动停用。

页面右下角的博客小助手可以用鼠标或触屏拖动。位置保存在浏览器本地；打开对话后点击“归位”可恢复默认位置。

文章管理支持技术、游记、随笔、记录四个栏目和自定义标签。前台 `/articles` 可搜索标题、摘要与正文，并按栏目或标签筛选；`/travel`、`/essays`、`/records` 目前不会填充虚构内容，站长发布对应文章后自动展示。更新旧数据库时需执行 `migrations/20261006001000_article_categories.sql` 的 Up 段。

文章正文支持 GitHub 风格 Markdown、语法高亮、代码复制、自动目录和移动端阅读布局。目录与正文由同一套 Markdown 解析规则生成锚点，重复标题也能准确跳转。

登录读者可以在文章下提交评论、在友链页申请交换链接。内容先进入审核队列；站长在 `/admin` 的“评论审核”和“友链申请”中通过或驳回。通过的评论才公开，通过的友链申请会自动创建公开友链。请一并执行 `migrations/20261006002000_reader_interactions.sql` 的 Up 段。

管理台 `/admin` 提供文章、项目、商品、友链、订单与审核工作区。首页统计来自实时管理接口；各类记录使用表格、服务端搜索与分页，文章编辑器可切换 Markdown 预览。审核栏可查看待审、已通过和已驳回记录。

开发时前端由 Vite 代理 `/api` 到 `http://localhost:8080`。API 健康检查：`GET /api/v1/health`。

文章助手默认从已发布文章中检索并返回相关段落与原文链接。若要让它生成自然语言回答，在 `.env` 中同时设置 `ASSISTANT_API_BASE_URL`、`ASSISTANT_API_KEY`、`ASSISTANT_MODEL`，服务端会调用兼容 Chat Completions 的模型接口。密钥只保存在服务端；模型不可用时仍返回文章摘录。

右下角的小人物可以打开对话，回答旁的“朗读”使用浏览器内置语音合成；语音是否可用取决于访客浏览器，不会自动播放。

静态项目预览：先在管理页创建项目，再上传包含 `index.html` 的构建产物 ZIP（可直接压缩 `dist/` 目录），最后发布项目。前端构建时请使用相对资源路径，例如 Vite 的 `base: './'`，否则从子路径打开时资源会指向站点根目录。压缩包上限 20 MiB；只接收网页资源文件，拒绝隐藏文件、软链接和越界路径。MinIO 存储使用 `.env` 中的 `MINIO_*` 配置；正式部署时请改用独立服务账号。

前后端动态预览：后台可上传完整项目 ZIP，目录结构见可下载的 [项目模板](web/public/downloads/project-runtime-template.zip)；站长在服务器执行 `go run ./cmd/preview deploy-zip --slug 项目标识`，运行器从 MinIO 读取代码、离线构建 Docker 镜像并启动前后端。也可使用预先构建的镜像执行 `deploy`。已发布的站内项目会在首页和项目页的隔离窗口中预览，外部项目链接仍可单独打开。应用容器各自运行在内部网络，通过只绑定 `127.0.0.1` 的网关连接主 API。操作步骤、镜像要求与限制见 [`docs/project-runtime.md`](docs/project-runtime.md)。普通用户上传 ZIP 不会触发容器执行。

## 接口文档

在 Apifox 选择“项目设置 → 导入数据 → OpenAPI/Swagger”，导入 `api/openapi.yaml`。每次接口行为变更应与该文件一起提交。

## 安全边界

上传的项目代码属于不可信内容。动态镜像只能由能操作宿主 Docker 的站长通过本地命令发布；预览容器不挂载 Docker Socket、数据库凭证或宿主目录，并限制 CPU、内存和进程数。Docker 构建本身仍会处理项目代码，站长应先审查 ZIP，面向不可信第三方代码时建议部署到专用机器。支付采用线下确认订单的首版流程，不在网站收集银行卡信息。
