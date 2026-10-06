# 前后端项目预览模板

将 `frontend/` 与 `backend/` 替换成自己的项目。两个 Dockerfile 的运行进程都须以 UID 10001、只读根文件系统在 8080 端口监听 `0.0.0.0`。前端资源使用相对路径，访问后端时使用相对路径 `backend/`。

`docker-compose.yml` 供本地开发复现目录结构；博客运行器只读取其中两个构建目录，不会执行上传的 Compose 配置。部署时基础镜像需提前存在于服务器，构建阶段不能访问网络。

压缩这五项到 ZIP 根目录：`docker-compose.yml`、`README.md`、`frontend/`、`backend/`。在博客后台创建项目后上传 ZIP，在服务器仓库 `backend/` 目录运行 `go run ./cmd/preview deploy-zip --slug 你的项目标识`，确认预览后发布项目。
