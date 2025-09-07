.PHONY: build run test clean docker-build docker-run docker-up docker-down

# 应用信息
APP_NAME=im-gateway
VERSION=latest

# Go 相关设置
GO=go
GOOS=linux
GOARCH=amd64

# 构建目录
BUILD_DIR=build
BINARY_NAME=main

# Docker 相关
DOCKER_IMAGE=$(APP_NAME):$(VERSION)
DOCKER_COMPOSE=docker-compose

# 默认目标
all: build

# 构建
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

# 运行
run:
	@echo "运行 $(APP_NAME)..."
	$(GO) run ./cmd/main.go

# 运行开发环境
dev:
	@echo "启动开发环境..."
	$(DOCKER_COMPOSE) up -d

# 停止开发环境
dev-stop:
	@echo "停止开发环境..."
	$(DOCKER_COMPOSE) down

# 重启开发环境
dev-restart:
	@echo "重启开发环境..."
	$(DOCKER_COMPOSE) restart

# 查看日志
logs:
	@echo "查看日志..."
	$(DOCKER_COMPOSE) logs -f

# 测试
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 测试覆盖率
test-coverage:
	@echo "运行测试覆盖率..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# 格式化代码
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

# 检查代码
lint:
	@echo "检查代码..."
	golangci-lint run

# 清理
clean:
	@echo "清理构建文件..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE) .

# 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	docker run -p 8080:8080 -p 9090:9090 --name $(APP_NAME) $(DOCKER_IMAGE)

# 停止 Docker 容器
docker-stop:
	@echo "停止 Docker 容器..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

# 启动 Docker Compose
docker-up:
	@echo "启动 Docker Compose..."
	$(DOCKER_COMPOSE) up -d

# 停止 Docker Compose
docker-down:
	@echo "停止 Docker Compose..."
	$(DOCKER_COMPOSE) down

# 重启 Docker Compose
docker-restart:
	@echo "重启 Docker Compose..."
	$(DOCKER_COMPOSE) restart

# 查看 Docker 状态
docker-ps:
	@echo "查看 Docker 容器状态..."
	$(DOCKER_COMPOSE) ps

# 查看 Docker 日志
docker-logs:
	@echo "查看 Docker 日志..."
	$(DOCKER_COMPOSE) logs -f

# 安装依赖
deps:
	@echo "安装依赖..."
	$(GO) mod download
	$(GO) mod tidy

# 生成 protobuf 代码
proto:
	@echo "生成 protobuf 代码..."
	make -C ../api proto

# 初始化项目
init: deps proto fmt
	@echo "项目初始化完成"

# 开发环境设置
dev-setup: dev
	@echo "开发环境设置完成"
	@echo "API 地址: http://localhost:8080"
	@echo "WebSocket 地址: ws://localhost:8080/ws"
	@echo "监控地址: http://localhost:9090/metrics"

# 生产构建
build-prod:
	@echo "构建生产版本..."
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-ldflags="-w -s" \
		-a -installsuffix cgo \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		./cmd/main.go

# 发布
release: build-prod docker-build
	@echo "发布完成"

# 帮助
help:
	@echo "可用的命令："
	@echo "  build          - 构建应用"
	@echo "  run            - 运行应用"
	@echo "  dev            - 启动开发环境"
	@echo "  dev-stop       - 停止开发环境"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试覆盖率"
	@echo "  fmt            - 格式化代码"
	@echo "  lint           - 检查代码"
	@echo "  clean          - 清理构建文件"
	@echo "  docker-build   - 构建 Docker 镜像"
	@echo "  docker-up      - 启动 Docker Compose"
	@echo "  docker-down    - 停止 Docker Compose"
	@echo "  deps           - 安装依赖"
	@echo "  proto          - 生成 protobuf 代码"
	@echo "  dev-setup      - 开发环境设置"
	@echo "  build-prod     - 构建生产版本"
	@echo "  release        - 发布"
	@echo "  help           - 显示帮助"
