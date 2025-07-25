package idgen

import (
	"context"
	"testing"
)

func TestUUIDGenerator(t *testing.T) {
	ctx := context.Background()

	// 测试 UUID v4 生成器
	config := &UUIDConfig{
		Version:   4,
		Format:    "standard",
		UpperCase: false,
	}

	generator, err := NewUUIDGenerator(config)
	if err != nil {
		t.Fatalf("创建 UUID 生成器失败: %v", err)
	}

	// 测试生成 UUID v4
	uuid, err := generator.GenerateV4(ctx)
	if err != nil {
		t.Fatalf("生成 UUID v4 失败: %v", err)
	}

	if len(uuid) != 36 {
		t.Errorf("UUID 长度不正确，期望 36，实际 %d", len(uuid))
	}

	// 测试验证
	if !generator.Validate(uuid) {
		t.Errorf("UUID 验证失败: %s", uuid)
	}

	// 测试生成 UUID v7
	uuid7, err := generator.GenerateV7(ctx)
	if err != nil {
		t.Fatalf("生成 UUID v7 失败: %v", err)
	}

	if len(uuid7) != 36 {
		t.Errorf("UUID v7 长度不正确，期望 36，实际 %d", len(uuid7))
	}

	// 测试验证 UUID v7
	if !generator.Validate(uuid7) {
		t.Errorf("UUID v7 验证失败: %s", uuid7)
	}

	t.Logf("生成的 UUID v4: %s", uuid)
	t.Logf("生成的 UUID v7: %s", uuid7)
}

func TestUUIDGeneratorSimpleFormat(t *testing.T) {
	ctx := context.Background()

	// 测试简单格式
	config := &UUIDConfig{
		Version:   4,
		Format:    "simple",
		UpperCase: true,
	}

	generator, err := NewUUIDGenerator(config)
	if err != nil {
		t.Fatalf("创建 UUID 生成器失败: %v", err)
	}

	uuid, err := generator.GenerateV4(ctx)
	if err != nil {
		t.Fatalf("生成 UUID 失败: %v", err)
	}

	if len(uuid) != 32 {
		t.Errorf("简单格式 UUID 长度不正确，期望 32，实际 %d", len(uuid))
	}

	// 检查是否为大写
	for _, c := range uuid {
		if c >= 'a' && c <= 'f' {
			t.Errorf("UUID 应该是大写格式，但包含小写字符: %s", uuid)
			break
		}
	}

	t.Logf("生成的简单格式 UUID: %s", uuid)
}
