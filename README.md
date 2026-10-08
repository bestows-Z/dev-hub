# DevHub

一个可自行部署的博客系统：写文章、展示项目、经营数字商品，也为读者交流和创作留出空间。

网站与管理台使用 React、TypeScript、Vite；API 使用 Go、Gin、GORM。PostgreSQL 保存业务数据，MinIO 保存上传文件，Redis 管理共享限流和邮箱验证码，Elasticsearch 提供文章搜索，RabbitMQ 与 MongoDB 处理访问统计。基础服务可通过 Docker Compose 启动。

> 项目持续开发中。下表区分已经可用和正在建设的功能，请按实际状态评估部署需求。

## 功能状态

| 模块 | 当前可用 | 正在建设 |
| --- | --- | --- |
| 文章 | 栏目、标签、全文搜索、Markdown 目录、代码高亮、草稿与发布；作者申请、审核和独立写作台；作者只能管理自己的文章 | 更完整的编辑历史与版本回退 |
| 互动 | 登录评论、评论回复与审核、友链申请与审核；评论及用户展示粗略 IP 属地（需配置本地地区库） | 评论提醒 |
| 项目 | 外链、静态 ZIP 预览、完整前后端 ZIP 上传、后台启停 Docker 预览 | 构建日志与运行监控 |
| 商店 | 登录后下单、库存预留、个人订单、取消订单、后台状态管理 | 邮箱通知与在线支付 |
| 账户 | 邮箱验证码注册与登录、密码登录、验证后更换邮箱、个人资料、头像、独立管理台、作者身份 | GitHub 和 Google 登录 |
| 内容 | 相册上传与管理；空白相册等待站长上传 | 更完整的游记模块 |
| 助手 | 可拖动的小人物、文章检索问答、可选模型生成和浏览器朗读 | 更多动作与语音输入 |

文章、商品、项目的封面由后台上传到本站，页面使用服务器生成的图片地址。主题、粒子和飘雪效果可在左下角切换；系统开启“减少动态效果”时，持续动画会停用。

## 技术结构

~~~mermaid
flowchart LR
  Reader[读者 / 作者 / 管理员] --> Web[React 网站、写作台与管理台]
  Web --> API[Go / Gin API]
  API --> PG[(PostgreSQL)]
  API --> MinIO[(MinIO)]
  API --> Redis[(Redis)]
  API --> ES[(Elasticsearch)]
  API --> MQ[RabbitMQ]
  MQ --> Mongo[(MongoDB)]
  API --> Jobs[(预览任务表)]
  Worker[独立预览工作进程] --> Jobs
  Worker --> Docker[Docker 项目容器]
~~~

PostgreSQL 是业务数据的事实来源。文章索引可从数据库重建；Elasticsearch 不可用时，搜索回退到数据库。访问事件经 RabbitMQ 写入 MongoDB，统计链路故障不会阻断公开页面。项目启动与停止由独立工作进程执行，API 不直接调用 Docker。

~~~text
backend/
  cmd/api/             HTTP API
  cmd/admin/           管理员本地命令
  cmd/preview/         Docker 预览运行器
  cmd/previewworker/   预览任务工作进程
  internal/            按业务拆分的处理器与服务
web/                  React + TypeScript 前端
migrations/           Goose 数据库迁移
deployments/          基础服务 Docker Compose
templates/            完整项目 ZIP 模板源文件
docs/                 架构与项目预览说明
~~~

## 本地启动

### 环境要求

- Go 1.25.9 或更高的 1.25 版本
- Node.js 24、npm
- Docker 与 Docker Compose
- Goose 数据库迁移命令

### 1. 配置环境

~~~sh
git clone https://github.com/bestows-Z/dev-hub.git
cd dev-hub
cp .env.example .env
~~~

编辑 .env，至少更换 PostgreSQL、Redis、MongoDB、RabbitMQ、MinIO 的示例密码，以及不少于 32 字符的 AUTH_JWT_SECRET。修改 PostgreSQL 密码时，也要同步修改 GOOSE_DBSTRING。.env 已被 Git 忽略，不要提交密钥。

### 2. 启动基础服务并迁移数据库

~~~sh
docker compose --env-file .env -f deployments/docker-compose.yml up -d
goose -env .env -dir migrations up
~~~

新环境应由 Goose 从第一条迁移顺序执行。已有环境先备份 PostgreSQL，再运行同一条 up 命令。迁移文件含 Up 与 Down 两段；不要把整个 SQL 文件直接交给 psql 执行。

### 3. 启动 API 与网站

在两个终端分别执行：

~~~sh
cd backend
go run ./cmd/api
~~~

~~~sh
cd web
npm install
npm run dev
~~~

打开 http://localhost:5173。Vite 将 /api 代理到本机 8080 端口。GET /api/v1/health 可检查 API 是否存活。

### 4. 启用项目的后台启动与停止

需要运行完整前后端项目时，在 backend/ 目录额外启动预览工作进程：

~~~sh
mkdir -p bin
go build -o bin/devhub-preview ./cmd/preview
go build -o bin/devhub-previewworker ./cmd/previewworker
./bin/devhub-previewworker
~~~

工作进程需要连接 PostgreSQL、MinIO 和本机 Docker。上线时用进程管理器保持它运行，Docker 权限只授予这个进程。若工作进程没有启动，后台的任务会显示“排队中”。ZIP 格式、端口与网络隔离见[项目预览说明](docs/project-runtime.md)。

## 首次配置管理员

1. 在网站 /register 创建普通账户。
2. 在 backend/ 目录执行 `go run ./cmd/admin promote <用户名>`。
3. 重新登录，打开 /admin。

公开注册不会获得管理权限。后台使用独立页面，不带前台导航和页脚。文章、项目、商品、订单、相册、评论、友链申请及作者申请都从后台管理；列表支持搜索与分页。

普通用户可在 `/account` 申请成为作者。管理员在「作者申请」中审核，通过后该账户获得作者角色。作者打开 `/studio`，可发布、编辑和删除自己的文章；服务端按 `author_id` 限制读写。项目、商品和全站管理仍只有管理员可操作。旧文章在迁移时归属到最早的管理员账户；若当时没有管理员，则保留无归属状态并以站点名显示署名。

### 邮箱验证码与 SMTP

注册必须验证邮箱。读者和管理员默认使用邮箱验证码登录，也可切换密码登录。更换个人邮箱时需验证新邮箱。验证码 10 分钟过期、使用后失效，同一邮箱与用途 60 秒内不能重复发送；连续输错 5 次后作废，每个 IP 与用途每小时最多发送 20 次。Redis 仅保存验证码的 HMAC 摘要。

本地 Compose 包含 Mailpit：SMTP 地址为 `127.0.0.1:1025`，测试收件箱为 [localhost:8025](http://localhost:8025)。本地邮件停留在测试收件箱，不会发往外部邮箱。

上线前按邮件服务商的配置设置 `SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`、`SMTP_TLS_MODE`、`SMTP_USERNAME`、`SMTP_PASSWORD`。`SMTP_TLS_MODE=tls` 表示连接时使用 TLS，`starttls` 表示通过 STARTTLS 升级；明文模式 `none` 仅供本地和测试环境。QQ 邮箱应使用 SMTP 授权码，并将发件人设置为对应邮箱。密钥只保存在服务器环境中。

GitHub 和 Google 登录仍在开发。

## 内容与文件

- **文章：** Markdown 支持 GitHub 风格表格、代码高亮、目录及复制按钮。搜索索引只收录已发布文章。栏目包括技术、游记、随笔、记录。
- **封面：** 文章、项目、商品在编辑时选择 PNG 或 JPEG 图片，保存时上传 MinIO；服务端重新编码并生成 /api/v1/media/... 地址。单张上限 8 MiB。
- **头像：** 账户页上传 PNG/JPEG，服务端重新编码后保存到 MinIO，单张上限 2 MiB。
- **相册：** 初始为空。管理员上传照片并发布后才出现在公开页面。
- **友链：** 登录用户提交申请，管理员审核通过后公开展示。
- **评论：** 登录用户可对已公开的文章留言，也可回复同一篇文章里已审核通过的评论；回复同样需要审核。页面显示回复对象，管理台可查看并审核。

### IP 属地

将自己取得的 GeoLite2-City `.mmdb` 文件放在服务器本地，并把绝对路径填入 `IP_REGION_DB_PATH`。数据库文件不随仓库分发；可从 [MaxMind GeoLite 页面](https://www.maxmind.com/en/geolite-free-ip-geolocation-data)了解注册及下载要求。API 启动时读取地区库，评论保存当次的国家或中国省级地区，账户页展示最近登录地区；不在这两个表保存原始 IP。未配置地区库时，公网地址显示「未知」，本机和内网地址显示「本地网络」。

在反向代理后部署时，把实际可信的代理 IP 或 CIDR 配置到 `HTTP_TRUSTED_PROXIES`，多项用逗号分隔。默认不信任转发头，避免访客伪造 IP 属地。更新地区库后重新启动 API。

文章助手只检索已发布文章。未配置模型时返回相关原文片段和链接；配置 ASSISTANT_API_BASE_URL、ASSISTANT_API_KEY、ASSISTANT_MODEL 后可生成自然语言回答。模型密钥只放在服务端环境变量中。

## 项目预览方式

| 方式 | 提交内容 | 运行位置 |
| --- | --- | --- |
| 外部链接 | 完整的 HTTPS 地址 | 原站点 |
| 静态预览 | 含 index.html 的构建产物 ZIP | 本站读取 MinIO 文件 |
| 完整项目 | 根目录含 frontend/、backend/、docker-compose.yml 的 ZIP | 独立工作进程构建 Docker 容器，本站代理预览 |

完整项目可下载 [ZIP 模板](web/public/downloads/project-runtime-template.zip)。后台上传后点击“启动”，可查看排队、执行、成功或失败状态，也可以点击“停止”。公开访问仍取决于项目的发布状态。需要的基础镜像应提前存在于服务器；构建关闭网络。项目容器使用独立网络、只读文件系统和资源限制，网关端口只绑定到 127.0.0.1。

完整项目 ZIP 含可执行代码。请只运行自己审查过的内容；为不受信任的第三方提供构建服务时，应将工作进程和 Docker 放在专用机器或虚拟机中。不要给公开 API 容器挂载 Docker Socket。

## 商店与订单

访客可以浏览商品；登录后才能创建订单。服务端从登录身份读取用户 ID 和邮箱，不接受客户端冒用的订单归属信息。创建订单时在数据库事务中检查库存、计算总价并预留数量；用户可在个人中心查看订单，取消待付款订单会退回库存。管理员可查看用户名和订单状态。

当前是**线下确认订单**流程，没有在线支付，也不会在网页中收集银行卡信息。SMTP 已用于验证码；订单通知尚未接入。请不要把“待付款”理解为已经收到款项。

## 开发与验证

~~~sh
cd backend && go test ./...
cd web && npm run build
~~~

连接 Docker 服务的集成测试使用显式环境开关，例如 DEVHUB_TEST_REDIS=1、DEVHUB_TEST_MEDIA=1、DEVHUB_TEST_ORDERS=1、DEVHUB_TEST_POSTGRES=1、DEVHUB_TEST_RUNTIME_WORKER=1。测试会创建并清理本地测试数据，不要对生产数据库开启这些开关。

提交更改前运行格式化与构建检查。数据库结构变化请增加可回滚的 Goose 迁移。新增 API 要核对权限、输入大小、错误处理和前端状态。仓库中的 .env.example 只包含示例值。

## 文档与参与

- [系统架构](docs/architecture.md)
- [动态项目预览](docs/project-runtime.md)
- [问题反馈与功能建议](https://github.com/bestows-Z/dev-hub/issues)

欢迎提交问题或改进。建议先描述复现步骤、预期结果与环境，再提交范围清楚的 PR。功能开发请连同迁移、必要的验证和使用说明一起提交。个人任务计划与临时工作笔记已加入 .gitignore，不会作为项目文档上传。
