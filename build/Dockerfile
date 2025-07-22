FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gochat.bin .

# 使用distroless镜像作为最终基础镜像
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# 从builder阶段复制编译好的二进制文件
COPY --from=builder /app/gochat.bin /app/
COPY --from=builder /app/site /app/site

# 使用非root用户运行
USER nonroot:nonroot

# 声明暴露的端口
EXPOSE 8080 8081 8082

# 设置入口点
ENTRYPOINT ["/app/gochat.bin"]