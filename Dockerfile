# 使用官方Go镜像作为构建环境
FROM golang:1.24.4-alpine AS builder

# 构建参数，可以从 docker-compose.yml 传入
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git

# 设置 Go 代理，提高下载速度
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ggcode .

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest

# 安装ca-certificates用于HTTPS请求
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/ggcode .

# 复制静态文件和模板
COPY --from=builder /app/web ./web

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./ggcode"] 