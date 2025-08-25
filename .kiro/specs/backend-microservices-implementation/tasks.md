# 实施计划

- [ ] 1. 建立项目结构和 im-infra 基础
  - 为所有服务创建具有适当目录布局的 Go 模块结构
  - 使用 Viper 和环境变量支持实现基本配置管理
  - 设置包含 trace ID 支持的 Zap 结构化日志
  - 创建基础服务接口和通用工具
  - _需求: 1.1, 1.5, 1.6_

- [ ] 2. 实现核心 im-infra 数据库模块
  - 使用 GORM 创建支持主从连接的数据库管理器
  - 实现连接池和健康检查机制
  - 添加具有适当错误处理的事务管理工具
  - 创建数据库迁移框架和初始表结构
  - 使用模拟连接为数据库操作编写单元测试
  - _需求: 1.1, 2.5, 2.6_

- [ ] 3. 实现 im-infra 缓存模块
  - 创建具有连接池和错误处理的 Redis 客户端包装器
  - 实现具有自动回退机制的 Cache-Aside 模式
  - 使用 Redis 添加分布式锁定工具
  - 创建缓存操作工具 (Get, Set, Del, Incr, ZAdd, ZRevRange)
  - 使用 Redis testcontainer 为缓存操作编写单元测试
  - _需求: 1.2, 2.5, 2.6_

- [ ] 4. 实现 im-infra 消息队列模块
  - 创建具有自动序列化和错误处理的 Kafka 生产者包装器
  - 实现具有消息处理器接口的 Kafka 消费者组抽象
  - 添加消息路由和主题管理工具
  - 创建重试机制和死信队列支持
  - 使用 Kafka testcontainer 为消息队列操作编写单元测试
  - _需求: 1.3, 6.3_

- [ ] 5. 实现 im-infra ID 生成和服务发现
  - 创建基于 Snowflake 的 ID 生成器，使用 etcd 机器 ID 管理
  - 实现用于服务注册和发现的 etcd 客户端工具
  - 添加支持热重载的分布式配置管理
  - 创建用于分布式追踪的 OpenTelemetry 集成
  - 为 ID 生成和服务发现编写单元测试
  - _需求: 1.4, 1.7, 1.8, 6.1_

- [ ] 6. 创建 protobuf 定义并生成 gRPC 代码
  - 为 im-repo、im-logic 服务定义所有 gRPC 服务接口
  - 为用户、消息、群组和会话操作创建消息结构
  - 定义 Kafka 消息格式和 WebSocket 消息协议
  - 从 protobuf 定义生成 Go 代码
  - 设置代码生成脚本和构建自动化
  - _需求: 2.1, 2.2, 2.3, 3.1, 3.2, 4.1, 4.2_

- [ ] 7. Implement im-repo user repository service
  - Create user CRUD operations with bcrypt password hashing
  - Implement user session management with Redis storage
  - Add user authentication and JWT token validation
  - Create gRPC server setup with proper middleware and error handling
  - Write unit tests for user operations and integration tests with database
  - _Requirements: 2.1, 2.7, 2.8, 3.1, 3.2_

- [ ] 8. Implement im-repo message repository service
  - Create message storage operations with atomic MySQL and Redis updates
  - Implement message retrieval with pagination and caching
  - Add conversation message management and sequence number generation
  - Create hot message cache management with automatic trimming
  - Write unit tests for message operations and cache consistency
  - _Requirements: 2.2, 2.3, 2.5, 2.6, 7.1, 7.2_

- [ ] 9. Implement im-repo group repository service
  - Create group CRUD operations with member management
  - Implement group member operations (add, remove, list)
  - Add online member tracking and batch operations
  - Create group information caching and invalidation
  - Write unit tests for group operations and member management
  - _Requirements: 2.4, 2.5, 2.6, 3.4_

- [ ] 10. Set up im-repo service infrastructure
  - Create main application with gRPC server setup and service registration
  - Implement health checks and metrics collection
  - Add graceful shutdown handling and connection cleanup
  - Create Docker containerization with multi-stage builds
  - Write integration tests for complete service functionality
  - _Requirements: 6.1, 6.6, 6.7_

- [ ] 11. Implement im-logic authentication service
  - Create user registration with input validation and duplicate checking
  - Implement user login with credential verification and JWT generation
  - Add token refresh and logout functionality
  - Create password reset and user profile management
  - Write unit tests for authentication flows and security validation
  - _Requirements: 3.1, 3.2, 6.4_

- [ ] 12. Implement im-logic message processing core
  - Create Kafka consumer for upstream message processing
  - Implement message validation, ID generation, and sequence numbering
  - Add idempotency checking using Redis-based deduplication
  - Create message persistence coordination with im-repo service
  - Write unit tests for message processing pipeline and error handling
  - _Requirements: 3.3, 3.7, 7.1, 7.3_

- [ ] 13. Implement im-logic message distribution engine
  - Create message distribution strategy based on conversation type
  - Implement single chat message routing to specific users
  - Add small group real-time fanout (≤500 members) with online member lookup
  - Create large group async task scheduling for im-task service
  - Write unit tests for distribution logic and routing decisions
  - _Requirements: 3.4, 3.5, 3.6_

- [ ] 14. Implement im-logic conversation management
  - Create conversation list aggregation with unread counts and last messages
  - Implement conversation metadata management and caching
  - Add group creation and management operations
  - Create conversation search and filtering capabilities
  - Write unit tests for conversation operations and data aggregation
  - _Requirements: 3.8, 2.4_

- [ ] 15. Set up im-logic service infrastructure
  - Create main application with gRPC server and Kafka consumer setup
  - Implement service registration, health checks, and metrics
  - Add graceful shutdown with proper Kafka consumer cleanup
  - Create Docker containerization and deployment configuration
  - Write integration tests for complete message processing workflow
  - _Requirements: 6.1, 6.2, 6.6, 6.7_

- [ ] 16. Implement im-gateway HTTP API handlers
  - Create REST API endpoints for authentication (register, login, logout)
  - Implement conversation management endpoints (list, messages, mark read)
  - Add request validation, JWT middleware, and error handling
  - Create API response formatting and status code management
  - Write unit tests for HTTP handlers and middleware
  - _Requirements: 4.5, 4.1, 4.2_

- [ ] 17. Implement im-gateway WebSocket connection management
  - Create WebSocket upgrade handler with JWT authentication
  - Implement connection hub for managing active user connections
  - Add connection lifecycle management (connect, disconnect, cleanup)
  - Create user session registration in Redis for service discovery
  - Write unit tests for WebSocket connection handling and session management
  - _Requirements: 4.1, 4.6, 4.7_

- [ ] 18. Implement im-gateway message handling
  - Create WebSocket message parsing and validation
  - Implement upstream message publishing to Kafka with proper serialization
  - Add immediate message acknowledgment for client UI optimization
  - Create downstream message consumption and delivery to WebSocket connections
  - Write unit tests for message flow and WebSocket communication
  - _Requirements: 4.2, 4.3, 4.4, 4.8_

- [ ] 19. Set up im-gateway service infrastructure
  - Create main application with HTTP server, WebSocket handler, and Kafka integration
  - Implement service registration, health checks, and connection metrics
  - Add graceful shutdown with proper WebSocket connection cleanup
  - Create Docker containerization and load balancer configuration
  - Write integration tests for complete gateway functionality
  - _Requirements: 6.1, 6.6, 6.7_

- [ ] 20. Implement im-task framework and large group processor
  - Create task processing framework with pluggable processor interface
  - Implement Kafka consumer for task queue with proper error handling
  - Add large group message fanout processor with batch member processing
  - Create task retry mechanisms with exponential backoff
  - Write unit tests for task processing and retry logic
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [ ] 21. Set up im-task service infrastructure
  - Create main application with Kafka consumer and task processor registration
  - Implement service registration, health checks, and processing metrics
  - Add graceful shutdown with proper task completion handling
  - Create Docker containerization and horizontal scaling configuration
  - Write integration tests for task processing workflow
  - _Requirements: 6.1, 6.6, 6.7, 5.5_

- [ ] 22. Implement system monitoring and observability
  - Add Prometheus metrics collection to all services
  - Implement distributed tracing with OpenTelemetry across service boundaries
  - Create structured logging with trace ID propagation
  - Add health check endpoints and service status monitoring
  - Write monitoring integration tests and alerting configuration
  - _Requirements: 6.7, 6.4, 1.8_

- [ ] 23. Implement error handling and resilience patterns
  - Add circuit breaker patterns for external service calls
  - Implement proper gRPC error mapping and status codes
  - Create retry mechanisms with exponential backoff for transient failures
  - Add graceful degradation for non-critical operations
  - Write unit tests for error scenarios and resilience patterns
  - _Requirements: 6.5, 2.7, 7.5_

- [ ] 24. Create deployment and configuration management
  - Create Docker Compose setup for local development environment
  - Implement Kubernetes deployment manifests with proper resource limits
  - Add configuration management with environment-specific settings
  - Create database migration scripts and initialization procedures
  - Write deployment automation and environment setup documentation
  - _Requirements: 6.1, 6.6_

- [ ] 25. Implement comprehensive testing suite
  - Create integration tests for complete message flow (send to receive)
  - Add load testing scenarios for concurrent users and message throughput
  - Implement end-to-end tests with all services running
  - Create performance benchmarks and optimization validation
  - Write test automation and continuous integration setup
  - _Requirements: 7.6, 7.7, 7.8_

- [ ] 26. 最终系统集成和验证
  - 集成所有服务并验证完整的系统功能
  - 测试前端与已实现后端 API 的集成
  - 根据需求验证系统性能（50K QPS，<50ms 延迟）
  - 创建系统文档和操作手册
  - 执行安全验证和渗透测试
  - _需求: 6.8, 7.4_