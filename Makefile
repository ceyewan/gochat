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
