package idempotent

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

func TestGlobalMethods(t *testing.T) {
	ctx := context.Background()
	key := "test:global:123"

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	// 测试 Check - 键不存在
	exists, err := Check(ctx, key)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	// 测试 Set - 首次设置
	success, err := Set(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if !success {
		t.Error("Expected first set to succeed")
	}

	// 测试 Check - 键存在
	exists, err = Check(ctx, key)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// 测试 Set - 重复设置
	success, err = Set(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if success {
		t.Error("Expected duplicate set to fail")
	}

	// 测试 TTL
	ttl, err := TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("Unexpected TTL: %v", ttl)
	}

	// 测试 Delete
	err = Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证删除后键不存在
	exists, err = Check(ctx, key)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

func TestSetWithResult(t *testing.T) {
	ctx := context.Background()
	key := "test:result:123"
	result := map[string]interface{}{
		"user_id": 123,
		"status":  "created",
	}

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	// 测试 SetWithResult - 首次设置
	success, err := SetWithResult(ctx, key, result, time.Minute)
	if err != nil {
		t.Fatalf("SetWithResult failed: %v", err)
	}
	if !success {
		t.Error("Expected first SetWithResult to succeed")
	}

	// 测试 GetResult
	cachedResult, err := GetResult(ctx, key)
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if cachedResult == nil {
		t.Error("Expected result to be cached")
	}

	// 验证结果内容
	resultMap, ok := cachedResult.(map[string]interface{})
	if !ok {
		t.Error("Expected result to be a map")
	} else {
		if resultMap["user_id"] != float64(123) { // JSON 反序列化数字为 float64
			t.Errorf("Expected user_id to be 123, got %v", resultMap["user_id"])
		}
		if resultMap["status"] != "created" {
			t.Errorf("Expected status to be 'created', got %v", resultMap["status"])
		}
	}

	// 测试 SetWithResult - 重复设置
	success, err = SetWithResult(ctx, key, result, time.Minute)
	if err != nil {
		t.Fatalf("SetWithResult failed: %v", err)
	}
	if success {
		t.Error("Expected duplicate SetWithResult to fail")
	}
}

func TestExecute(t *testing.T) {
	ctx := context.Background()
	key := "test:execute:123"
	callCount := 0

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	callback := func() (interface{}, error) {
		callCount++
		return map[string]interface{}{
			"call_count": callCount,
			"timestamp":  time.Now().Unix(),
		}, nil
	}

	// 第一次执行
	t.Logf("第一次执行前，检查键是否存在")
	exists1, _ := Check(ctx, key)
	t.Logf("第一次执行前，键存在: %v", exists1)

	result1, err := Execute(ctx, key, time.Minute, callback)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected result from first execution")
	}

	// 验证回调被调用了一次
	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}

	t.Logf("第一次执行后，result1: %+v", result1)

	// 检查键是否存在
	exists2, _ := Check(ctx, key)
	t.Logf("第一次执行后，键存在: %v", exists2)

	// 第二次执行（应该返回缓存结果）
	t.Logf("第二次执行前，callCount: %d", callCount)
	result2, err := Execute(ctx, key, time.Minute, callback)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 == nil {
		t.Error("Expected cached result from second execution")
	}

	t.Logf("第二次执行后，result2: %+v", result2)
	t.Logf("第二次执行后，callCount: %d", callCount)

	// 验证回调没有被再次调用
	if callCount != 1 {
		t.Errorf("Expected callback to be called only once, got %d", callCount)
	}

	// 验证两次结果相同
	result1Map := result1.(map[string]interface{})
	result2Map := result2.(map[string]interface{})
	t.Logf("result1Map: %+v", result1Map)
	t.Logf("result2Map: %+v", result2Map)

	// 由于 JSON 反序列化会将数字类型变为 float64，因此需要进行类型转换后再比较
	callCount1 := result1Map["call_count"].(int)
	callCount2 := int(result2Map["call_count"].(float64))
	if callCount1 != callCount2 {
		t.Errorf("Expected call_count to be %d, got %d", callCount1, callCount2)
	}

	timestamp1 := result1Map["timestamp"].(int64)
	timestamp2 := int64(result2Map["timestamp"].(float64))
	if timestamp1 != timestamp2 {
		t.Errorf("Expected timestamp to be %d, got %d", timestamp1, timestamp2)
	}
}

func TestExecuteSimple(t *testing.T) {
	ctx := context.Background()
	key := "test:execute_simple:123"
	callCount := 0

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	callback := func() error {
		callCount++
		return nil
	}

	// 第一次执行
	err := ExecuteSimple(ctx, key, time.Minute, callback)
	if err != nil {
		t.Fatalf("ExecuteSimple failed: %v", err)
	}

	// 验证回调被调用了一次
	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}

	// 第二次执行（应该跳过）
	err = ExecuteSimple(ctx, key, time.Minute, callback)
	if err != nil {
		t.Fatalf("ExecuteSimple failed: %v", err)
	}

	// 验证回调没有被再次调用
	if callCount != 1 {
		t.Errorf("Expected callback to be called only once, got %d", callCount)
	}
}

func TestCustomClient(t *testing.T) {
	ctx := context.Background()
	key := "test:custom:123"

	// 创建自定义配置
	cfg := NewConfigBuilder().
		KeyPrefix("test").
		DefaultTTL(30 * time.Second).
		CacheConfig(cache.TestConfig()).
		Build()

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create custom client: %v", err)
	}
	defer client.Close()

	// 清理测试数据
	defer func() {
		_ = client.Delete(ctx, key)
	}()

	// 测试自定义客户端的基本操作
	success, err := client.Set(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Custom client Set failed: %v", err)
	}
	if !success {
		t.Error("Expected custom client set to succeed")
	}

	exists, err := client.Check(ctx, key)
	if err != nil {
		t.Fatalf("Custom client Check failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist in custom client")
	}
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()
	key := "test:refresh:123"

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	// 设置键
	success, err := Set(ctx, key, time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if !success {
		t.Error("Expected set to succeed")
	}

	// 获取初始 TTL
	initialTTL, err := TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	// 刷新 TTL
	err = Refresh(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// 获取刷新后的 TTL
	newTTL, err := TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	// 验证 TTL 被延长了
	if newTTL <= initialTTL {
		t.Errorf("Expected TTL to be extended, initial: %v, new: %v", initialTTL, newTTL)
	}
}

func TestConfigValidation(t *testing.T) {
	// 测试无效配置
	cfg := Config{
		KeyPrefix:  "",         // 空前缀应该失败
		DefaultTTL: -time.Hour, // 负数 TTL 应该失败
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected configimpl validation to fail")
	}

	// 测试有效配置
	cfg = DefaultConfig()
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected default configimpl to be valid: %v", err)
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	key := "test:exists:123"

	// 清理测试数据
	defer func() {
		_ = Delete(ctx, key)
	}()

	// 测试不存在的键
	exists, err := Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	// 设置键
	_, err = Set(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 测试存在的键
	exists, err = Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}
}
