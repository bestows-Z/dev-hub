# =========================
# Build Stage
# =========================

FROM golang:1.26-alpine AS builder

# MinIO 构建时需要 git 拉取依赖。
RUN apk add --no-cache \
    git \
    ca-certificates

# MinIO 本身可以构建为静态 Go 程序。
ENV CGO_ENABLED=0

# 官方目前推荐 Community Edition 从源码安装。
RUN go install github.com/minio/minio@latest


# =========================
# Runtime Stage
# =========================

FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    tzdata

# 将编译好的 MinIO 二进制复制到运行镜像
COPY --from=builder /go/bin/minio /usr/local/bin/minio

# MinIO 数据目录
RUN mkdir -p /data

EXPOSE 9000 9001

VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/minio"]

CMD ["server", "/data", "--console-address", ":9001"]