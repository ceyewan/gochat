# Kafka/MQ 组件配置文件契约 (kafka.yaml)

**注意：这是一个 Markdown 格式的“契约”文件。在部署时，应将此内容转换为一个真实的 `kafka.yaml` 文件，并由 `coord` 配置中心进行管理。**

```yaml
# Kafka/MQ 组件的统一配置文件
#
# 本文件是 gochat 项目中所有 Kafka 相关配置的唯一真相来源。
# mq 组件将通过 coord 配置中心读取此文件。

# 通用设置，适用于所有客户端
common:
  # Kafka broker 地址列表
  brokers:
    - "kafka-1:9092"
    - "kafka-2:9092"
    - "kafka-3:9092"
  # 客户端ID前缀，最终的 ClientID 会是 <clientIDPrefix>-<serviceName>
  clientIDPrefix: "gochat"
  # 安全协议相关配置 (SASL, SSL)，留空表示不启用
  security:
    sasl:
      mechanism: "" # e.g., "PLAIN", "SCRAM-SHA-256"
      username: ""
      password: ""
    ssl:
      ca_cert_file: ""
      client_cert_file: ""
      client_key_file: ""
      insecure_skip_verify: false

# 生产者的默认配置
# 这些配置可以被特定服务的生产者配置覆盖
producer:
  # 确认级别: 0=不等待, 1=等待leader, -1=等待所有副本
  requiredAcks: -1
  # 是否启用幂等性，强烈建议保持 true
  enableIdempotence: true
  # 压缩算法: "none", "gzip", "snappy", "lz4", "zstd"
  compression: "lz4"
  # 批处理大小 (字节)
  batchSize: 16384 # 16KB
  # 批处理等待时间 (毫秒)，增加此值可提高吞吐量，但会增加延迟
  lingerMs: 5

# 消费者的默认配置
consumer:
  # 自动提交偏移量的间隔
  autoCommitIntervalMs: 5000 # 5 seconds
  # 当没有初始偏移量时，从何处开始消费: "earliest", "latest"
  autoOffsetReset: "latest"
  # 会话超时时间 (毫秒)
  sessionTimeoutMs: 30000
  # 心跳间隔 (毫秒)，应小于 sessionTimeoutMs 的 1/3
  heartbeatIntervalMs: 10000