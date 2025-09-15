package kafka

// Config 是 kafka 组件的配置结构体。
type Config struct {
	// Brokers 是 Kafka 集群的地址列表
	Brokers []string `json:"brokers"`
	// SecurityProtocol 安全协议，如 "PLAINTEXT", "SASL_PLAINTEXT", "SASL_SSL"
	SecurityProtocol string `json:"securityProtocol"`
	// SASLMechanism SASL 认证机制，如 "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"
	SASLMechanism string `json:"saslMechanism,omitempty"`
	// SASLUsername SASL 用户名
	SASLUsername string `json:"saslUsername,omitempty"`
	// SASLPassword SASL 密码
	SASLPassword string `json:"saslPassword,omitempty"`
	// ProducerConfig 生产者专用配置
	ProducerConfig *ProducerConfig `json:"producerConfig,omitempty"`
	// ConsumerConfig 消费者专用配置
	ConsumerConfig *ConsumerConfig `json:"consumerConfig,omitempty"`
}

// ProducerConfig 定义生产者的专用配置
type ProducerConfig struct {
	// Acks 确认级别: 0, 1, -1(all)
	Acks int `json:"acks"`
	// RetryMax 最大重试次数
	RetryMax int `json:"retryMax"`
	// BatchSize 批处理大小（字节）
	BatchSize int `json:"batchSize"`
	// LingerMs 延迟发送时间(毫秒)
	LingerMs int `json:"lingerMs"`
	// DeliveryTimeoutMs 消息传递超时时间(毫秒)
	DeliveryTimeoutMs int `json:"deliveryTimeoutMs"`
	// RequestTimeoutMs 请求超时时间(毫秒)
	RequestTimeoutMs int `json:"requestTimeoutMs"`
	// MaxInFlightRequestsPerBroker 每个broker最大飞行中请求数
	MaxInFlightRequestsPerBroker int `json:"maxInFlightRequestsPerBroker"`
	// EnableIdempotence 是否启用幂等性
	EnableIdempotence bool `json:"enableIdempotence"`
	// Compression 压缩算法: "none", "gzip", "snappy", "lz4", "zstd"
	Compression string `json:"compression"`
	// MaxBufferedRecords 最大缓冲记录数
	MaxBufferedRecords int `json:"maxBufferedRecords"`
	// MaxBufferedBytes 最大缓冲字节数
	MaxBufferedBytes int `json:"maxBufferedBytes"`
	// UnknownTopicRetries 未知主题重试次数
	UnknownTopicRetries int `json:"unknownTopicRetries"`
}

// ConsumerConfig 定义消费者的专用配置
type ConsumerConfig struct {
	// AutoOffsetReset 偏移量重置策略: "earliest", "latest", "none"
	AutoOffsetReset string `json:"autoOffsetReset"`
	// EnableAutoCommit 是否启用自动提交偏移量
	EnableAutoCommit bool `json:"enableAutoCommit"`
	// AutoCommitIntervalMs 自动提交间隔(毫秒)
	AutoCommitIntervalMs int `json:"autoCommitIntervalMs"`
	// SessionTimeoutMs 会话超时时间(毫秒)
	SessionTimeoutMs int `json:"sessionTimeoutMs"`
	// HeartbeatIntervalMs 心跳间隔(毫秒)
	HeartbeatIntervalMs int `json:"heartbeatIntervalMs"`
	// RebalanceTimeoutMs 重平衡超时时间(毫秒)
	RebalanceTimeoutMs int `json:"rebalanceTimeoutMs"`
	// FetchMinBytes 最小拉取字节数
	FetchMinBytes int `json:"fetchMinBytes"`
	// FetchMaxBytes 最大拉取字节数
	FetchMaxBytes int `json:"fetchMaxBytes"`
	// FetchMaxWaitMs 最大等待时间(毫秒)
	FetchMaxWaitMs int `json:"fetchMaxWaitMs"`
	// MaxPollRecords 最大拉取记录数
	MaxPollRecords int `json:"maxPollRecords"`
	// MaxPartitionFetchBytes 最大分区拉取字节数
	MaxPartitionFetchBytes int `json:"maxPartitionFetchBytes"`
	// EnableAutoCommitOnClose 关闭时是否自动提交
	EnableAutoCommitOnClose bool `json:"enableAutoCommitOnClose"`
	// CheckCRCs 是否检查CRC校验
	CheckCRCs bool `json:"checkCRCs"`
	// ClientID 客户端ID
	ClientID string `json:"clientId"`
}

// GetDefaultConfig 返回默认的 kafka 配置。
// 开发环境：较少的重试次数，较小的批处理大小
// 生产环境：较多的重试次数，较大的批处理大小，更强的持久性保证
func GetDefaultConfig(env string) *Config {
	if env == "production" {
		return &Config{
			Brokers:          []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
			SecurityProtocol: "SASL_SSL",
			ProducerConfig: &ProducerConfig{
				Acks:                       -1,
				RetryMax:                   10,
				BatchSize:                  65536,
				LingerMs:                   10,
				DeliveryTimeoutMs:          30000,
				RequestTimeoutMs:           30000,
				MaxInFlightRequestsPerBroker: 5,
				EnableIdempotence:          true,
				Compression:                "lz4",
				MaxBufferedRecords:         10000,
				MaxBufferedBytes:           33554432, // 32MB
				UnknownTopicRetries:        3,
			},
			ConsumerConfig: &ConsumerConfig{
				AutoOffsetReset:          "earliest",
				EnableAutoCommit:         true,
				AutoCommitIntervalMs:     5000,
				SessionTimeoutMs:         30000,
				HeartbeatIntervalMs:       3000,
				RebalanceTimeoutMs:        60000,
				FetchMinBytes:             1,
				FetchMaxBytes:             10485760, // 10MB
				FetchMaxWaitMs:            5000,
				MaxPollRecords:            500,
				MaxPartitionFetchBytes:    1048576, // 1MB
				EnableAutoCommitOnClose:   true,
				CheckCRCs:                 true,
				ClientID:                  "kafka-consumer",
			},
		}
	}

	// development environment
	return &Config{
		Brokers:          []string{"localhost:9092"},
		SecurityProtocol: "PLAINTEXT",
		ProducerConfig: &ProducerConfig{
			Acks:                       1,
			RetryMax:                   3,
			BatchSize:                  16384,
			LingerMs:                   5,
			DeliveryTimeoutMs:          10000,
			RequestTimeoutMs:           10000,
			MaxInFlightRequestsPerBroker: 5,
			EnableIdempotence:          false,
			Compression:                "none",
			MaxBufferedRecords:         1000,
			MaxBufferedBytes:           3355443, // 3.2MB
			UnknownTopicRetries:        2,
		},
		ConsumerConfig: &ConsumerConfig{
			AutoOffsetReset:          "latest",
			EnableAutoCommit:         true,
			AutoCommitIntervalMs:     1000,
			SessionTimeoutMs:         10000,
			HeartbeatIntervalMs:       3000,
			RebalanceTimeoutMs:        30000,
			FetchMinBytes:             1,
			FetchMaxBytes:             1048576, // 1MB
			FetchMaxWaitMs:            1000,
			MaxPollRecords:            100,
			MaxPartitionFetchBytes:    524288, // 512KB
			EnableAutoCommitOnClose:   true,
			CheckCRCs:                 false,
			ClientID:                  "kafka-consumer-dev",
		},
	}
}