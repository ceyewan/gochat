package cache

import (
	"context"
	"testing"
	"time"
)

func TestBasicOperations(t *testing.T) {
	ctx := context.Background()

	// Test Set and Get
	err := Set(ctx, "test:key", "test value", time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := Get(ctx, "test:key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value != "test value" {
		t.Errorf("Expected 'test value', got '%s'", value)
	}

	// Test Del
	err = Del(ctx, "test:key")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	// Verify key is deleted
	_, err = Get(ctx, "test:key")
	if err == nil {
		t.Error("Expected error for deleted key, got nil")
	}
}

func TestHashOperations(t *testing.T) {
	ctx := context.Background()
	key := "test:hash"

	// Test HSet and HGet
	err := HSet(ctx, key, "field1", "value1")
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}

	value, err := HGet(ctx, key, "field1")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}

	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Test HGetAll
	HSet(ctx, key, "field2", "value2")
	fields, err := HGetAll(ctx, key)
	if err != nil {
		t.Fatalf("HGetAll failed: %v", err)
	}

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	// Test HExists
	exists, err := HExists(ctx, key, "field1")
	if err != nil {
		t.Fatalf("HExists failed: %v", err)
	}

	if !exists {
		t.Error("Expected field1 to exist")
	}

	// Test HDel
	err = HDel(ctx, key, "field1")
	if err != nil {
		t.Fatalf("HDel failed: %v", err)
	}

	exists, err = HExists(ctx, key, "field1")
	if err != nil {
		t.Fatalf("HExists failed: %v", err)
	}

	if exists {
		t.Error("Expected field1 to not exist after deletion")
	}

	// Cleanup
	Del(ctx, key)
}

func TestSetOperations(t *testing.T) {
	ctx := context.Background()
	key := "test:set"

	// Test SAdd and SMembers
	err := SAdd(ctx, key, "member1", "member2", "member3")
	if err != nil {
		t.Fatalf("SAdd failed: %v", err)
	}

	members, err := SMembers(ctx, key)
	if err != nil {
		t.Fatalf("SMembers failed: %v", err)
	}

	if len(members) != 3 {
		t.Errorf("Expected 3 members, got %d", len(members))
	}

	// Test SIsMember
	isMember, err := SIsMember(ctx, key, "member1")
	if err != nil {
		t.Fatalf("SIsMember failed: %v", err)
	}

	if !isMember {
		t.Error("Expected member1 to be in set")
	}

	// Test SCard
	count, err := SCard(ctx, key)
	if err != nil {
		t.Fatalf("SCard failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	// Test SRem
	err = SRem(ctx, key, "member1")
	if err != nil {
		t.Fatalf("SRem failed: %v", err)
	}

	isMember, err = SIsMember(ctx, key, "member1")
	if err != nil {
		t.Fatalf("SIsMember failed: %v", err)
	}

	if isMember {
		t.Error("Expected member1 to not be in set after removal")
	}

	// Cleanup
	Del(ctx, key)
}

func TestCustomConfig(t *testing.T) {
	// Test config builder
	cfg := NewConfigBuilder().
		Addr("localhost:6379").
		DB(1).
		PoolSize(5).
		KeyPrefix("test").
		Build()

	if cfg.Addr != "localhost:6379" {
		t.Errorf("Expected addr 'localhost:6379', got '%s'", cfg.Addr)
	}

	if cfg.DB != 1 {
		t.Errorf("Expected DB 1, got %d", cfg.DB)
	}

	if cfg.PoolSize != 5 {
		t.Errorf("Expected PoolSize 5, got %d", cfg.PoolSize)
	}

	if cfg.KeyPrefix != "test" {
		t.Errorf("Expected KeyPrefix 'test', got '%s'", cfg.KeyPrefix)
	}

	// Test config validation
	err := ValidateConfig(cfg)
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}
}

func TestPresetConfigs(t *testing.T) {
	// Test development config
	devCfg := DevelopmentConfig()
	if devCfg.KeyPrefix != "dev" {
		t.Errorf("Expected dev KeyPrefix 'dev', got '%s'", devCfg.KeyPrefix)
	}

	// Test production config
	prodCfg := ProductionConfig()
	if prodCfg.KeyPrefix != "prod" {
		t.Errorf("Expected prod KeyPrefix 'prod', got '%s'", prodCfg.KeyPrefix)
	}

	// Test test config
	testCfg := TestConfig()
	if testCfg.KeyPrefix != "test" {
		t.Errorf("Expected test KeyPrefix 'test', got '%s'", testCfg.KeyPrefix)
	}

	if testCfg.DB != 1 {
		t.Errorf("Expected test DB 1, got %d", testCfg.DB)
	}
}

func TestPing(t *testing.T) {
	ctx := context.Background()

	err := Ping(ctx)
	if err != nil {
		t.Logf("Ping failed (Redis may not be running): %v", err)
		// Don't fail the test if Redis is not running
		return
	}

	t.Log("Ping successful")
}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()
	Set(ctx, "bench:key", "bench value", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Get(ctx, "bench:key")
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkSet(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Set(ctx, "bench:key", "bench value", time.Hour)
		if err != nil {
			b.Fatalf("Set failed: %v", err)
		}
	}
}
