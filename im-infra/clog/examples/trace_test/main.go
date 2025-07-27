package main

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== TraceID Hook 测试 ===")

	// 测试 1: 使用 traceID key
	fmt.Println("\n1. 测试 traceID key:")
	ctx1 := context.WithValue(context.Background(), "traceID", "test-trace-001")
	clog.C(ctx1).Info("测试 traceID", clog.String("test", "1"))

	// 测试 2: 使用 trace_id key
	fmt.Println("\n2. 测试 trace_id key:")
	ctx2 := context.WithValue(context.Background(), "trace_id", "test-trace-002")
	clog.C(ctx2).Info("测试 trace_id", clog.String("test", "2"))

	// 测试 3: 使用默认 TraceID key
	fmt.Println("\n3. 测试默认 TraceID key:")
	ctx3 := context.WithValue(context.Background(), clog.DefaultTraceIDKey, "test-trace-003")
	clog.C(ctx3).Info("测试默认 key", clog.String("test", "3"))

	// 测试 4: 没有 TraceID
	fmt.Println("\n4. 测试没有 TraceID:")
	ctx4 := context.Background()
	clog.C(ctx4).Info("没有 TraceID", clog.String("test", "4"))

	fmt.Println("\n=== 测试完成 ===")
}
