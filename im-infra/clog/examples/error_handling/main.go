package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 错误处理示例 ===")

	// 1. 基本错误记录
	demonstrateBasicErrorLogging()

	// 2. 包装错误记录
	demonstrateWrappedErrorLogging()

	// 3. 带上下文的错误记录
	demonstrateContextualErrorLogging()

	// 4. 错误分类和严重级别
	demonstrateErrorClassification()

	// 5. 错误恢复和重试模式
	demonstrateErrorRecoveryPatterns()

	// 6. 业务逻辑错误处理
	demonstrateBusinessLogicErrors()

	// 7. 外部服务错误处理
	demonstrateExternalServiceErrors()

	// 8. 系统资源错误处理
	demonstrateSystemResourceErrors()

	fmt.Println("\n=== 错误处理示例完成 ===")
}

// demonstrateBasicErrorLogging 基本错误记录
func demonstrateBasicErrorLogging() {
	fmt.Println("\n1. 基本错误记录:")

	logger := clog.Module("basic_errors")

	// 简单的错误记录
	err := fmt.Errorf("数据库连接失败")
	logger.Error("无法连接到数据库", clog.Err(err))

	// 带有更多上下文信息的错误记录
	err = fmt.Errorf("用户认证失败: 无效的凭据")
	logger.Error("用户登录失败",
		clog.Err(err),
		clog.String("username", "alice"),
		clog.String("ip_address", "192.168.1.100"),
		clog.String("user_agent", "Mozilla/5.0"))

	// 记录系统级错误
	err = fmt.Errorf("磁盘空间不足")
	logger.Error("系统资源错误",
		clog.Err(err),
		clog.String("resource_type", "disk"),
		clog.String("mount_point", "/var/log"),
		clog.Int("available_mb", 50))
}

// demonstrateWrappedErrorLogging 包装错误记录
func demonstrateWrappedErrorLogging() {
	fmt.Println("\n2. 包装错误记录:")

	logger := clog.Module("wrapped_errors")

	// 多层错误包装
	originalErr := fmt.Errorf("connection refused")
	networkErr := fmt.Errorf("网络连接错误: %w", originalErr)
	serviceErr := fmt.Errorf("支付服务不可用: %w", networkErr)

	logger.Error("调用支付服务失败",
		clog.Err(serviceErr),
		clog.String("service", "payment-gateway"),
		clog.String("endpoint", "https://api.payment.com/charge"),
		clog.String("method", "POST"))

	// 业务层错误包装
	validationErr := fmt.Errorf("字段验证失败: email 格式不正确")
	requestErr := fmt.Errorf("用户请求处理失败: %w", validationErr)

	logger.Error("API请求处理错误",
		clog.Err(requestErr),
		clog.String("endpoint", "/api/users"),
		clog.String("request_id", "req-12345"),
		clog.String("field", "email"),
		clog.String("provided_value", "invalid-email"))
}

// demonstrateContextualErrorLogging 带上下文的错误记录
func demonstrateContextualErrorLogging() {
	fmt.Println("\n3. 带上下文的错误记录:")

	logger := clog.Module("contextual_errors")

	// 模拟一个请求处理流程中的错误
	ctx := context.WithValue(context.Background(), "trace_id", "error-trace-001")

	// 在请求处理的不同阶段记录错误
	logger.InfoContext(ctx, "开始处理用户注册请求",
		clog.String("username", "bob"),
		clog.String("email", "bob@example.com"))

	// 验证阶段错误
	validationError := fmt.Errorf("密码强度不足")
	logger.WarnContext(ctx, "用户输入验证失败",
		clog.Err(validationError),
		clog.String("validation_type", "password"),
		clog.String("requirement", "至少8位包含数字和字母"))

	// 数据库操作错误
	dbError := fmt.Errorf("唯一约束冲突: 用户名已存在")
	logger.ErrorContext(ctx, "数据库操作失败",
		clog.Err(dbError),
		clog.String("operation", "INSERT"),
		clog.String("table", "users"),
		clog.String("constraint", "username_unique"))

	// 请求最终失败
	logger.ErrorContext(ctx, "用户注册流程失败",
		clog.String("reason", "数据库约束冲突"),
		clog.Duration("processing_time", 250*time.Millisecond))
}

// demonstrateErrorClassification 错误分类和严重级别
func demonstrateErrorClassification() {
	fmt.Println("\n4. 错误分类和严重级别:")

	logger := clog.Module("error_classification")

	// 用户错误 (Warn 级别 - 用户可以修正)
	logger.Warn("用户输入错误",
		clog.String("error_type", "user_input"),
		clog.String("field", "phone_number"),
		clog.String("provided_value", "abc123"),
		clog.String("expected_format", "11位数字"),
		clog.String("suggestion", "请输入正确的手机号码"))

	// 业务逻辑错误 (Warn 级别 - 业务规则限制)
	logger.Warn("业务规则违反",
		clog.String("error_type", "business_rule"),
		clog.String("rule", "单日转账限额"),
		clog.Int("attempted_amount", 100000),
		clog.Int("daily_limit", 50000),
		clog.String("user_id", "user-789"))

	// 系统错误 (Error 级别 - 需要技术介入)
	logger.Error("系统配置错误",
		clog.String("error_type", "configuration"),
		clog.String("config_key", "database.host"),
		clog.String("error", "配置项缺失"),
		clog.String("impact", "服务无法启动"))

	// 严重系统错误 (Error 级别 - 需要立即关注)
	logger.Error("严重系统故障",
		clog.String("error_type", "critical_system"),
		clog.String("component", "database_pool"),
		clog.String("error", "所有连接均已失效"),
		clog.Bool("service_degraded", true),
		clog.String("action_required", "立即检查数据库状态"))
}

// demonstrateErrorRecoveryPatterns 错误恢复和重试模式
func demonstrateErrorRecoveryPatterns() {
	fmt.Println("\n5. 错误恢复和重试模式:")

	logger := clog.Module("error_recovery")
	ctx := context.WithValue(context.Background(), "trace_id", "recovery-001")

	// 模拟重试机制
	operation := "call_external_api"
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 模拟外部API调用失败
		err := fmt.Errorf("外部服务暂时不可用")

		if attempt < maxRetries {
			logger.WarnContext(ctx, "操作失败，准备重试",
				clog.Err(err),
				clog.String("operation", operation),
				clog.Int("current_attempt", attempt),
				clog.Int("max_retries", maxRetries),
				clog.Duration("retry_delay", time.Duration(attempt)*time.Second))

			// 模拟重试延迟
			time.Sleep(time.Millisecond * 10)
		} else {
			// 最终失败
			logger.ErrorContext(ctx, "操作最终失败",
				clog.Err(err),
				clog.String("operation", operation),
				clog.Int("total_attempts", maxRetries),
				clog.String("next_action", "转入错误队列"))
		}
	}

	// 降级处理
	logger.InfoContext(ctx, "启用降级处理",
		clog.String("fallback_strategy", "使用缓存数据"),
		clog.String("data_source", "local_cache"),
		clog.Duration("cache_age", 5*time.Minute))

	// 错误恢复成功
	logger.InfoContext(ctx, "错误恢复成功",
		clog.String("recovery_method", "fallback"),
		clog.Bool("service_restored", true),
		clog.Duration("downtime", 30*time.Second))
}

// demonstrateBusinessLogicErrors 业务逻辑错误处理
func demonstrateBusinessLogicErrors() {
	fmt.Println("\n6. 业务逻辑错误处理:")

	logger := clog.Module("business_logic")

	// 库存不足错误
	logger.Warn("商品库存不足",
		clog.String("product_id", "PROD-001"),
		clog.String("product_name", "iPhone 15"),
		clog.Int("requested_quantity", 5),
		clog.Int("available_stock", 2),
		clog.String("user_id", "USER-123"),
		clog.String("action", "更新购物车数量"))

	// 权限不足错误
	logger.Warn("用户权限不足",
		clog.String("user_id", "USER-456"),
		clog.String("requested_action", "删除订单"),
		clog.String("user_role", "customer"),
		clog.String("required_role", "admin"),
		clog.String("resource", "order-789"))

	// 业务状态冲突错误
	logger.Error("业务状态冲突",
		clog.String("entity_type", "order"),
		clog.String("entity_id", "ORDER-001"),
		clog.String("current_status", "已发货"),
		clog.String("attempted_action", "取消订单"),
		clog.String("business_rule", "已发货订单无法取消"))

	// 业务流程超时错误
	logger.Error("业务流程超时",
		clog.String("workflow", "支付确认"),
		clog.String("payment_id", "PAY-123"),
		clog.Duration("timeout_limit", 10*time.Minute),
		clog.Duration("actual_duration", 12*time.Minute),
		clog.String("action", "标记为超时并退款"))
}

// demonstrateExternalServiceErrors 外部服务错误处理
func demonstrateExternalServiceErrors() {
	fmt.Println("\n7. 外部服务错误处理:")

	logger := clog.Module("external_services")
	ctx := context.WithValue(context.Background(), "trace_id", "ext-svc-001")

	// 外部API调用错误
	logger.ErrorContext(ctx, "外部API调用失败",
		clog.String("service_name", "支付宝API"),
		clog.String("endpoint", "https://openapi.alipay.com/gateway.do"),
		clog.String("method", "POST"),
		clog.Int("http_status", 503),
		clog.String("error_code", "SERVICE_UNAVAILABLE"),
		clog.String("error_message", "系统繁忙，请稍后再试"),
		clog.Duration("response_time", 5*time.Second))

	// 第三方服务超时
	logger.ErrorContext(ctx, "第三方服务调用超时",
		clog.String("service_name", "短信服务"),
		clog.String("provider", "阿里云SMS"),
		clog.Duration("timeout_setting", 3*time.Second),
		clog.Duration("actual_duration", 3*time.Second),
		clog.String("phone_number", "138****8888"),
		clog.String("message_type", "验证码"))

	// 外部服务认证错误
	logger.ErrorContext(ctx, "外部服务认证失败",
		clog.String("service_name", "微信支付"),
		clog.String("auth_method", "API密钥"),
		clog.String("error_type", "INVALID_SIGNATURE"),
		clog.String("merchant_id", "1234567890"),
		clog.String("suggestion", "检查API密钥配置"))

	// 外部服务数据格式错误
	logger.ErrorContext(ctx, "外部服务响应格式错误",
		clog.String("service_name", "物流查询API"),
		clog.String("expected_format", "JSON"),
		clog.String("actual_content_type", "text/html"),
		clog.String("tracking_number", "SF1234567890"),
		clog.String("response_preview", "<html><body>系统维护中</body></html>"))
}

// demonstrateSystemResourceErrors 系统资源错误处理
func demonstrateSystemResourceErrors() {
	fmt.Println("\n8. 系统资源错误处理:")

	logger := clog.Module("system_resources")

	// 内存不足错误
	logger.Error("系统内存不足",
		clog.String("resource_type", "memory"),
		clog.Uint64("current_usage_mb", 7680),
		clog.Uint64("total_memory_mb", 8192),
		clog.Float64("usage_percentage", 93.75),
		clog.String("action", "触发内存清理"))

	// 磁盘空间不足错误
	logger.Error("磁盘空间不足",
		clog.String("resource_type", "disk"),
		clog.String("mount_point", "/var/log"),
		clog.Uint64("available_mb", 100),
		clog.Uint64("total_mb", 10240),
		clog.Float64("usage_percentage", 99.0),
		clog.String("action", "清理旧日志文件"))

	// 数据库连接池耗尽
	logger.Error("数据库连接池耗尽",
		clog.String("resource_type", "database_connections"),
		clog.Int("active_connections", 100),
		clog.Int("max_connections", 100),
		clog.Int("queued_requests", 25),
		clog.Duration("average_connection_time", 500*time.Millisecond),
		clog.String("action", "拒绝新连接请求"))

	// CPU使用率过高
	logger.Warn("CPU使用率过高",
		clog.String("resource_type", "cpu"),
		clog.Float64("cpu_usage_percentage", 95.2),
		clog.Int("cpu_cores", 4),
		clog.Float64("load_average_1m", 3.8),
		clog.String("top_process", "java"),
		clog.String("action", "启用请求限流"))

	// 网络连接错误
	logger.Error("网络连接异常",
		clog.String("resource_type", "network"),
		clog.String("target_host", "database.internal"),
		clog.Int("target_port", 5432),
		clog.String("error_type", "connection_timeout"),
		clog.Duration("timeout_duration", 10*time.Second),
		clog.String("network_interface", "eth0"))

	// 文件句柄耗尽
	logger.Error("文件句柄耗尽",
		clog.String("resource_type", "file_descriptors"),
		clog.Int("current_open_files", 65535),
		clog.Int("max_open_files", 65536),
		clog.Float64("usage_percentage", 99.99),
		clog.String("process_name", "app-server"),
		clog.String("action", "强制关闭空闲连接"))
}
