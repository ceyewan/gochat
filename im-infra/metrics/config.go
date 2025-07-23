package metrics

import "time"

// Config 定义了 metrics 和 tracing 系统的公共配置结构。
//
// 该结构体包含了初始化可观测性系统所需的所有参数，包括：
//   - 服务标识信息
//   - 数据导出配置
//   - 采样策略设置
//   - 性能优化参数
//
// 所有配置项都有合理的默认值，可以通过 DefaultConfig() 函数获取。
// 用户可以根据实际需求修改特定的配置项。
type Config struct {
	// ServiceName 定义服务的唯一标识名称。
	//
	// 这是唯一的必填配置项，用于：
	//   - 在分布式追踪中标识服务
	//   - 在监控系统中区分不同服务的指标
	//   - 在日志中添加服务上下文
	//
	// 命名建议：使用小写字母和连字符，如 "im-logic"、"im-gateway"。
	ServiceName string

	// ExporterType 指定 trace 数据的导出器类型。
	//
	// 支持的类型：
	//   - "jaeger": 导出到 Jaeger 分布式追踪系统
	//   - "zipkin": 导出到 Zipkin 分布式追踪系统
	//   - "stdout": 输出到标准输出（主要用于开发和调试）
	//
	// 默认值："stdout"
	ExporterType string

	// ExporterEndpoint 指定 trace exporter 的目标地址。
	//
	// 不同类型的 exporter 需要不同的端点格式：
	//   - Jaeger: "http://jaeger-collector:14268/api/traces"
	//   - Zipkin: "http://zipkin:9411/api/v2/spans"
	//   - stdout: 此配置项被忽略
	//
	// 默认值："http://localhost:14268/api/traces"
	ExporterEndpoint string

	// PrometheusListenAddr 指定 Prometheus metrics 端点的监听地址。
	//
	// 如果设置了有效地址（如 ":9090"），会启动一个 HTTP 服务器
	// 在 /metrics 路径提供 Prometheus 格式的指标数据。
	//
	// 如果为空字符串，则不会启动 Prometheus 服务器。
	//
	// 地址格式示例：
	//   - ":9090": 监听所有接口的 9090 端口
	//   - "localhost:9090": 仅监听本地回环接口
	//   - "": 禁用 Prometheus 端点
	//
	// 默认值：""（禁用）
	PrometheusListenAddr string

	// SamplerType 指定 trace 采样策略类型。
	//
	// 采样策略决定了哪些请求会被记录为 trace：
	//   - "always_on": 记录所有请求（100% 采样）
	//   - "always_off": 不记录任何请求（0% 采样）
	//   - "trace_id_ratio": 基于 trace ID 进行概率采样
	//
	// 在生产环境中，推荐使用 "trace_id_ratio" 以平衡性能和可观测性。
	//
	// 默认值："always_on"
	SamplerType string

	// SamplerRatio 指定采样比例（仅当 SamplerType 为 "trace_id_ratio" 时有效）。
	//
	// 取值范围：0.0 到 1.0
	//   - 0.0: 不采样任何请求
	//   - 0.1: 采样 10% 的请求
	//   - 1.0: 采样所有请求
	//
	// 推荐的生产环境设置：
	//   - 高流量服务：0.01 - 0.1（1% - 10%）
	//   - 中等流量服务：0.1 - 0.5（10% - 50%）
	//   - 低流量服务：1.0（100%）
	//
	// 默认值：1.0
	SamplerRatio float64

	// SlowRequestThreshold 定义慢请求的时间阈值。
	//
	// 当请求处理时间超过此阈值时，会在日志中记录为慢请求，
	// 并可能触发特殊的监控告警。
	//
	// 该配置影响：
	//   - 拦截器的日志级别选择
	//   - 慢请求的识别和标记
	//   - 性能监控和告警
	//
	// 推荐的设置：
	//   - Web API: 500ms - 1s
	//   - RPC 调用: 100ms - 500ms
	//   - 数据库操作: 1s - 5s
	//
	// 默认值：500ms
	SlowRequestThreshold time.Duration
}

// DefaultConfig 返回一个包含合理默认值的新 Config 实例。
//
// 默认配置适用于开发环境和快速原型验证，具有以下特点：
//   - 使用 stdout exporter，便于本地调试
//   - 启用完整采样，确保所有请求都被追踪
//   - 禁用 Prometheus 端点，减少资源消耗
//   - 设置合理的慢请求阈值
//
// 在生产环境中，建议根据实际需求调整以下配置：
//   - ServiceName: 设置为实际的服务名称
//   - ExporterType: 更改为 "jaeger" 或 "zipkin"
//   - PrometheusListenAddr: 启用 metrics 导出
//   - SamplerType: 更改为 "trace_id_ratio"
//   - SamplerRatio: 设置为合适的采样比例
//
// 返回：
//   - *Config: 包含默认配置的 Config 实例
func DefaultConfig() *Config {
	return &Config{
		ServiceName:          "unknown-service",
		ExporterType:         "stdout", // 使用 stdout 便于开发调试
		ExporterEndpoint:     "http://localhost:14268/api/traces",
		PrometheusListenAddr: "", // 默认禁用，通过提供地址如 ":9090" 来启用
		SamplerType:          "always_on",
		SamplerRatio:         1.0,
		SlowRequestThreshold: 500 * time.Millisecond,
	}
}
