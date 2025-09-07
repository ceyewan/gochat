# 使用 `im-infra` 组件

`im-infra` 目录包含所有微服务共享的核心基础库。本指南旨在为开发者提供清晰的指引，说明如何在业务代码中正确、高效地使用这些关键组件。

所有组件的设计都遵循 `docs/08_infra/README.md` 中定义的核心规范。

---

## 1. `clog` - 结构化日志

`clog` 提供标准化的结构化日志接口，自动注入 `service` 和 `trace_id`。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/clog"

  // 在服务启动时，初始化 logger 并设置全局 service 名称
  logger := clog.New(clog.WithService("im-logic"))
  ```

- **核心用法 (业务逻辑中)**:
  ```go
  // 在请求入口（如 Gateway）生成 trace_id 并存入 context
  ctx := clog.SetTraceID(context.Background(), newTraceID())

  // 在业务逻辑中，从 context 获取包含 trace_id 的 logger
  logger := clog.FromContext(ctx)

  // 使用结构化上下文记录消息
  logger.Info("用户已登录",
      clog.String("username", "test"),
      clog.Uint64("user_id", 123),
  )

  logger.Error("无法连接到数据库", clog.Err(err))
  ```

---

## 2. `coord` - 分布式协调

`coord` 提供服务发现、配置管理和分布式锁等功能。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/coord"

  // config 从配置文件或配置中心加载
  var config coord.Config
  // ...

  coordinator, err := coord.New(context.Background(), &config)
  if err != nil {
      // 处理错误
  }
  defer coordinator.Close()
  ```

- **核心用法**:
  ```go
  // 1. 服务发现: 获取 gRPC 连接
  conn, err := coordinator.Registry().GetConnection(ctx, "user-service")
  // ...
  userClient := userpb.NewUserServiceClient(conn)

  // 2. 配置管理: 获取配置
  var dbConfig myapp.DatabaseConfig
  err = coordinator.Config().Get(ctx, "/config/dev/im-repo/db", &dbConfig)
  // ...

  // 3. 分布式锁
  lock, err := coordinator.Lock().Acquire(ctx, "my-resource-key", 30*time.Second)
  if err != nil {
      // 锁已被持有
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

  var config mq.Config
  // ...
  
  producer, err := mq.NewProducer(context.Background(), &config)
  // ...
  consumer, err := mq.NewConsumer(context.Background(), &config, "group-id")
  // ...
  ```

- **核心用法**:
  ```go
  // 生产消息
  err := producer.Publish(ctx, "topic-name", mq.NewMessage("key", []byte("value")))

  // 消费消息
  err := consumer.Subscribe(ctx, "topic-name", func(msg mq.Message) error {
      // 处理消息
      return nil
  })
  ```

---

## 4. `db` - 数据库

`db` 组件提供对 SQL 数据库的访问和操作接口。

- **初始化 (main.go)**:
  ```go
  import "github.com/ceyewan/gochat/im-infra/db"

  var config db.Config
  // ...

  db, err := db.New(context.Background(), &config)
  // ...
  ```

- **核心用法**:
  ```go
  // 查询
  var user User
  err := db.Reader().Where("id = ?", 1).First(&user).Error

  // 写入
  err = db.Writer().Create(&User{Name: "test"}).Error

  // 事务
  err = db.Transaction(ctx, func(tx *gorm.DB) error {
      // ... 在事务中执行操作
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