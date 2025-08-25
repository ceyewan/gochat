---
inclusion: fileMatch
fileMatchPattern: "im-*/**/*.go"
---

# GoChat 微服务模式

## 服务实现模式

### gRPC 服务结构
实现 gRPC 服务时，遵循这个标准结构：

```go
// Service interface definition
type ServiceInterface interface {
    Method(ctx context.Context, req *pb.Request) (*pb.Response, error)
}

// Service implementation
type serviceImpl struct {
    logger *zap.Logger
    config *config.Config
    // dependencies
}

// Constructor with dependency injection
func NewService(logger *zap.Logger, config *config.Config) ServiceInterface {
    return &serviceImpl{
        logger: logger,
        config: config,
    }
}

// Method implementation with proper error handling
func (s *serviceImpl) Method(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Input validation
    if err := validateRequest(req); err != nil {
        s.logger.Error("invalid request", zap.Error(err))
        return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
    }
    
    // Business logic with tracing
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("method", "Method"))
    
    // Implementation
    result, err := s.processRequest(ctx, req)
    if err != nil {
        s.logger.Error("failed to process request", zap.Error(err))
        return nil, status.Errorf(codes.Internal, "processing failed: %v", err)
    }
    
    return result, nil
}
```

### Kafka 消费者模式
实现带有适当错误处理和优雅关闭的 Kafka 消费者：

```go
type Consumer struct {
    reader *kafka.Reader
    logger *zap.Logger
    processor MessageProcessor
}

func (c *Consumer) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := c.reader.ReadMessage(ctx)
            if err != nil {
                c.logger.Error("failed to read message", zap.Error(err))
                continue
            }
            
            if err := c.processor.Process(ctx, msg); err != nil {
                c.logger.Error("failed to process message", 
                    zap.Error(err),
                    zap.String("topic", msg.Topic),
                    zap.Int("partition", msg.Partition),
                    zap.Int64("offset", msg.Offset))
                // Implement retry logic or dead letter queue
                continue
            }
        }
    }
}
```

### 仓储模式
实现带有适当缓存和错误处理的数据访问：

```go
type Repository interface {
    Get(ctx context.Context, id string) (*Entity, error)
    Save(ctx context.Context, entity *Entity) error
}

type repositoryImpl struct {
    db    *gorm.DB
    cache *redis.Client
    logger *zap.Logger
}

func (r *repositoryImpl) Get(ctx context.Context, id string) (*Entity, error) {
    // Try cache first (Cache-Aside pattern)
    cacheKey := fmt.Sprintf("entity:%s", id)
    cached, err := r.cache.Get(ctx, cacheKey).Result()
    if err == nil {
        var entity Entity
        if err := json.Unmarshal([]byte(cached), &entity); err == nil {
            return &entity, nil
        }
    }
    
    // Fallback to database
    var entity Entity
    if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrEntityNotFound
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    // Update cache (ignore cache errors)
    if data, err := json.Marshal(entity); err == nil {
        r.cache.Set(ctx, cacheKey, data, time.Hour).Err()
    }
    
    return &entity, nil
}
```

## 消息处理模式

### 幂等性实现
使用 Redis 确保消息处理的幂等性：

```go
func (p *MessageProcessor) ProcessMessage(ctx context.Context, msg *Message) error {
    // Check for duplicate processing
    dedupKey := fmt.Sprintf("msg_dedup:%s", msg.ClientMsgID)
    exists, err := p.redis.SetNX(ctx, dedupKey, "1", time.Minute).Result()
    if err != nil {
        p.logger.Warn("failed to check deduplication", zap.Error(err))
        // Continue processing (fail-open for availability)
    } else if !exists {
        p.logger.Info("duplicate message detected", zap.String("client_msg_id", msg.ClientMsgID))
        return nil // Already processed
    }
    
    // Process message
    return p.doProcessMessage(ctx, msg)
}
```

### 消息分发策略
基于群组大小实现智能消息分发：

```go
func (l *LogicService) DistributeMessage(ctx context.Context, msg *Message) error {
    switch msg.ConversationType {
    case "single":
        return l.distributeSingleMessage(ctx, msg)
    case "group":
        groupInfo, err := l.repo.GetGroupInfo(ctx, msg.GroupID)
        if err != nil {
            return fmt.Errorf("failed to get group info: %w", err)
        }
        
        if groupInfo.MemberCount <= 500 {
            // Small group: real-time fanout
            return l.distributeSmallGroupMessage(ctx, msg, groupInfo)
        } else {
            // Large group: async processing
            return l.scheduleAsyncDistribution(ctx, msg, groupInfo)
        }
    case "world":
        return l.distributeWorldMessage(ctx, msg)
    default:
        return fmt.Errorf("unknown conversation type: %s", msg.ConversationType)
    }
}
```

## 错误处理模式

### 结构化错误类型
定义领域特定的错误类型：

```go
type ErrorCode int

const (
    ErrCodeInvalidInput ErrorCode = iota + 1000
    ErrCodeUserNotFound
    ErrCodePermissionDenied
    ErrCodeRateLimited
)

type DomainError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

func (e *DomainError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func NewDomainError(code ErrorCode, message string, cause error) *DomainError {
    return &DomainError{
        Code:    code,
        Message: message,
        Cause:   cause,
    }
}
```

### 熔断器模式
为外部服务调用实现熔断器：

```go
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    failures    int
    lastFailure time.Time
    state       string // "closed", "open", "half-open"
    mutex       sync.RWMutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mutex.RLock()
    state := cb.state
    cb.mutex.RUnlock()
    
    if state == "open" {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.mutex.Lock()
            cb.state = "half-open"
            cb.mutex.Unlock()
        } else {
            return errors.New("circuit breaker is open")
        }
    }
    
    err := fn()
    
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }
    
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

## 配置模式

### 服务配置结构
标准化所有服务的配置：

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    Kafka    KafkaConfig    `yaml:"kafka"`
    Etcd     EtcdConfig     `yaml:"etcd"`
    Logging  LoggingConfig  `yaml:"logging"`
    Tracing  TracingConfig  `yaml:"tracing"`
}

type ServerConfig struct {
    Host         string        `yaml:"host"`
    Port         int           `yaml:"port"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
}

// Load configuration with validation
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &config, nil
}
```

## 健康检查模式

### 服务健康监控
实现全面的健康检查：

```go
type HealthChecker struct {
    db    *gorm.DB
    redis *redis.Client
    kafka *kafka.Writer
}

func (h *HealthChecker) Check(ctx context.Context) error {
    // Check database connectivity
    if err := h.checkDatabase(ctx); err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }
    
    // Check Redis connectivity
    if err := h.checkRedis(ctx); err != nil {
        return fmt.Errorf("redis health check failed: %w", err)
    }
    
    // Check Kafka connectivity
    if err := h.checkKafka(ctx); err != nil {
        return fmt.Errorf("kafka health check failed: %w", err)
    }
    
    return nil
}

func (h *HealthChecker) checkDatabase(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    var result int
    return h.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
}
```