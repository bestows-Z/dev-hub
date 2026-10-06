# 前后端项目的动态预览

动态预览由后台发起，独立工作进程在服务器上构建和启停容器。访客只会看到已发布项目的前端与 API 入口。API 进程无需 Docker Socket；只有工作进程需要访问 Docker。

## 上传完整项目 ZIP

从后台项目管理下载 [完整项目模板](../web/public/downloads/project-runtime-template.zip)。ZIP 根目录固定为：

```text
docker-compose.yml
frontend/Dockerfile
frontend/…
backend/Dockerfile
backend/…
README.md（可选）
```

Compose 文件必须恰好包含 `frontend` 和 `backend` 两个服务，构建目录分别为 `./frontend`、`./backend`。它方便在本机复现；运行器只读取构建目录，不执行上传的 Compose 指令。ZIP 最大 50 MiB，解压总量最大 250 MiB，拒绝越界路径、符号链接和其他特殊文件。前后端服务应监听容器内 `0.0.0.0:8080`，以 UID 10001 在只读根文件系统运行。基础镜像需事先存在于服务器；构建时关闭网络。

在服务器的 `backend/` 目录构建并启动工作进程：

```sh
go build -o bin/devhub-preview ./cmd/preview
go build -o bin/devhub-previewworker ./cmd/previewworker
./bin/devhub-previewworker
```

完成 `20261006007000_project_runtime_jobs.sql` 迁移后，在管理台上传完整 ZIP，点击项目行的「启动」或「停止」。后台会显示排队、执行、成功和失败状态。工作进程从 MinIO 读取 ZIP，解压到临时目录，分别用 Docker 构建前后端镜像，再将其放入项目独立网络中运行。构建失败时，已有预览保持运行。在后台更改外部“预览地址”也可直接使用外链，不需要上传 ZIP。

工作进程应由服务器的进程管理器保持运行，并在部署时单独配置 Docker 权限。`PREVIEW_RUNNER_BIN` 可指定运行器二进制文件路径，默认 `./bin/devhub-preview`。API 与前端不需要 Docker 权限。若只启动 API、没有启动工作进程，任务会停留在「排队中」。

## 准备

1. 启动 PostgreSQL、API 与 Docker，并执行包括 `migrations/20261005230000_project_runtime.sql` 在内的数据库迁移。
2. 在管理页创建项目，记下项目的 `slug` 和数字 ID。可以先保持草稿，在本机检查网关后再发布。
3. 自行审查项目代码。可使用上述完整 ZIP 流程，也可分别构建前端和后端镜像。命令只接受本机已有的基础镜像；不会从远端拉取项目镜像。还需要本机已有 `node:20-alpine`，用于受限的内部网关。
4. 两个镜像都要能以 UID `10001`、只读根文件系统启动，默认服务端口为 `8080`。如端口不同，在命令中指定。容器启动时不能依赖宿主目录、Docker Socket、数据库凭证或外网下载。

例如，先在各自项目目录构建镜像：

```sh
docker build -t my-site:preview ./my-site
docker build -t my-api:preview ./my-api
```

仍可在本仓库 `backend/` 目录使用命令行方式部署已有镜像：

```sh
go run ./cmd/preview deploy --slug my-project \
  --frontend-image my-site:preview --backend-image my-api:preview \
  --frontend-port 8080 --backend-port 8080
```

发布成功后，前端入口是 `/api/v1/project-runtimes/my-project/`，后端入口是 `/api/v1/project-runtimes/my-project/backend/`。草稿阶段，这两个公开入口会返回 404；站长可以先用 `docker port devhub-preview-<项目数字ID>-gateway 8080/tcp` 查到本机端口，再从本机打开 `http://127.0.0.1:<端口>/` 和 `/backend/` 检查服务。确认后，在管理页将项目状态改为“已发布”，访客才能访问。停止运行：

```sh
go run ./cmd/preview stop --slug my-project
```

如项目此前上传过静态 ZIP，停止动态预览后，公开预览会恢复到静态版本。删除项目之前必须先停止运行。

## 子路径与网络要求

前端从子路径打开。静态资源使用相对路径，例如 Vite 的 `base: './'`；前端请求自己的后端时也应使用项目根目录下的 `backend/` 路径。后端会收到去掉 `/backend` 前缀的请求路径。项目服务返回以 `/` 开头的本地跳转时，公开代理会补上项目路径前缀。首版转发常规 HTTP 请求，不支持 WebSocket 升级。

每个项目有独立的内部 Docker 网络。前后端容器没有宿主端口，只能通过网关访问；网关只向宿主 `127.0.0.1` 发布随机端口，公共 API 按项目状态转发请求。应用容器限制为每个 256 MiB 内存、0.5 CPU、128 个进程，并去掉 Linux capabilities、启用只读根文件系统和 `no-new-privileges`。网关也有资源限制。这个机制适合站长审核后的项目演示，不应当作为承接任意第三方代码的公共构建平台；需要更强隔离时使用专用机器或虚拟机。

公开预览的 HTML 带 CSP sandbox，且不会把博客 Cookie 转给项目服务。站长应避免在项目镜像里打包秘密。项目如果需要外部数据库或写入持久数据，应单独设计隔离的数据服务；当前运行器不会提供这些资源。

## 排查

- `image ... is not built locally`：先构建相应镜像，并确认 `node:20-alpine` 已在本机。
- `runtime ... did not become healthy`：检查镜像是否以 UID `10001` 正常启动，服务是否监听 `0.0.0.0` 与指定端口，并查看项目容器日志。
- 页面资源 404：检查前端构建的相对资源路径。
- 访客收到 404：项目须处于“已发布”且运行状态为 `running`。
- 访客收到 502：项目服务可能退出；检查 Docker 容器状态和日志，然后重新部署。

启动和停止接口仅接受管理员登录令牌，普通用户和公开页面不能提交构建任务。工作进程领取任务后调用本地运行器；因此仅在站长上传并确认的项目上使用此功能。
