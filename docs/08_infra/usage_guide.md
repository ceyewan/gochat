# 使用 `im-infra` 组件

`im-infra` 目录包含所有微服务共享的核心基础库。本指南旨在为开发者提供清晰的指引，说明如何在业务代码中正确、高效地使用这些关键组件。

所有组件的设计都遵循 `docs/08_infra/README.md` 中定义的核心规范。

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
          coord.WithLogger(clog.Module("coord")),
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
  conn, err := coordinator.Registry().GetConnection(ctx, "user-service")
  if err != nil {
      return fmt.Errorf("获取服务连接失败: %w", err)
  }
  userClient := userpb.NewUserServiceClient(conn)

  // 2. 配置管理: 获取配置
  var dbConfig myapp.DatabaseConfig
  err = coordinator.Config().Get(ctx, "/config/dev/global/db", &dbConfig)
  if err != nil {
      return fmt.Errorf("获取配置失败: %w", err)
  }

  // 3. 分布式锁
  lock, err := coordinator.Lock().Acquire(ctx, "my-resource-key", 30*time.Second)
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
          mq.WithLogger(clog.NameSpace("mq-producer")),
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
          mq.WithLogger(clog.Module("mq-consumer")),
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
          db.WithLogger(clog.Module("gorm")),
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

  var config cache.Config
  // ...

  c, err := cache.New(context.Background(), &config)
  // ...
  ```

- **核心用法**:
  ```go
  // 设置缓存
  err := c.Set(ctx, "key", "value", 10*time.Minute)

  // 获取缓存
  var val string
  err := c.Get(ctx, "key", &val)
  if err == cache.ErrCacheMiss {
      // 缓存未命中
  }
  ```

---

## 6. `uid` - 分布式 ID

`uid` 用于生成全局唯一的 ID。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/uid"

  // coordProvider 是已经初始化好的 coord.Provider 实例
  generator, err := uid.New(context.Background(), coordProvider, "my-service-name")
  // ...
  ```

- **核心用法**:
  ```go
  // 生成一个新 ID
  id, err := generator.NextID()
  ```

---

## 7. `ratelimit` - 分布式限流

`ratelimit` 用于控制对资源的访问速率。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/ratelimit"

  var config ratelimit.Config
  // ...

  limiter, err := ratelimit.New(context.Background(), &config)
  // ...
  ```

- **核心用法**:
  ```go
  // 检查是否允许
  allowed, err := limiter.Allow(ctx, "resource-key")
  if !allowed {
      // 请求被限流
  }
  ```

---

## 8. `retry` - 优雅重试

`retry` 提供策略驱动的、统一的错误重试机制。

- **核心用法 (无需初始化)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/retry"

  policy := retry.Policy{
      MaxRetries: 3,
      Backoff:    retry.ExponentialBackoff(100*time.Millisecond, 2),
      Jitter:     true,
  }

  err := retry.Do(ctx, policy, func(ctx context.Context) error {
      // ... 执行可能会失败的操作 ...
      return someOperation()
  })
  ```

---

## 9. `breaker` - 熔断器

`breaker` 用于保护服务，防止因依赖故障引起的雪崩效应。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/breaker"

  var config breaker.Config
  // ...

  provider, err := breaker.New(context.Background(), &config)
  // ...
  ```

- **核心用法**:
  ```go
  // 获取特定资源的熔断器
  br := provider.GetBreaker("downstream-service-A")

  // 将操作包裹在熔断器中执行
  err := br.Do(ctx, func() error {
      // ... 调用下游服务 ...
      return callDownstream()
  })

  if err == breaker.ErrOpenState {
      // 熔断器处于打开状态，请求被拒绝
  }
  ```

### 组合使用: `retry` + `breaker`

最佳实践是将 `retry` 嵌套在 `breaker` 内部，以避免在熔断器打开时进行无效的重试。

```go
br := provider.GetBreaker("my-resource")
policy := retry.Policy{ /* ... */ }

err := br.Do(ctx, func() error {
    return retry.Do(ctx, policy, func(ctx context.Context) error {
        // 只有在熔断器闭合或半开时，才会执行此处的重试逻辑
        return someFlakyOperation()
    })
})