# 使用 `im-infra` 组件

`im-infra` 目录包含所有微服务共享的核心基础库。本指南旨在为开发者提供清晰的指引，说明如何在业务代码中正确、高效地使用这些关键组件。

所有组件的设计都遵循 `docs/08_infra/README.md` 中定义的核心规范。

---
## 统一初始化：典型服务 `main.go`

本章节提供一个**生产级别的 `main` 函数**示例，展示如何在一个典型的微服务（如 `message-service`）中，按正确的依赖顺序，一次性初始化所有需要的 `im-infra` 组件。这是上手 `im-infra` 的**黄金路径**和**最佳实践**。

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/breaker"
	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/es"
	"github.com/ceyewan/gochat/im-infra/metrics"
	"github.com/ceyewan/gochat/im-infra/mq"
	"github.com/ceyewan/gochat/im-infra/once"
	"github.com/ceyewan/gochat/im-infra/ratelimit"
	"github.com/ceyewan/gochat/im-infra/uid"
)

const (
	serviceName = "message-service"
	environment = "production" // "development" or "production"
)

func main() {
	// --- 1. 基础组件初始化 (无依赖或仅依赖 context) ---

	// 初始化日志 (clog)
	clogConfig := clog.GetDefaultConfig(environment)
	if err := clog.Init(context.Background(), clogConfig, clog.WithNamespace(serviceName)); err != nil {
		log.Fatalf("初始化 clog 失败: %v", err)
	}
	clog.Info("服务开始启动...")

	// 初始化可观测性 (metrics)
	metricsConfig := metrics.GetDefaultConfig(serviceName, environment)
	metricsProvider, err := metrics.New(context.Background(), metricsConfig,
		metrics.WithLogger(clog.Namespace("metrics")),
	)
	if err != nil {
		clog.Fatal("初始化 metrics 失败", clog.Err(err))
	}
	defer metricsProvider.Shutdown(context.Background())
	clog.Info("metrics Provider 初始化成功")

	// --- 2. 核心依赖组件初始化 (依赖 clog) ---

	// 初始化分布式协调 (coord)
	coordConfig := coord.GetDefaultConfig(environment)
	// coordConfig.Endpoints = []string{"etcd1:2379", "etcd2:2379", "etcd3:2379"} // 按需覆盖
	coordProvider, err := coord.New(context.Background(), coordConfig,
		coord.WithLogger(clog.Namespace("coord")),
	)
	if err != nil {
		clog.Fatal("初始化 coord 失败", clog.Err(err))
	}
	defer coordProvider.Close()
	clog.Info("coord Provider 初始化成功")

	// --- 3. 上层组件初始化 (依赖 clog, coord, etc.) ---

	// 初始化缓存 (cache)
	cacheConfig := cache.GetDefaultConfig(environment)
	// cacheConfig.Addr = "redis-cluster:6379" // 按需覆盖
	cacheProvider, err := cache.New(context.Background(), cacheConfig,
		cache.WithLogger(clog.Namespace("cache")),
		cache.WithCoordProvider(coordProvider),
	)
	if err != nil {
		clog.Fatal("初始化 cache 失败", clog.Err(err))
	}
	defer cacheProvider.Close()
	clog.Info("cache Provider 初始化成功")

	// 初始化数据库 (db)
	dbConfig := db.GetDefaultConfig(environment)
	// dbConfig.DSN = "..." // 按需覆盖
	dbProvider, err := db.New(context.Background(), dbConfig,
		db.WithLogger(clog.Namespace("gorm")),
	)
	if err != nil {
		clog.Fatal("初始化 db 失败", clog.Err(err))
	}
	defer dbProvider.Close()
	clog.Info("db Provider 初始化成功")

	// 初始化唯一ID (uid)
	uidConfig := uid.GetDefaultConfig(environment)
	uidConfig.ServiceName = serviceName
	uidProvider, err := uid.New(context.Background(), uidConfig,
		uid.WithLogger(clog.Namespace("uid")),
		uid.WithCoordProvider(coordProvider),
	)
	if err != nil {
		clog.Fatal("初始化 uid 失败", clog.Err(err))
	}
	defer uidProvider.Close()
	clog.Info("uid Provider 初始化成功")

	// 初始化消息队列 (mq)
	mqConfig := mq.GetDefaultConfig(environment)
	// mqConfig.Brokers = []string{"kafka1:9092", "kafka2:9092"} // 按需覆盖
	mqProducer, err := mq.NewProducer(context.Background(), mqConfig,
		mq.WithLogger(clog.Namespace("mq-producer")),
		mq.WithCoordProvider(coordProvider),
	)
	if err != nil {
		clog.Fatal("初始化 mq producer 失败", clog.Err(err))
	}
	defer mqProducer.Close()
	clog.Info("mq producer 初始化成功")

	// 初始化限流 (ratelimit)
	ratelimitConfig := ratelimit.GetDefaultConfig(environment)
	ratelimitConfig.ServiceName = serviceName
	// ratelimitConfig.RulesPath = "/config/prod/message-service/ratelimit/" // 按需覆盖
	rateLimitProvider, err := ratelimit.New(context.Background(), ratelimitConfig,
		ratelimit.WithLogger(clog.Namespace("ratelimit")),
		ratelimit.WithCoordProvider(coordProvider),
		ratelimit.WithCacheProvider(cacheProvider),
	)
	if err != nil {
		clog.Fatal("初始化 ratelimit 失败", clog.Err(err))
	}
	defer rateLimitProvider.Close()
	clog.Info("ratelimit Provider 初始化成功")

	// 初始化幂等 (once)
	onceConfig := once.GetDefaultConfig(environment)
	onceConfig.ServiceName = serviceName
	onceProvider, err := once.New(context.Background(), onceConfig,
		once.WithLogger(clog.Namespace("once")),
		once.WithCacheProvider(cacheProvider),
	)
	if err != nil {
		clog.Fatal("初始化 once 失败", clog.Err(err))
	}
	defer onceProvider.Close()
	clog.Info("once Provider 初始化成功")

	// 初始化熔断器 (breaker)
	breakerConfig := breaker.GetDefaultConfig(serviceName, environment)
	breakerProvider, err := breaker.New(context.Background(), breakerConfig,
		breaker.WithLogger(clog.Namespace("breaker")),
		breaker.WithCoordProvider(coordProvider),
	)
	if err != nil {
		clog.Fatal("初始化 breaker 失败", clog.Err(err))
	}
	defer breakerProvider.Close()
	clog.Info("breaker Provider 初始化成功")

	// 初始化消息索引 (es)
	esConfig := es.GetDefaultConfig(environment)
	// esConfig.Addresses = []string{"http://es1:9200"} // 按需覆盖
	esProvider, err := es.New(context.Background(), esConfig,
		es.WithLogger(clog.Namespace("es")),
	)
	if err != nil {
		clog.Fatal("初始化 es 失败", clog.Err(err))
	}
	defer esProvider.Close()
	clog.Info("es Provider 初始化成功")

	// --- 4. 启动业务服务 ---
	// ... 在这里，将初始化好的 providers 注入到你的业务服务中 ...
	// e.g., messageSvc := service.NewMessageService(dbProvider, cacheProvider, mqProducer, ...)
	// ... 启动 gRPC/HTTP 服务器 ...
	clog.Info("所有组件初始化完毕，服务正在运行...")

	// --- 5. 优雅关闭 ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	clog.Warn("服务开始关闭...")

	// defer 调用会自动执行，以相反的顺序关闭所有组件
	clog.Info("服务已优雅关闭")
}
```

---

## 1. `clog` - 结构化日志

`clog` 提供基于**层次化命名空间**和**上下文感知**的结构化日志解决方案。

- **初始化 (main.go)**:
  ```go
  import (
      "context"
      "log"
      "github.com/ceyewan/gochat/im-infra/clog"
  )

  // 在服务的 main 函数中，初始化全局 Logger。
  func main() {
      // 1. 使用默认配置（推荐），或从配置中心加载
      config := clog.GetDefaultConfig("development") // "development" or "production"

      // 2. 初始化全局 logger，并设置根命名空间（通常是服务名）
      if err := clog.Init(context.Background(), config, clog.WithNamespace("im-logic")); err != nil {
          log.Fatalf("初始化 clog 失败: %v", err)
      }

      clog.Info("服务启动成功")
      // 输出: {"level":"info", "namespace":"im-logic", "msg":"服务启动成功"}
  }
  ```

- **核心用法 (业务逻辑中)**:
  ```go
  // 这是一个典型的业务处理函数
  func (s *UserService) GetUser(ctx context.Context, userID string) {
      // 1. 从请求上下文中获取带 trace_id 的 logger
      //    WithContext 是 clog.C 的别名，两者等价
      logger := clog.WithContext(ctx)

      // 2. (可选) 创建一个特定于当前操作的子命名空间 logger
      //    这会自动继承根命名空间 "im-logic" 和 trace_id
      opLogger := logger.Namespace("get_user")
      
      opLogger.Info("开始获取用户信息", clog.String("user_id", userID))
      
      // ... 业务逻辑 ...
      
      if err != nil {
          opLogger.Error("获取用户信息失败", clog.Err(err))
          return
      }
      
      opLogger.Info("成功获取用户信息")
  }

  // --- 在中间件或拦截器中 ---
  // func TraceMiddleware(c *gin.Context) {
  //     // ...
  //     // 使用 WithTraceID 将 traceID 注入 context
  //     ctx := clog.WithTraceID(c.Request.Context(), traceID)
  //     c.Request = c.Request.WithContext(ctx)
  //     c.Next()
  // }
  ```

---

## 2. `coord` - 分布式协调

`coord` 提供服务发现、配置管理和分布式锁等功能。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/coord"

  // 在服务的 main 函数中，初始化 coord Provider。
  func main() {
      // ... 首先初始化 clog ...
      clog.Init(...)

      // 1. 使用默认配置（推荐），或从配置中心加载
      config := coord.GetDefaultConfig("development") // "development" or "production"
      
      // 2. 根据环境覆盖必要的配置
      config.Endpoints = []string{"localhost:2379"} // 开发环境单节点
      // config.Endpoints = []string{"etcd1:2379", "etcd2:2379", "etcd3:2379"} // 生产环境集群
      
      // 3. 创建 coord Provider 实例
      coordProvider, err := coord.New(
          context.Background(),
          config,
          coord.WithLogger(clog.Namespace("coord")),
      )
      if err != nil {
          log.Fatalf("初始化 coord 失败: %v", err)
      }
      defer coordProvider.Close()
      
      // 后续可以将 coordProvider 注入到其他需要的组件中
      // ...
  }
  ```

- **核心用法**:
  ```go
  // 1. 服务发现: 获取 gRPC 连接
  conn, err := coordProvider.Registry().GetConnection(ctx, "user-service")
  if err != nil {
      return fmt.Errorf("获取服务连接失败: %w", err)
  }
  userClient := userpb.NewUserServiceClient(conn)

  // 2. 配置管理: 获取配置
  var dbConfig myapp.DatabaseConfig
  err = coordProvider.Config().Get(ctx, "/config/dev/global/db", &dbConfig)
  if err != nil {
      return fmt.Errorf("获取配置失败: %w", err)
  }

  // 2.1. 前缀监听: 动态配置热更新
  // 示例：监听所有限流规则变更，实现无需重启的热更新
  var watchValue interface{}
  watcher, err := coordProvider.Config().WatchPrefix(ctx, "/config/ratelimit/rules/", &watchValue)
  if err != nil {
      return fmt.Errorf("创建前缀监听器失败: %w", err)
  }
  defer watcher.Close()

  go func() {
      for event := range watcher.Chan() {
          log.Printf("检测到限流规则变更: type=%s, key=%s", event.Type, event.Key)
          // 重新加载规则逻辑...
      }
  }()

  // 3. 分布式锁
  lock, err := coordProvider.Lock().Acquire(ctx, "my-resource-key", 30*time.Second)
  if err != nil {
      return fmt.Errorf("获取锁失败: %w", err)
  }
  defer lock.Unlock(ctx)
  // ... 执行关键部分 ...
  ```

---

## 3. `mq` - 消息队列

`mq` 提供了生产和消费消息的统一接口。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/mq"

  // 在服务的 main 函数中，初始化 mq Producer 和 Consumer。
  func main() {
      // ... 首先初始化 clog 和 coord ...
      clog.Init(...)
      coordProvider, _ := coord.New(...)

      // 1. 使用默认配置（推荐），或从配置中心加载
      config := mq.GetDefaultConfig("development") // "development" or "production"
      
      // 2. 根据环境覆盖必要的配置
      config.Brokers = []string{"localhost:9092"} // 开发环境单节点
      // config.Brokers = []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"} // 生产环境集群
      
      // 3. 创建 Producer 实例
      producer, err := mq.NewProducer(
          context.Background(),
          config,
          mq.WithLogger(clog.Namespace("mq-producer")),
          mq.WithCoordProvider(coordProvider),
      )
      if err != nil {
          log.Fatalf("初始化 mq producer 失败: %v", err)
      }
      defer producer.Close()
      
      // 4. 创建 Consumer 实例
      consumer, err := mq.NewConsumer(
          context.Background(),
          config,
          "notification-service-user-events-group", // 遵循命名规范的 GroupID
          mq.WithLogger(clog.Namespace("mq-consumer")),
          mq.WithCoordProvider(coordProvider),
      )
      if err != nil {
          log.Fatalf("初始化 mq consumer 失败: %v", err)
      }
      defer consumer.Close()
      
      // 后续可以将 producer 和 consumer 注入到业务服务中
      // ...
  }
  ```

- **核心用法**:
  ```go
  // 生产消息
  msg := &mq.Message{
      Topic: "user.events.registered",
      Key:   []byte("user123"),
      Value: []byte(`{"id":"user123","name":"John"}`),
  }
  
  // 异步发送（推荐）
  producer.Send(ctx, msg, func(err error) {
      if err != nil {
          clog.WithContext(ctx).Error("发送消息失败", clog.Err(err))
      }
  })
  
  // 同步发送（需要强一致性时）
  if err := producer.SendSync(ctx, msg); err != nil {
      return fmt.Errorf("发送消息失败: %w", err)
  }

  // 消费消息
  handler := func(ctx context.Context, msg *mq.Message) error {
      logger := clog.WithContext(ctx)
      logger.Info("收到消息", clog.String("topic", msg.Topic))
      
      // 处理消息逻辑
      return nil
  }
  
  topics := []string{"user.events.registered"}
  err := consumer.Subscribe(ctx, topics, handler)
  if err != nil {
      return fmt.Errorf("订阅消息失败: %w", err)
  }
  ```

---

## 4. `db` - 数据库

`db` 组件提供基于 GORM 的、支持分库分表的高性能数据库操作层。

- **初始化 (main.go)**:
  ```go
  import (
      "context"
      "encoding/json"
      "log"
      "time"

      "github.com/ceyewan/gochat/im-infra/clog"
      "github.com/ceyewan/gochat/im-infra/db"
  )

  // 在服务的 main 函数中，初始化 db Provider。
  func main() {
      // ... 首先初始化 clog ...
      clog.Init(...)

      // 1. 使用默认配置（推荐），或从配置中心加载
      config := db.GetDefaultConfig("development") // "development" or "production"
      
      // 2. 根据环境覆盖必要的配置
      config.DSN = "user:password@tcp(127.0.0.1:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local"
      
      // 3. (可选) 配置分片
      config.Sharding = &db.ShardingConfig{
          ShardingKey:    "user_id",
          NumberOfShards: 16,
          Tables: map[string]*db.TableShardingConfig{
              "messages": {},
          },
      }

      // 4. 创建 db Provider 实例
      // 最佳实践：使用 WithLogger 将 GORM 日志接入 clog
      dbProvider, err := db.New(
          context.Background(),
          config,
          db.WithLogger(clog.Namespace("gorm")),
      )
      if err != nil {
          log.Fatalf("初始化 db 失败: %v", err)
      }
      
      // 后续可以将 dbProvider 注入到业务 Repo 中
      // ...
  }
  ```

- **核心用法 (在 Repository 或 Service 中)**:
  ```go
  // 假设 dbProvider 已经通过依赖注入传入
  
  // 1. 基本查询/写入
  // 通过 db.DB(ctx) 获取带上下文的 gorm.DB 实例
  var user User
  err := dbProvider.DB(ctx).Where("id = ?", 1).First(&user).Error
  if err != nil {
      return fmt.Errorf("查询用户失败: %w", err)
  }
  
  newUser := &User{Name: "test"}
  err = dbProvider.DB(ctx).Create(newUser).Error
  if err != nil {
      return fmt.Errorf("创建用户失败: %w", err)
  }

  // 2. 涉及分片键的查询
  // 查询时必须带上分片键 `user_id`，以便 GORM 能定位到正确的表
  var messages []*Message
  err = dbProvider.DB(ctx).Where("user_id = ?", currentUserID).Find(&messages).Error
  if err != nil {
      return fmt.Errorf("查询消息失败: %w", err)
  }

  // 3. 事务
  // Transaction 方法会自动处理上下文和提交/回滚
  err = dbProvider.Transaction(ctx, func(tx *gorm.DB) error {
      // tx 实例已包含事务和上下文，可直接使用
      if err := tx.Model(&Account{}).Where("user_id = ?", fromUserID).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
          return err
      }
      if err := tx.Model(&Account{}).Where("user_id = ?", toUserID).Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
          return err
      }
      return nil
  })
  ```

---

## 5. `cache` - 缓存

`cache` 提供统一的分布式缓存接口。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/cache"

  // 在服务的 main 函数中，初始化 cache Provider。
  func main() {
      // ... 首先初始化 clog 和 coord ...
      
      // 使用默认配置（推荐）
      config := cache.GetDefaultConfig("production") // 或 "development"
      
      // 根据实际部署环境覆盖特定配置
      config.Addr = "redis-cluster:6379"
      config.Password = "your-redis-password"
      config.KeyPrefix = "gochat:"
      
      // 创建 cache Provider
      cacheProvider, err := cache.New(context.Background(), config, 
          cache.WithLogger(clog.Namespace("cache")),
          cache.WithCoordProvider(coordProvider),
      )
      if err != nil {
          clog.Fatal("初始化 cache 失败", clog.Err(err))
      }
      defer cacheProvider.Close()
      
      clog.Info("cache Provider 初始化成功")
  }
  ```

- **核心用法**:
  ```go
  // 设置缓存
  err := cacheProvider.String().Set(ctx, "user:123", "John", 10*time.Minute)
  if err != nil {
      return fmt.Errorf("设置缓存失败: %w", err)
  }

  // 获取缓存
  val, err := cacheProvider.String().Get(ctx, "user:123")
  if err != nil {
      if errors.Is(err, cache.ErrCacheMiss) {
          // 缓存未命中
          clog.WithContext(ctx).Info("缓存未命中", clog.String("key", "user:123"))
          return nil, nil
      }
      return fmt.Errorf("获取缓存失败: %w", err)
  }

  // 使用分布式锁
  lock, err := cacheProvider.Lock().Acquire(ctx, "critical-section", 30*time.Second)
  if err != nil {
      return fmt.Errorf("获取锁失败: %w", err)
  }
  defer lock.Unlock(ctx)
  
  // ... 执行关键代码 ...

  // 使用布隆过滤器检测重复
  exists, err := cacheProvider.Bloom().BFExists(ctx, "seen-items", "item123")
  if err != nil {
      return fmt.Errorf("检查布隆过滤器失败: %w", err)
  }
  if !exists {
      // 首次出现，添加到过滤器
      cacheProvider.Bloom().BFAdd(ctx, "seen-items", "item123")
  }
  ```

---

## 6. `uid` - 分布式 ID

`uid` 用于生成全局唯一的 ID，支持 Snowflake 和 UUID v7 两种方案。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/uid"

  // 在服务的 main 函数中，初始化 uid Provider。
  func main() {
      // ... 首先初始化 clog 和 coord ...
      
      // 使用默认配置（推荐）
      config := uid.GetDefaultConfig("production") // 或 "development"
      
      // 根据实际服务覆盖配置
      config.ServiceName = "message-service"
      
      // 创建 uid Provider
      uidProvider, err := uid.New(context.Background(), config,
          uid.WithLogger(clog.Namespace("uid")),
          uid.WithCoordProvider(coordProvider), // 对于 Snowflake 是必需的依赖
      )
      if err != nil {
          clog.Fatal("初始化 uid 失败", clog.Err(err))
      }
      defer uidProvider.Close()
      
      clog.Info("uid Provider 初始化成功")
  }
  ```

- **核心用法**:
  ```go
  // 生成 UUID v7（无状态，用于请求ID、资源ID等）
  requestID := uidProvider.GetUUIDV7()
  logger.Info("生成请求ID", clog.String("request_id", requestID))

  // 生成 Snowflake ID（有状态，用于数据库主键、消息ID等）
  messageID, err := uidProvider.GenerateSnowflake()
  if err != nil {
      return fmt.Errorf("生成消息ID失败: %w", err)
  }
  logger.Info("生成消息ID", clog.Int64("message_id", messageID))

  // 解析 Snowflake ID
  timestamp, instanceID, sequence := uidProvider.ParseSnowflake(messageID)
  logger.Info("解析ID",
      clog.Int64("timestamp", timestamp),
      clog.Int64("instance_id", instanceID),
      clog.Int64("sequence", sequence))
  ```

---

## 7. `ratelimit` - 分布式限流

`ratelimit` 用于控制对资源的访问速率，支持分布式和单机两种模式。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/ratelimit"

  // 在服务的 main 函数中，初始化 ratelimit Provider。
  func main() {
      // ... 首先初始化 clog、coord 和 cache ...
      
      // 1. 获取并覆盖配置
      config := ratelimit.GetDefaultConfig("production")
      config.ServiceName = "message-service"
      config.RulesPath = "/config/prod/message-service/ratelimit/"
      
      // 2. 创建 Provider
      // New 函数内部会根据 config.Mode 决定是否使用 cacheProvider
      rateLimitProvider, err := ratelimit.New(context.Background(), config,
          ratelimit.WithLogger(clog.Namespace("ratelimit")),
          ratelimit.WithCoordProvider(coordProvider),
          ratelimit.WithCacheProvider(cacheProvider), // 分布式模式依赖 cache 组件
      )
      if err != nil {
          clog.Fatal("初始化 ratelimit 失败", clog.Err(err))
      }
      defer rateLimitProvider.Close()
      
      clog.Info("ratelimit Provider 初始化成功", clog.String("mode", config.Mode))
  }
  ```

- **核心用法**:
  ```go
  // 检查单个请求是否被允许
  allowed, err := rateLimitProvider.Allow(ctx, "user:123", "send_message")
  if err != nil {
      // 降级策略：限流器异常时的处理
      clog.WithContext(ctx).Error("限流检查失败", clog.Err(err))
      // 根据业务需求决定是放行还是拒绝
      return
  }
  if !allowed {
      // 请求被限流，直接返回错误或特定状态码
      return fmt.Errorf("请求过于频繁，请稍后再试")
  }

  // ... 执行核心业务逻辑 ...
  ```

---
## 8. `once` - 分布式幂等

`once` 用于保证操作的幂等性，支持分布式和单机两种模式。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/once"

  // 在服务的 main 函数中，初始化 once Provider。
  func main() {
      // ... 首先初始化 clog 和 cache ...
      
      // 1. 获取并覆盖配置
      config := once.GetDefaultConfig("production") // 或 "development"
      config.ServiceName = "message-service"
      config.KeyPrefix = "idempotent:"
      
      // 2. 创建 Provider
      onceProvider, err := once.New(context.Background(), config,
          once.WithLogger(clog.Namespace("once")),
          once.WithCacheProvider(cacheProvider), // 分布式模式依赖 cache 组件
      )
      if err != nil {
          clog.Fatal("初始化 once 失败", clog.Err(err))
      }
      defer onceProvider.Close()
      
      clog.Info("once Provider 初始化成功", clog.String("mode", config.Mode))
  }
  ```

- **核心用法**:
  ```go
  // 无返回值的幂等操作（最常用）
  err := onceProvider.Do(ctx, "payment:process:order-123", 24*time.Hour, func() error {
      // 核心业务逻辑，只会执行一次
      return processPayment(ctx, orderData)
  })
  if err != nil {
      return fmt.Errorf("处理支付失败: %w", err)
  }

  // 有返回值的幂等操作（带结果缓存）
  result, err := onceProvider.Execute(ctx, "doc:create:xyz", 48*time.Hour, func() (any, error) {
      // 创建文档并返回结果，结果会被缓存
      return createDocument(ctx, docData)
  })
  if err != nil {
      return fmt.Errorf("创建文档失败: %w", err)
  }
  doc := result.(*Document)

  // 清除幂等状态（用于数据订正或手动重试）
  err = onceProvider.Clear(ctx, "payment:process:order-123")
  if err != nil {
      return fmt.Errorf("清除幂等状态失败: %w", err)
  }
  ```

---

## 9. `breaker` - 熔断器

`breaker` 用于保护服务，防止因依赖故障引起的雪崩效应。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/breaker"

  // 在服务的 main 函数中，初始化 breaker Provider。
  func main() {
      // ... 首先初始化 clog 和 coord ...
      
      // 1. 获取并覆盖配置
      // 推荐使用 GetDefaultConfig 获取标准配置，然后按需覆盖
      config := breaker.GetDefaultConfig("message-service", "production")
      // config.PoliciesPath = "/custom/path/if/needed" // 按需覆盖
      
      // 2. 创建 Provider，并通过 With... Options 注入依赖
      breakerProvider, err := breaker.New(context.Background(), config,
          breaker.WithLogger(clog.Namespace("breaker")),
          breaker.WithCoordProvider(coordProvider), // 依赖 coord 组件
      )
      if err != nil {
          clog.Fatal("初始化 breaker 失败", clog.Err(err))
      }
      defer breakerProvider.Close()
      
      clog.Info("breaker Provider 初始化成功")
  }
  ```

- **核心用法**:
  ```go
  // 获取熔断器实例
  b := breakerProvider.GetBreaker("grpc:user-service:GetUserInfo")
  
  // 将操作包裹在熔断器中执行
  err := b.Do(ctx, func() error {
      // 核心业务逻辑，如gRPC调用、HTTP请求等
      return callDownstreamService(ctx)
  })
  
  // 处理熔断器错误
  if errors.Is(err, breaker.ErrBreakerOpen) {
      // 熔断器处于打开状态，请求被拒绝，可以返回特定错误或执行降级逻辑
      return fmt.Errorf("服务暂时不可用，请稍后重试")
  }
  if err != nil {
      // 其他业务错误
      return fmt.Errorf("调用失败: %w", err)
  }
  ```

---

## 10. `metrics` - 可观测性

`metrics` 组件基于 OpenTelemetry，为所有服务提供开箱即用的指标 (Metrics) 和链路追踪 (Tracing) 能力。其核心是**自动化**和**零侵入**。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/metrics"

  // 在服务的 main 函数中，初始化 metrics Provider。
  func main() {
      // ... 首先初始化 clog ...
      
      // 1. 获取并覆盖配置
      // 推荐使用 GetDefaultConfig 获取标准配置
      config := metrics.GetDefaultConfig("message-service", "production")
      // config.ExporterEndpoint = "http://my-jaeger:14268/api/traces" // 按需覆盖
      
      // 2. 创建 Provider
      metricsProvider, err := metrics.New(context.Background(), config,
          metrics.WithLogger(clog.Namespace("metrics")),
      )
      if err != nil {
          clog.Fatal("初始化 metrics 失败", clog.Err(err))
      }
      defer metricsProvider.Shutdown(context.Background())
      
      clog.Info("metrics Provider 初始化成功")
  }
  ```

- **核心用法 (集成)**:

  `metrics` 的主要用法是在创建 gRPC 服务器或客户端时，链入由 `Provider` 提供的拦截器。

  ```go
  // 在创建 gRPC Server 时集成
  server := grpc.NewServer(
      grpc.ChainUnaryInterceptor(
          metricsProvider.GRPCServerInterceptor(),
          // ... 其他拦截器，如 a, b, c ...
      ),
  )

  // 在创建 gRPC Client 时集成
  conn, err := grpc.Dial(
      "target-service",
      grpc.WithUnaryInterceptor(metricsProvider.GRPCClientInterceptor()),
  )
  
  // 对于 Gin HTTP 服务
  // engine.Use(metricsProvider.HTTPMiddleware())
  ```
 
 ---
 
 ## 11. `es` - 分布式泛型索引
 
 `es` 组件提供了与 Elasticsearch 交互的统一接口，用于索引和搜索任何实现了 `es.Indexable` 接口的数据。
 
 - **初始化 (main.go)**:
   ```go
   import "github.com/ceyewan/gochat/im-infra/es"
 
   // 在服务的 main 函数中，初始化 es Provider。
   func main() {
       // ... 首先初始化 clog ...
       
       // 1. 获取并覆盖配置
       config := es.GetDefaultConfig("production")
       config.Addresses = []string{"http://elasticsearch:9200"}
       
       // 2. 创建 Provider
       esProvider, err := es.New(context.Background(), config,
           es.WithLogger(clog.Namespace("es")),
       )
       if err != nil {
           clog.Fatal("初始化 es 失败", clog.Err(err))
       }
       defer esProvider.Close()
       
       clog.Info("es Provider 初始化成功")
   }
   ```
 
 - **核心用法**:
   ```go
   // 1. 在业务代码中定义你的模型
   type MyMessage struct {
       MessageID string `json:"message_id"`
       SessionID string `json:"session_id"`
       Content   string `json:"content"`
   }
 
   // 2. 实现 es.Indexable 接口
   func (m MyMessage) GetID() string {
       return m.MessageID
   }
 
   // 3. 批量索引 (泛型调用)
   messages := []MyMessage{
       { MessageID: "1", SessionID: "s1", Content: "你好" },
       { MessageID: "2", SessionID: "s1", Content: "世界" },
   }
   err := esProvider.BulkIndex(ctx, messages)
   if err != nil {
       // 处理错误
   }
 
   // 4. 在会话内搜索 (泛型调用)
   results, err := esProvider.SearchInSession[MyMessage](ctx, "user1", "s1", "你好", 1, 10)
   if err != nil {
       // 处理错误
   }
   log.Printf("找到 %d 条消息", results.Total)
   for _, msg := range results.Messages {
       // msg 是 *MyMessage 类型
       log.Printf("消息内容: %s", msg.Content)
   }
   ```