# GoChat 微服务项目 Makefile
# 提供统一的构建、测试、部署命令

# 项目信息
PROJECT_NAME := gochat
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)
GO_VERSION := $(shell go version | awk '{print $$3}')

# 构建参数
LDFLAGS := -X 'main.Version=$(VERSION)' \
           -X 'main.BuildTime=$(BUILD_TIME)' \
           -X 'main.GoVersion=$(GO_VERSION)'

# 服务列表
SERVICES := im-gateway im-logic im-repo im-task

# Docker 相关
DOCKER_REGISTRY := your-registry.com
DOCKER_TAG := $(VERSION)

# 默认目标
.PHONY: all
all: build

# 帮助信息
.PHONY: help
help:
	@echo "GoChat 微服务项目构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build          构建所有服务"
	@echo "  build-service  构建指定服务 (make build-service SERVICE=im-gateway)"
	@echo "  test           运行所有测试"
	@echo "  test-service   测试指定服务 (make test-service SERVICE=im-gateway)"
	@echo "  proto          生成 protobuf 代码"
	@echo "  clean          清理构建文件"
	@echo "  docker-build   构建 Docker 镜像"
	@echo "  docker-push    推送 Docker 镜像"
	@echo "  lint           代码检查"
	@echo "  fmt            格式化代码"
	@echo "  deps           下载依赖"
	@echo "  dev            启动开发环境"
	@echo ""

# 构建所有服务
.PHONY: build
build:
	@echo "构建所有服务..."
	@for service in $(SERVICES); do \
		echo "构建 $$service..."; \
		CGO_ENABLED=0 GOOS=linux go build -ldflags "$(LDFLAGS)" \
			-o bin/$$service ./$$service/cmd/; \
	done
	@echo "构建完成"

# 构建指定服务
.PHONY: build-service
build-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "请指定服务名称: make build-service SERVICE=im-gateway"; \
		exit 1; \
	fi
	@echo "构建服务 $(SERVICE)..."
	@CGO_ENABLED=0 GOOS=linux go build -ldflags "$(LDFLAGS)" \
		-o bin/$(SERVICE) ./$(SERVICE)/cmd/
	@echo "构建完成: bin/$(SERVICE)"

# 运行测试
.PHONY: test
test:
	@echo "运行所有测试..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "测试完成，覆盖率报告: coverage.html"

# 测试指定服务
.PHONY: test-service
test-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "请指定服务名称: make test-service SERVICE=im-gateway"; \
		exit 1; \
	fi
	@echo "测试服务 $(SERVICE)..."
	@go test -v -race ./$(SERVICE)/...

# 生成 protobuf 代码
.PHONY: proto
proto:
	@echo "生成 protobuf 代码..."
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "错误: 未找到 protoc 命令，请安装 Protocol Buffers"; \
		exit 1; \
	fi
	@if ! command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "安装 protoc-gen-go..."; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; \
	fi
	@if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "安装 protoc-gen-go-grpc..."; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; \
	fi
	@mkdir -p api/gen
	@find api/proto -name "*.proto" -exec protoc \
		--proto_path=api/proto \
		--go_out=api/gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=api/gen \
		--go-grpc_opt=paths=source_relative \
		{} \;
	@echo "protobuf 代码生成完成"

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf bin/
	@rm -rf api/gen/
	@rm -f coverage.out coverage.html
	@go clean -cache
	@echo "清理完成"

# 构建 Docker 镜像
.PHONY: docker-build
docker-build:
	@echo "构建 Docker 镜像..."
	@for service in $(SERVICES); do \
		echo "构建 $$service Docker 镜像..."; \
		docker build -f $$service/Dockerfile \
			-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-$$service:$(DOCKER_TAG) \
			-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)-$$service:latest \
			.; \
	done
	@echo "Docker 镜像构建完成"

# 推送 Docker 镜像
.PHONY: docker-push
docker-push:
	@echo "推送 Docker 镜像..."
	@for service in $(SERVICES); do \
		echo "推送 $$service Docker 镜像..."; \
		docker push $(DOCKER_REGISTRY)/$(PROJECT_NAME)-$$service:$(DOCKER_TAG); \
		docker push $(DOCKER_REGISTRY)/$(PROJECT_NAME)-$$service:latest; \
	done
	@echo "Docker 镜像推送完成"

# 代码检查
.PHONY: lint
lint:
	@echo "运行代码检查..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "安装 golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@golangci-lint run ./...
	@echo "代码检查完成"

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "建议安装 goimports: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi
	@echo "代码格式化完成"

# 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy
	@echo "依赖下载完成"

# 启动开发环境
.PHONY: dev
dev:
	@echo "启动开发环境..."
	@if [ -f docker-compose.dev.yml ]; then \
		docker-compose -f docker-compose.dev.yml up -d; \
		echo "开发环境已启动"; \
	else \
		echo "未找到 docker-compose.dev.yml 文件"; \
	fi

# 停止开发环境
.PHONY: dev-down
dev-down:
	@echo "停止开发环境..."
	@if [ -f docker-compose.dev.yml ]; then \
		docker-compose -f docker-compose.dev.yml down; \
		echo "开发环境已停止"; \
	else \
		echo "未找到 docker-compose.dev.yml 文件"; \
	fi

# 查看服务状态
.PHONY: status
status:
	@echo "服务状态:"
	@for service in $(SERVICES); do \
		if [ -f bin/$$service ]; then \
			echo "  ✓ $$service (已构建)"; \
		else \
			echo "  ✗ $$service (未构建)"; \
		fi; \
	done

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "安装开发工具..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "开发工具安装完成"

# 生成版本信息
.PHONY: version
version:
	@echo "项目版本信息:"
	@echo "  项目名称: $(PROJECT_NAME)"
	@echo "  版本号: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Go 版本: $(GO_VERSION)"