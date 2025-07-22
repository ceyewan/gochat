package idgen

import (
	"context"
	"testing"
)

func TestSnowflakeGenerator(t *testing.T) {
	ctx := context.Background()

	// 测试自动节点 ID 配置
	config := &SnowflakeConfig{
		NodeID:     0,
		AutoNodeID: true,
		Epoch:      1288834974657, // Twitter 雪花算法起始时间
	}

	generator, err := NewSnowflakeGenerator(config)
	if err != nil {
		t.Fatalf("创建雪花算法生成器失败: %v", err)
	}
	defer generator.Close()

	// 测试生成 int64 ID
	id1, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成雪花 ID 失败: %v", err)
	}

	id2, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成第二个雪花 ID 失败: %v", err)
	}

	// 验证 ID 唯一性
	if id1 == id2 {
		t.Errorf("生成的 ID 应该唯一，但得到相同的 ID: %d", id1)
	}

	// 验证 ID 为正数
	if id1 <= 0 || id2 <= 0 {
		t.Errorf("雪花 ID 应该为正数，得到: %d, %d", id1, id2)
	}

	// 测试生成字符串 ID
	idStr, err := generator.GenerateString(ctx)
	if err != nil {
		t.Fatalf("生成字符串雪花 ID 失败: %v", err)
	}

	if idStr == "" {
		t.Error("字符串 ID 不应该为空")
	}

	// 测试获取节点 ID
	nodeID := generator.GetNodeID()
	if nodeID < 0 || nodeID > 1023 {
		t.Errorf("节点 ID 应该在 0-1023 范围内，得到: %d", nodeID)
	}

	// 测试解析 ID
	timestamp, parsedNodeID, sequence := generator.ParseID(id1)
	if timestamp <= 0 {
		t.Errorf("解析的时间戳应该为正数，得到: %d", timestamp)
	}
	if parsedNodeID < 0 || parsedNodeID > 1023 {
		t.Errorf("解析的节点 ID 应该在 0-1023 范围内，得到: %d", parsedNodeID)
	}
	if sequence < 0 || sequence > 4095 {
		t.Errorf("解析的序列号应该在 0-4095 范围内，得到: %d", sequence)
	}

	t.Logf("生成的雪花 ID: %d, %d, %s", id1, id2, idStr)
	t.Logf("节点 ID: %d", nodeID)
	t.Logf("解析结果 - 时间戳: %d, 节点ID: %d, 序列号: %d", timestamp, parsedNodeID, sequence)
}

func TestSnowflakeGeneratorWithFixedNodeID(t *testing.T) {
	ctx := context.Background()

	// 测试固定节点 ID 配置
	config := &SnowflakeConfig{
		NodeID:     123,
		AutoNodeID: false,
		Epoch:      1288834974657,
	}

	generator, err := NewSnowflakeGenerator(config)
	if err != nil {
		t.Fatalf("创建固定节点 ID 雪花算法生成器失败: %v", err)
	}
	defer generator.Close()

	// 验证节点 ID
	nodeID := generator.GetNodeID()
	if nodeID != 123 {
		t.Errorf("节点 ID 应该为 123，得到: %d", nodeID)
	}

	// 生成 ID 并解析
	id, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成雪花 ID 失败: %v", err)
	}

	_, parsedNodeID, _ := generator.ParseID(id)
	if parsedNodeID != 123 {
		t.Errorf("解析的节点 ID 应该为 123，得到: %d", parsedNodeID)
	}

	t.Logf("固定节点 ID 测试 - 生成的 ID: %d, 节点 ID: %d", id, nodeID)
}

func TestSnowflakeGeneratorType(t *testing.T) {
	config := &SnowflakeConfig{
		NodeID:     1,
		AutoNodeID: false,
		Epoch:      1288834974657,
	}

	generator, err := NewSnowflakeGenerator(config)
	if err != nil {
		t.Fatalf("创建雪花算法生成器失败: %v", err)
	}
	defer generator.Close()

	// 测试生成器类型
	if generator.Type() != SnowflakeType {
		t.Errorf("生成器类型应该为 %s，实际为 %s", SnowflakeType, generator.Type())
	}

	t.Logf("生成器类型: %s", generator.Type())
}

func TestSnowflakeGeneratorConcurrency(t *testing.T) {
	ctx := context.Background()

	config := &SnowflakeConfig{
		NodeID:     1,
		AutoNodeID: false,
		Epoch:      1288834974657,
	}

	generator, err := NewSnowflakeGenerator(config)
	if err != nil {
		t.Fatalf("创建雪花算法生成器失败: %v", err)
	}
	defer generator.Close()

	// 并发生成 ID 测试
	const numGoroutines = 10
	const numIDsPerGoroutine = 100
	
	idChan := make(chan int64, numGoroutines*numIDsPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIDsPerGoroutine; j++ {
				id, err := generator.GenerateInt64(ctx)
				if err != nil {
					t.Errorf("并发生成 ID 失败: %v", err)
					return
				}
				idChan <- id
			}
		}()
	}

	// 收集所有 ID
	ids := make(map[int64]bool)
	for i := 0; i < numGoroutines*numIDsPerGoroutine; i++ {
		id := <-idChan
		if ids[id] {
			t.Errorf("发现重复的 ID: %d", id)
		}
		ids[id] = true
	}

	t.Logf("并发测试完成，生成了 %d 个唯一 ID", len(ids))
}

// 测试向后兼容性
func TestBackwardCompatibility(t *testing.T) {
	// 测试原有的 GetSnowflakeID 函数是否仍然可用
	id := GetSnowflakeID()
	if id <= 0 {
		t.Errorf("GetSnowflakeID 应该返回正数，得到: %d", id)
	}

	// 多次调用确保唯一性
	id2 := GetSnowflakeID()
	if id == id2 {
		t.Errorf("GetSnowflakeID 应该生成唯一 ID，但得到相同的 ID: %d", id)
	}

	t.Logf("向后兼容性测试 - 生成的 ID: %d, %d", id, id2)
}
