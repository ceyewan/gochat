---
inclusion: fileMatch
fileMatchPattern: "im-infra/**/*.go"
---

# IM-Infra 库指南

## 库设计原则

### 统一接口设计
所有基础设施组件应提供一致、易用的接口:

```go
// Standard initialization pattern
type Component interface {
    Init(config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error
}

// Factory pattern for component creation
type Factory interface {
    Create(config Config) (Component, error)
}
```

### 配置管理
实现灵活的配置和验证:

```go
// Base configuration interface
type Configurable interface {
    Validate() error
    SetDefaults()
}

// Configuration loader with multiple sources
type ConfigLoader struct {
    sources []ConfigSource
}

func (cl *ConfigLoader) Load(target interface{}) error {
    for _, source := range cl.sources {
        if err := source.Load(target); err != nil {
            return fmt.Errorf("failed to load from %s: %w", source.Name(), err)
        }
    }
    
    if validator, ok := target.(Configurable); ok {
        validator.SetDefaults()
        return validator.Validate()
    }
    
    return nil
}
```

## 数据库模块 (im-infra/db)

### 连接管理
实现健壮的数据库连接处理:

```go
type DBManager struct {
    master *gorm.DB
    slaves []*gorm.DB
    config DBConfig
    logger *zap.Logger
}

func NewDBManager(config DBConfig, logger *zap.Logger) (*DBManager, error) {
    master, err := gorm.Open(mysql.Open(config.Master.DSN), &gorm.Config{
        Logger: newGormLogger(logger),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to master: %w", err)
    }
    
    var slaves []*gorm.DB
    for _, slaveConfig := range config.Slaves {
        slave, err := gorm.Open(mysql.Open(slaveConfig.DSN), &gorm.Config{
            Logger: newGormLogger(logger),
        })
        if err != nil {
            logger.Warn("failed to connect to slave", 
                zap.String("dsn", slaveConfig.DSN), 
                zap.Error(err))
            continue
        }
        slaves = append(slaves, slave)
    }
    
    return &DBManager{
        master: master,
        slaves: slaves,
        config: config,
        logger: logger,
    }, nil
}

// Read operations use slaves with fallback to master
func (dm *DBManager) Reader() *gorm.DB {
    if len(dm.slaves) == 0 {
        return dm.master
    }
    
    // Simple round-robin selection
    index := rand.Intn(len(dm.slaves))
    return dm.slaves[index]
}

// Write operations always use master
func (dm *DBManager) Writer() *gorm.DB {
    return dm.master
}
```

### 事务管理
提供带有适当错误处理的事务工具:

```go
type TxManager struct {
    db     *gorm.DB
    logger *zap.Logger
}

func (tm *TxManager) WithTransaction(ctx context.Context, fn func(*gorm.DB) error) error {
    tx := tm.db.WithContext(ctx).Begin()
    if tx.Error != nil {
        return fmt.Errorf("failed to begin transaction: %w", tx.Error)
    }
    
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            tm.logger.Error("transaction panicked", zap.Any("panic", r))
            panic(r)
        }
    }()
    
    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback().Error; rbErr != nil {
            tm.logger.Error("failed to rollback transaction", zap.Error(rbErr))
        }
        return err
    }
    
    if err := tx.Commit().Error; err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

## 缓存模块 (im-infra/cache)

### Redis 客户端包装器
实现带有适当错误处理和监控的 Redis 操作:

```go
type CacheClient struct {
    client  *redis.Client
    logger  *zap.Logger
    metrics *CacheMetrics
}

func (c *CacheClient) Get(ctx context.Context, key string) (string, error) {
    start := time.Now()
    defer func() {
        c.metrics.RecordOperation("get", time.Since(start))
    }()
    
    result, err := c.client.Get(ctx, key).Result()
    if err != nil {
        if errors.Is(err, redis.Nil) {
            c.metrics.RecordCacheMiss(key)
            return "", ErrCacheMiss
        }
        c.metrics.RecordError("get")
        return "", fmt.Errorf("cache get failed: %w", err)
    }
    
    c.metrics.RecordCacheHit(key)
    return result, nil
}

func (c *CacheClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
    start := time.Now()
    defer func() {
        c.metrics.RecordOperation("set", time.Since(start))
    }()
    
    if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
        c.metrics.RecordError("set")
        return fmt.Errorf("cache set failed: %w", err)
    }
    
    return nil
}
```

### 分布式锁实现
提供带有适当超时和续期的分布式锁:

```go
type DistributedLock struct {
    client   *redis.Client
    key      string
    value    string
    ttl      time.Duration
    logger   *zap.Logger
    stopCh   chan struct{}
    renewalWg sync.WaitGroup
}

func (dl *DistributedLock) Acquire(ctx context.Context) error {
    script := `
        if redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2], "NX") then
            return 1
        else
            return 0
        end
    `
    
    result, err := dl.client.Eval(ctx, script, []string{dl.key}, dl.value, dl.ttl.Milliseconds()).Result()
    if err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    
    if result.(int64) == 0 {
        return ErrLockAcquisitionFailed
    }
    
    // Start renewal goroutine
    dl.stopCh = make(chan struct{})
    dl.renewalWg.Add(1)
    go dl.renewalLoop()
    
    return nil
}

func (dl *DistributedLock) renewalLoop() {
    defer dl.renewalWg.Done()
    
    ticker := time.NewTicker(dl.ttl / 3) // Renew at 1/3 of TTL
    defer ticker.Stop()
    
    for {
        select {
        case <-dl.stopCh:
            return
        case <-ticker.C:
            if err := dl.renew(); err != nil {
                dl.logger.Error("failed to renew lock", zap.Error(err))
                return
            }
        }
    }
}
```

## 消息队列模块 (im-infra/mq)

### Kafka 生产者包装器
实现带有重试的可靠消息生产:

```go
type Producer struct {
    writer  *kafka.Writer
    logger  *zap.Logger
    metrics *MQMetrics
}

func (p *Producer) SendMessage(ctx context.Context, topic string, msg *Message) error {
    kafkaMsg := kafka.Message{
        Topic: topic,
        Key:   []byte(msg.Key),
        Value: msg.Value,
        Headers: []kafka.Header{
            {Key: "trace_id", Value: []byte(msg.TraceID)},
            {Key: "timestamp", Value: []byte(strconv.FormatInt(time.Now().Unix(), 10))},
        },
    }
    
    start := time.Now()
    err := p.writer.WriteMessages(ctx, kafkaMsg)
    duration := time.Since(start)
    
    if err != nil {
        p.metrics.RecordError(topic, "send")
        p.logger.Error("failed to send message",
            zap.String("topic", topic),
            zap.String("trace_id", msg.TraceID),
            zap.Error(err))
        return fmt.Errorf("failed to send message: %w", err)
    }
    
    p.metrics.RecordSuccess(topic, "send", duration)
    return nil
}
```

### 消费者组实现
提供健壮的消费者组处理:

```go
type ConsumerGroup struct {
    reader    *kafka.Reader
    processor MessageProcessor
    logger    *zap.Logger
    metrics   *MQMetrics
}

func (cg *ConsumerGroup) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := cg.reader.ReadMessage(ctx)
            if err != nil {
                cg.logger.Error("failed to read message", zap.Error(err))
                cg.metrics.RecordError(msg.Topic, "consume")
                continue
            }
            
            if err := cg.processMessage(ctx, msg); err != nil {
                cg.logger.Error("failed to process message",
                    zap.String("topic", msg.Topic),
                    zap.Int("partition", msg.Partition),
                    zap.Int64("offset", msg.Offset),
                    zap.Error(err))
                // Implement retry logic or dead letter queue
                continue
            }
        }
    }
}

func (cg *ConsumerGroup) processMessage(ctx context.Context, msg kafka.Message) error {
    start := time.Now()
    defer func() {
        cg.metrics.RecordProcessingTime(msg.Topic, time.Since(start))
    }()
    
    // Extract trace ID for distributed tracing
    traceID := extractTraceID(msg.Headers)
    ctx = context.WithValue(ctx, "trace_id", traceID)
    
    return cg.processor.Process(ctx, &msg)
}
```

## ID 生成模块 (im-infra/idgen)

### Snowflake 实现
提供带有机器 ID 管理的分布式 ID 生成:

```go
type SnowflakeGenerator struct {
    node      *snowflake.Node
    machineID int64
    logger    *zap.Logger
    etcdClient *clientv3.Client
}

func NewSnowflakeGenerator(etcdClient *clientv3.Client, logger *zap.Logger) (*SnowflakeGenerator, error) {
    machineID, err := acquireMachineID(etcdClient)
    if err != nil {
        return nil, fmt.Errorf("failed to acquire machine ID: %w", err)
    }
    
    node, err := snowflake.NewNode(machineID)
    if err != nil {
        return nil, fmt.Errorf("failed to create snowflake node: %w", err)
    }
    
    sg := &SnowflakeGenerator{
        node:       node,
        machineID:  machineID,
        logger:     logger,
        etcdClient: etcdClient,
    }
    
    // Start machine ID renewal
    go sg.renewMachineID()
    
    return sg, nil
}

func (sg *SnowflakeGenerator) NextID() int64 {
    return sg.node.Generate().Int64()
}

func acquireMachineID(client *clientv3.Client) (int64, error) {
    // Try to acquire a machine ID from 1 to 1023
    for i := int64(1); i <= 1023; i++ {
        key := fmt.Sprintf("/gochat/machine_ids/%d", i)
        
        resp, err := client.Txn(context.Background()).
            If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
            Then(clientv3.OpPut(key, "acquired", clientv3.WithLease(leaseID))).
            Commit()
        
        if err != nil {
            continue
        }
        
        if resp.Succeeded {
            return i, nil
        }
    }
    
    return 0, errors.New("no available machine ID")
}
```

## 日志模块 (im-infra/clog)

### 结构化日志实现
为所有服务提供一致的日志记录:

```go
type Logger struct {
    zap    *zap.Logger
    config LogConfig
}

func NewLogger(config LogConfig) (*Logger, error) {
    zapConfig := zap.NewProductionConfig()
    zapConfig.Level = zap.NewAtomicLevelAt(config.Level)
    zapConfig.OutputPaths = config.OutputPaths
    zapConfig.ErrorOutputPaths = config.ErrorOutputPaths
    
    // Add custom encoder for structured logging
    zapConfig.EncoderConfig.TimeKey = "timestamp"
    zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    logger, err := zapConfig.Build()
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }
    
    return &Logger{
        zap:    logger,
        config: config,
    }, nil
}

func (l *Logger) WithTraceID(traceID string) *Logger {
    return &Logger{
        zap:    l.zap.With(zap.String("trace_id", traceID)),
        config: l.config,
    }
}

func (l *Logger) WithService(service string) *Logger {
    return &Logger{
        zap:    l.zap.With(zap.String("service", service)),
        config: l.config,
    }
}
```

## 监控和指标

### 指标收集
为所有组件实现全面的指标:

```go
type Metrics struct {
    registry prometheus.Registerer
    
    // Database metrics
    dbConnections    prometheus.Gauge
    dbQueryDuration  *prometheus.HistogramVec
    dbQueryErrors    *prometheus.CounterVec
    
    // Cache metrics
    cacheHits        *prometheus.CounterVec
    cacheMisses      *prometheus.CounterVec
    cacheOperations  *prometheus.HistogramVec
    
    // Message queue metrics
    mqMessagesProduced *prometheus.CounterVec
    mqMessagesConsumed *prometheus.CounterVec
    mqProcessingTime   *prometheus.HistogramVec
}

func NewMetrics(registry prometheus.Registerer) *Metrics {
    m := &Metrics{
        registry: registry,
        
        dbConnections: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "gochat_db_connections_active",
            Help: "Number of active database connections",
        }),
        
        dbQueryDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "gochat_db_query_duration_seconds",
                Help: "Database query duration in seconds",
            },
            []string{"operation", "table"},
        ),
        
        // ... other metrics
    }
    
    // Register all metrics
    registry.MustRegister(
        m.dbConnections,
        m.dbQueryDuration,
        // ... other metrics
    )
    
    return m
}
```