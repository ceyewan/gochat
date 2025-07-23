package internal

import "time"

// Config 定义了 metrics 和 tracing 系统的内部配置结构。
//
// 这是内部配置结构体，与公共 API 中的 Config 保持一致，
// 但添加了 mapstructure 标签以支持从配置文件中加载。
//
// 该结构体包含了初始化 OpenTelemetry 组件所需的所有参数：
//   - 服务标识和资源信息
//   - Exporter 配置（Jaeger、Zipkin、Prometheus 等）
//   - 采样策略和比例设置
//   - 性能调优参数
type Config struct {
	// ServiceName 定义服务的唯一标识名称。
	//
	// 用于创建 OpenTelemetry Resource，标识 traces 和 metrics 的来源。
	// 这个名称会出现在所有的可观测性数据中，应该具有唯一性和描述性。
	ServiceName string `mapstructure:"service_name"`

	// ExporterType 指定 trace 数据的导出器类型。
	//
	// 支持的导出器类型：
	//   - "jaeger": 使用 Jaeger collector 导出器
	//   - "zipkin": 使用 Zipkin HTTP 导出器
	//   - "stdout": 使用标准输出导出器（调试用）
	ExporterType string `mapstructure:"exporter_type"`

	// ExporterEndpoint 指定 trace exporter 的目标端点地址。
	//
	// 不同类型导出器需要不同格式的端点：
	//   - Jaeger: HTTP collector 端点，如 "http://jaeger:14268/api/traces"
	//   - Zipkin: HTTP API 端点，如 "http://zipkin:9411/api/v2/spans"
	//   - stdout: 忽略此配置
	ExporterEndpoint string `mapstructure:"exporter_endpoint"`

	// PrometheusListenAddr 指定 Prometheus metrics HTTP 服务器的监听地址。
	//
	// 如果配置了有效地址，会启动一个内置的 HTTP 服务器，
	// 在 /metrics 路径提供 Prometheus 格式的指标数据。
	//
	// 地址格式：
	//   - ":port": 监听所有接口的指定端口
	//   - "host:port": 监听指定主机和端口
	//   - "": 禁用 Prometheus 服务器
	PrometheusListenAddr string `mapstructure:"prometheus_listen_addr"`

	// SamplerType 指定 OpenTelemetry trace 采样器的类型。
	//
	// 采样器决定哪些 trace 会被记录和导出：
	//   - "always_on": AlwaysSample，记录所有 trace
	//   - "always_off": NeverSample，不记录任何 trace
	//   - "trace_id_ratio": TraceIDRatioBased，基于 trace ID 进行概率采样
	SamplerType string `mapstructure:"sampler_type"`

	// SamplerRatio 指定基于比例的采样率。
	//
	// 仅当 SamplerType 为 "trace_id_ratio" 时生效。
	// 取值范围：0.0（不采样）到 1.0（全采样）
	//
	// 采样是基于 trace ID 进行的，确保同一个 trace 的所有 span
	// 要么全部被采样，要么全部不被采样，保证链路的完整性。
	SamplerRatio float64 `mapstructure:"sampler_ratio"`

	// SlowRequestThreshold 定义慢请求的判定阈值。
	//
	// 当请求处理时间超过此阈值时，拦截器会：
	//   - 在日志中标记为慢请求
	//   - 使用更高的日志级别（通常是 Warn）
	//   - 添加特殊的属性标签便于监控
	//
	// 该配置有助于识别性能问题和优化热点。
	SlowRequestThreshold time.Duration `mapstructure:"slow_request_threshold"`
}
