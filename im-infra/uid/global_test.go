package idgen

import (
	"context"
	"testing"

	"github.com/ceyewan/gochat/im-infra/cache"
)

func TestGlobalMethods(t *testing.T) {
	ctx := context.Background()

	// 测试全局生成方法（默认使用雪花算法）
	id1, err := GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("全局生成 int64 ID 失败: %v", err)
	}

	id2, err := GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("全局生成第二个 int64 ID 失败: %v", err)
	}

	if id1 == id2 {
		t.Errorf("全局生成的 ID 应该唯一，但得到相同的 ID: %d", id1)
	}

	// 测试全局生成字符串 ID
	idStr, err := GenerateString(ctx)
	if err != nil {
		t.Fatalf("全局生成字符串 ID 失败: %v", err)
	}

	if idStr == "" {
		t.Error("全局生成的字符串 ID 不应该为空")
	}

	// 测试全局生成器类型
	generatorType := Type()
	if generatorType != SnowflakeType {
		t.Errorf("默认生成器类型应该为 %s，实际为 %s", SnowflakeType, generatorType)
	}

	t.Logf("全局方法测试 - ID: %d, %d, %s, 类型: %s", id1, id2, idStr, generatorType)
}

func TestDefaultConfigs(t *testing.T) {
	// 测试默认配置
	defaultCfg := DefaultConfig()
	if defaultCfg.Type != SnowflakeType {
		t.Errorf("默认配置类型应该为 %s，实际为 %s", SnowflakeType, defaultCfg.Type)
	}

	// 测试雪花算法默认配置
	snowflakeCfg := DefaultSnowflakeConfig()
	if snowflakeCfg.Type != SnowflakeType {
		t.Errorf("雪花算法默认配置类型应该为 %s，实际为 %s", SnowflakeType, snowflakeCfg.Type)
	}
	if snowflakeCfg.Snowflake == nil {
		t.Error("雪花算法配置不应该为空")
	}

	// 测试 UUID 默认配置
	uuidCfg := DefaultUUIDConfig()
	if uuidCfg.Type != UUIDType {
		t.Errorf("UUID 默认配置类型应该为 %s，实际为 %s", UUIDType, uuidCfg.Type)
	}
	if uuidCfg.UUID == nil {
		t.Error("UUID 配置不应该为空")
	}

	// 测试 Redis 默认配置
	redisCfg := DefaultRedisConfig()
	if redisCfg.Type != RedisType {
		t.Errorf("Redis 默认配置类型应该为 %s，实际为 %s", RedisType, redisCfg.Type)
	}
	if redisCfg.Redis == nil {
		t.Error("Redis 配置不应该为空")
	}

	t.Logf("默认配置测试通过")
}

func TestCustomGenerators(t *testing.T) {
	ctx := context.Background()

	// 测试自定义雪花算法生成器
	snowflakeGen, err := NewSnowflakeGenerator(&SnowflakeConfig{
		NodeID:     100,
		AutoNodeID: false,
		Epoch:      1288834974657,
	})
	if err != nil {
		t.Fatalf("创建自定义雪花算法生成器失败: %v", err)
	}
	defer snowflakeGen.Close()

	snowflakeID, err := snowflakeGen.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("自定义雪花算法生成器生成 ID 失败: %v", err)
	}

	// 测试自定义 UUID 生成器
	uuidGen, err := NewUUIDGenerator(&UUIDConfig{
		Version:   7,
		Format:    "simple",
		UpperCase: true,
	})
	if err != nil {
		t.Fatalf("创建自定义 UUID 生成器失败: %v", err)
	}
	defer uuidGen.Close()

	uuid, err := uuidGen.GenerateString(ctx)
	if err != nil {
		t.Fatalf("自定义 UUID 生成器生成 ID 失败: %v", err)
	}

	if len(uuid) != 32 {
		t.Errorf("简单格式 UUID 长度应该为 32，实际为 %d", len(uuid))
	}

	t.Logf("自定义生成器测试 - 雪花 ID: %d, UUID: %s", snowflakeID, uuid)
}

func TestConfigValidation(t *testing.T) {
	// 测试无效的生成器类型
	invalidCfg := Config{
		Type: GeneratorType("invalid"),
	}
	err := invalidCfg.Validate()
	if err == nil {
		t.Error("无效的生成器类型应该导致验证失败")
	}

	// 测试雪花算法配置验证
	snowflakeCfg := Config{
		Type: SnowflakeType,
		Snowflake: &SnowflakeConfig{
			NodeID:     2000, // 超出范围
			AutoNodeID: false,
			Epoch:      1288834974657,
		},
	}
	err = snowflakeCfg.Validate()
	if err == nil {
		t.Error("超出范围的节点 ID 应该导致验证失败")
	}

	// 测试 UUID 配置验证
	uuidCfg := Config{
		Type: UUIDType,
		UUID: &UUIDConfig{
			Version: 5, // 不支持的版本
			Format:  "standard",
		},
	}
	err = uuidCfg.Validate()
	if err == nil {
		t.Error("不支持的 UUID 版本应该导致验证失败")
	}

	// 测试 Redis 配置验证
	redisCfg := Config{
		Type: RedisType,
		Redis: &RedisConfig{
			CacheConfig: cache.Config{
				Addr: "localhost:6379",
			},
			KeyPrefix:    "", // 空前缀应该失败
			DefaultKey:   "test",
			Step:         1,
			InitialValue: 1,
		},
	}
	err = redisCfg.Validate()
	if err == nil {
		t.Error("空的键前缀应该导致验证失败")
	}

	t.Logf("配置验证测试通过")
}

func TestFactoryMethods(t *testing.T) {
	ctx := context.Background()

	// 测试通过配置创建生成器
	snowflakeCfg := Config{
		Type: SnowflakeType,
		Snowflake: &SnowflakeConfig{
			NodeID:     50,
			AutoNodeID: false,
			Epoch:      1288834974657,
		},
	}

	generator, err := New(snowflakeCfg)
	if err != nil {
		t.Fatalf("通过配置创建雪花算法生成器失败: %v", err)
	}
	defer generator.Close()

	if generator.Type() != SnowflakeType {
		t.Errorf("生成器类型应该为 %s，实际为 %s", SnowflakeType, generator.Type())
	}

	id, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("工厂方法创建的生成器生成 ID 失败: %v", err)
	}

	if id <= 0 {
		t.Errorf("生成的 ID 应该为正数，得到: %d", id)
	}

	t.Logf("工厂方法测试 - 生成的 ID: %d", id)
}
