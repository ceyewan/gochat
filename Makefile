# Makefile for the GoChat project

# 定义 Go 编译器和相关工具
GO := go
GOFMT := gofmt
GOLINT := golangci-lint
BUF := buf

# 定义项目源码路径
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")
GO_PACKAGES := ./...

# ==============================================================================
# 常用开发命令
# ==============================================================================

.PHONY: all
all: fmt lint test
	@echo "✅ All checks passed!"

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Run all checks: fmt, lint, test"
	@echo "  proto        - Generate protobuf code from api definitions"
	@echo "  fmt          - Format all Go source files"
	@echo "  lint         - Run static analysis using golangci-lint"
	@echo "  test         - Run all unit tests with race detector enabled"
	@echo "  tidy         - Tidy go.mod and go.sum files"
	@echo "  clean        - Clean up generated files and build artifacts"
	@echo ""
	@echo "Deployment Targets:"
	@echo "  infra-up     - Start core infrastructure services only"
	@echo "  infra-down   - Stop core infrastructure services only"
	@echo "  infra-up-all - Start all infrastructure services (core + monitoring + admin)"
	@echo "  infra-down-all - Stop all infrastructure services"
	@echo "  monitoring-up - Start core + monitoring services"
	@echo "  monitoring-down - Stop core + monitoring services"
	@echo "  admin-up     - Start core + monitoring + admin services"
	@echo "  admin-down   - Stop core + monitoring + admin services"
	@echo "  app-up       - Start all application services"
	@echo "  app-down     - Stop all application services"
	@echo "  config-sync  - Sync all configurations to etcd"
	@echo "  config-sync-dev - Sync dev configurations to etcd"

# ==============================================================================
# 基础设施部署命令
# ==============================================================================

.PHONY: infra-up
infra-up:
	@echo "🚀 Starting core infrastructure services only (etcd, kafka, mysql, redis)..."
	@docker compose -f deployment/infrastructure/docker-compose.yml up -d

.PHONY: infra-down
infra-down:
	@echo "🛑 Stopping core infrastructure services only..."
	@docker compose -f deployment/infrastructure/docker-compose.yml down

.PHONY: infra-up-all
infra-up-all:
	@echo "🚀 Starting all infrastructure services (core + monitoring + admin)..."
	@./deployment/scripts/start-infra.sh all

.PHONY: infra-down-all
infra-down-all:
	@echo "🛑 Stopping all infrastructure services..."
	@./deployment/scripts/cleanup.sh infra

.PHONY: monitoring-up
monitoring-up:
	@echo "🚀 Starting core + monitoring services..."
	@./deployment/scripts/start-infra.sh monitoring

.PHONY: monitoring-down
monitoring-down:
	@echo "🛑 Stopping core + monitoring services..."
	@docker compose -f deployment/infrastructure/docker-compose.yml -f deployment/infrastructure/docker-compose.monitoring.yml down

.PHONY: admin-up
admin-up:
	@echo "🚀 Starting core + monitoring + admin services..."
	@./deployment/scripts/start-infra.sh admin

.PHONY: admin-down
admin-down:
	@echo "🛑 Stopping core + monitoring + admin services..."
	@docker compose -f deployment/infrastructure/docker-compose.yml -f deployment/infrastructure/docker-compose.monitoring.yml -f deployment/infrastructure/docker-compose.admin.yml down

# ==============================================================================
# 应用服务部署命令
# ==============================================================================

.PHONY: app-up
app-up:
	@echo "🚀 Starting applications via script..."
	@./deployment/scripts/start-apps.sh

.PHONY: app-down
app-down:
	@echo "🛑 Stopping applications..."
	@docker compose -f deployment/applications/docker-compose.yml down

.PHONY: config-sync
config-sync:
	@echo "🔄 Syncing all configurations to etcd..."
	@cd config/config-cli && $(GO) run . sync --force

.PHONY: config-sync-dev
config-sync-dev:
	@echo "🔄 Syncing dev configurations to etcd..."
	@cd config/config-cli && $(GO) run . sync dev --force

# ==============================================================================
# 代码生成与格式化
# ==============================================================================

.PHONY: proto
proto:
	@echo "✨ Generating protobuf code..."
	@cd api && $(BUF) generate

.PHONY: fmt
fmt:
	@echo "🎨 Formatting Go files..."
	@$(GOFMT) -w -s $(GO_FILES)

# ==============================================================================
# 代码质量与测试
# ==============================================================================

.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@$(GOLINT) run ./...

.PHONY: test
test:
	@echo "🧪 Running tests with race detector..."
	@$(GO) test -race -cover $(GO_PACKAGES)

# ==============================================================================
# 依赖管理与清理
# ==============================================================================

.PHONY: tidy
tidy:
	@echo "🧹 Tidying go modules..."
	@$(GO) mod tidy

.PHONY: clean
clean:
	@echo "🗑️ Cleaning up..."
	@rm -rf ./gen
	@$(GO) clean -cache -testcache -modcache
