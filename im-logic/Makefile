.PHONY: build run test clean docker-build docker-up docker-down

# 默认目标
all: build

# 构建应用
build:
	go build -o bin/im-logic ./cmd/server

# 运行应用
run:
	go run ./cmd/server

# 运行测试
test:
	go test -v -race -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
lint:
	golangci-lint run

# 生成 protobuf 代码
proto:
	@echo "Generating protobuf code..."
	@cd ../api && make proto

# 清理构建文件
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# 构建 Docker 镜像
docker-build:
	docker build -t gochat/im-logic:latest .

# 启动 Docker 服务
docker-up:
	docker-compose up -d

# 停止 Docker 服务
docker-down:
	docker-compose down

# 重启 Docker 服务
docker-restart:
	docker-compose restart

# 查看 Docker 日志
docker-logs:
	docker-compose logs -f

# 进入容器
docker-shell:
	docker-compose exec im-logic sh

# 安装依赖
deps:
	go mod download
	go mod tidy

# 安装开发工具
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 开发环境启动
dev:
	@echo "Starting development environment..."
	docker-compose up -d mysql redis kafka zookeeper etcd
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Starting application..."
	go run ./cmd/server

# 健康检查
health-check:
	@echo "Checking health status..."
	curl -f http://localhost:9002/health || exit 1

# 性能测试
bench:
	go test -bench=. -benchmem ./...

# 安全扫描
security:
	gosec ./...

# 依赖检查
deps-check:
	go mod verify
	go list -m all | grep -v indirect

# 更新依赖
deps-update:
	go get -u ./...
	go mod tidy

# 生成 API 文档
docs:
	@echo "Generating API documentation..."
	swag init -g ./cmd/server/main.go -o ./docs

# 部署到测试环境
deploy-test:
	@echo "Deploying to test environment..."
	docker build -t gochat/im-logic:test .
	docker tag gochat/im-logic:test registry.example.com/gochat/im-logic:test
	docker push registry.example.com/gochat/im-logic:test

# 部署到生产环境
deploy-prod:
	@echo "Deploying to production environment..."
	docker build -t gochat/im-logic:latest .
	docker tag gochat/im-logic:latest registry.example.com/gochat/im-logic:latest
	docker push registry.example.com/gochat/im-logic:latest

# 回滚版本
rollback:
	@echo "Rolling back to previous version..."
	docker tag gochat/im-logic:previous registry.example.com/gochat/im-logic:latest
	docker push registry.example.com/gochat/im-logic:latest

# 监控指标查看
metrics:
	@echo "Fetching metrics..."
	curl -s http://localhost:9003/metrics | head -20

# 数据库迁移
migrate:
	@echo "Running database migrations..."
	@cd ../scripts/mysql && ./migrate.sh

# 创建数据库备份
backup:
	@echo "Creating database backup..."
	docker exec mysql mysqldump -u root -ppassword gochat > backup.sql

# 恢复数据库
restore:
	@echo "Restoring database from backup..."
	docker exec -i mysql mysql -u root -ppassword gochat < backup.sql
