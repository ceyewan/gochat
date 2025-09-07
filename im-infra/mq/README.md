# MQ - 高性能 Kafka 消息队列库 (V2)

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Kafka Version](https://img.shields.io/badge/Kafka-2.8+-231F20?style=flat&logo=apache-kafka)](https://kafka.apache.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一个基于 [franz-go](https://github.com/twmb/franz-go) 的高性能、极简 Kafka 消息队列基础库，专为 `gochat` 项目优化。

## 📖 设计与接口

**本项目的设计哲学是“极简与约定”。**

我们提供了一个极小化的 API 集合，并与 `gochat` 的基础设施（如 `coord` 配置中心和 `clog` 日志库）深度集成，旨在为业务开发者提供最简单、最直接的消息收发体验。

**详细的接口定义、设计哲学和使用示例，请参阅唯一权威的设计文档：**

➡️ **[./DESIGN.md](./DESIGN.md)**

## 🚀 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/mq
```

### 核心用法

请参考 [DESIGN.md 中的使用示例](./DESIGN.md#4-使用示例)。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request。所有代码实现都应严格遵循 [DESIGN.md](./DESIGN.md) 中定义的接口和规范。
