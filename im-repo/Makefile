# im-repo 服务 Makefile

# 变量定义
APP_NAME = im-repo
VERSION ?= latest
BUILD_DIR = build
CMD_DIR = cmd/server
MAIN_FILE = $(CMD_DIR)/main.go

# Go 相关变量
GO = go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Docker 相关变量
DOCKER_IMAGE = $(APP_NAME):$(VERSION)
DOCKER_REGISTRY ?= localhost:5000

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME).bin \
		$(MAIN_FILE)
	@echo "构建完成: $(BUILD_DIR)/$(APP_NAME)"

# 运行
.PHONY: run
run:
	@echo "运行 $(APP_NAME)..."
	$(GO) run $(MAIN_FILE)

# 测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "运行测试覆盖率..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告生成: coverage.html"

# 代码检查
.PHONY: lint
lint:
	@echo "运行代码检查..."
	golangci-lint run

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

# 整理依赖
.PHONY: tidy
tidy:
	@echo "整理依赖..."
	$(GO) mod tidy

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker 构建
.PHONY: docker-build
docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE) .

# Docker 运行
.PHONY: docker-run
docker-run:
	@echo "运行 Docker 容器..."
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE)

# Docker 推送
.PHONY: docker-push
docker-push:
	@echo "推送 Docker 镜像..."
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "安装开发工具..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 生成 protobuf
.PHONY: proto
proto:
	@echo "生成 protobuf 代码..."
	@if [ -d "../api/proto" ]; then \
		cd ../api && make generate; \
	else \
		echo "protobuf 目录不存在，跳过生成"; \
	fi

# 开发环境设置
.PHONY: dev-setup
dev-setup: install-tools tidy
	@echo "开发环境设置完成"

# 帮助
.PHONY: help
help:
	@echo "可用的 make 目标:"
	@echo "  build          - 构建应用程序"
	@echo "  run            - 运行应用程序"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  lint           - 运行代码检查"
	@echo "  fmt            - 格式化代码"
	@echo "  tidy           - 整理依赖"
	@echo "  clean          - 清理构建文件"
	@echo "  docker-build   - 构建 Docker 镜像"
	@echo "  docker-run     - 运行 Docker 容器"
	@echo "  docker-push    - 推送 Docker 镜像"
	@echo "  install-tools  - 安装开发工具"
	@echo "  proto          - 生成 protobuf 代码"
	@echo "  dev-setup      - 设置开发环境"
	@echo "  help           - 显示此帮助信息"