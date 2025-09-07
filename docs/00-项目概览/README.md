# GoChat 项目概览

## 项目简介

GoChat 是一个基于 Go 语言开发的分布式即时通讯系统，采用微服务架构设计，支持高并发、高可用的实时消息传输。

## 核心特性

- **分布式架构**: 基于微服务的分布式系统设计
- **高并发支持**: 支持百万级用户在线
- **实时通讯**: 基于 WebSocket 的实时消息传输
- **消息可靠性**: 基于 Kafka 的消息队列保证消息不丢失
- **水平扩展**: 支持服务实例的水平扩展
- **容器化部署**: 基于 Docker 的容器化部署方案

## 系统架构

### 服务组成

- **im-gateway**: 网关服务，处理客户端连接和消息路由
- **im-logic**: 逻辑服务，处理业务逻辑
- **im-repo**: 数据服务，负责数据持久化
- **im-task**: 任务服务，处理异步任务和大群消息扇出

### 技术栈

- **后端**: Go 1.21+
- **数据库**: MySQL 8.0+
- **缓存**: Redis 7.0+
- **消息队列**: Apache Kafka
- **服务发现**: etcd
- **容器化**: Docker & Docker Compose

## 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 7.0+
- Apache Kafka
- etcd

### 开发环境启动

```bash
# 启动开发环境
make dev

# 安装开发工具
make install-tools

# 下载依赖
make deps
```

### 构建和运行

```bash
# 构建所有服务
make build

# 运行测试
make test

# 启动服务
docker-compose up -d
```

## 项目结构

```
gochat/
├── api/                 # API 定义和生成代码
├── cmd/                 # 命令行入口
├── configs/             # 配置文件
├── internal/            # 内部实现
├── pkg/                 # 公共包
├── scripts/             # 脚本文件
├── docs/                # 文档
├── Makefile             # 构建脚本
└── docker-compose.yml   # Docker 编排
```

## 开发指南

详细的开发指南请参考 [开发指南](../06-开发指南/README.md)

## 部署指南

详细的部署指南请参考 [部署运维](../05-部署运维/README.md)

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License