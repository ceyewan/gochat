package db_test

import (
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultConfig(t *testing.T) {
	// 测试开发环境配置
	t.Run("Development", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")

		assert.Equal(t, "mysql", cfg.Driver)
		assert.Equal(t, "root:mysql@tcp(localhost:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local", cfg.DSN)
		assert.Equal(t, 25, cfg.MaxOpenConns)
		assert.Equal(t, 5, cfg.MaxIdleConns)
		assert.Equal(t, 5*time.Minute, cfg.ConnMaxLifetime)
		assert.Equal(t, 30*time.Minute, cfg.ConnMaxIdleTime)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, 100*time.Millisecond, cfg.SlowThreshold)
		assert.False(t, cfg.EnableMetrics)
		assert.False(t, cfg.EnableTracing)
		assert.True(t, cfg.AutoCreateDatabase)
		assert.Nil(t, cfg.Sharding) // 默认不分片
	})

	// 测试生产环境配置
	t.Run("Production", func(t *testing.T) {
		cfg := db.GetDefaultConfig("production")

		assert.Equal(t, "mysql", cfg.Driver)
		assert.Equal(t, "", cfg.DSN) // 生产环境需要手动设置
		assert.Equal(t, 100, cfg.MaxOpenConns)
		assert.Equal(t, 10, cfg.MaxIdleConns)
		assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
		assert.Equal(t, 30*time.Minute, cfg.ConnMaxIdleTime)
		assert.Equal(t, "warn", cfg.LogLevel)
		assert.Equal(t, 500*time.Millisecond, cfg.SlowThreshold)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.EnableTracing)
		assert.False(t, cfg.AutoCreateDatabase)
		assert.Nil(t, cfg.Sharding)
	})

	// 测试未知环境（默认为开发环境）
	t.Run("UnknownEnvironment", func(t *testing.T) {
		cfg := db.GetDefaultConfig("unknown")
		defaultCfg := db.GetDefaultConfig("development")

		assert.Equal(t, defaultCfg.Driver, cfg.Driver)
		assert.Equal(t, defaultCfg.MaxOpenConns, cfg.MaxOpenConns)
		assert.Equal(t, defaultCfg.MaxIdleConns, cfg.MaxIdleConns)
		assert.Equal(t, defaultCfg.LogLevel, cfg.LogLevel)
		assert.Equal(t, defaultCfg.SlowThreshold, cfg.SlowThreshold)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("DefaultConfigUsesDevelopment", func(t *testing.T) {
		cfg := db.DefaultConfig()
		devCfg := db.GetDefaultConfig("development")

		assert.Equal(t, devCfg.Driver, cfg.Driver)
		assert.Equal(t, devCfg.MaxOpenConns, cfg.MaxOpenConns)
		assert.Equal(t, devCfg.MaxIdleConns, cfg.MaxIdleConns)
		assert.Equal(t, devCfg.LogLevel, cfg.LogLevel)
		assert.Equal(t, devCfg.SlowThreshold, cfg.SlowThreshold)
	})
}

func TestNewShardingConfig(t *testing.T) {
	t.Run("CreateBasicShardingConfig", func(t *testing.T) {
		shardingKey := "user_id"
		numberOfShards := 16

		cfg := db.NewShardingConfig(shardingKey, numberOfShards)

		require.NotNil(t, cfg)
		assert.Equal(t, shardingKey, cfg.ShardingKey)
		assert.Equal(t, numberOfShards, cfg.NumberOfShards)
		assert.NotNil(t, cfg.Tables)
		assert.Empty(t, cfg.Tables)
	})

	t.Run("CreateShardingConfigWithTables", func(t *testing.T) {
		cfg := db.NewShardingConfig("user_id", 8)

		// 添加分片表
		cfg.Tables["users"] = &db.TableShardingConfig{}
		cfg.Tables["orders"] = &db.TableShardingConfig{
			NumberOfShards: 4, // 覆盖全局分片数
		}

		assert.Len(t, cfg.Tables, 2)
		assert.NotNil(t, cfg.Tables["users"])
		assert.Equal(t, 4, cfg.Tables["orders"].NumberOfShards)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "user:pass@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"

		err := db.ValidateConfig(&cfg)
		assert.NoError(t, err)
	})

	t.Run("EmptyDSN", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = ""

		err := db.ValidateConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DSN cannot be empty")
	})

	t.Run("UnsupportedDriver", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.Driver = "postgres"

		err := db.ValidateConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported driver")
	})

	t.Run("AutoCorrectInvalidValues", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.MaxOpenConns = -1
		cfg.MaxIdleConns = -1
		cfg.ConnMaxLifetime = 0
		cfg.ConnMaxIdleTime = 0
		cfg.LogLevel = ""
		cfg.SlowThreshold = 0

		// 验证配置会自动修正无效值（通过比较验证前后的值）
		originalOpenConns := cfg.MaxOpenConns
		originalIdleConns := cfg.MaxIdleConns
		originalConnMaxLifetime := cfg.ConnMaxLifetime
		originalConnMaxIdleTime := cfg.ConnMaxIdleTime
		originalLogLevel := cfg.LogLevel
		originalSlowThreshold := cfg.SlowThreshold

		err := db.ValidateConfig(&cfg)
		assert.NoError(t, err)

		// 验证配置被正确修正
		assert.Equal(t, 25, cfg.MaxOpenConns, "MaxOpenConns 应该被修正为默认值")
		assert.Equal(t, 10, cfg.MaxIdleConns, "MaxIdleConns 应该被修正为默认值")
		assert.Equal(t, time.Hour, cfg.ConnMaxLifetime, "ConnMaxLifetime 应该被修正为默认值")
		assert.Equal(t, 30*time.Minute, cfg.ConnMaxIdleTime, "ConnMaxIdleTime 应该被修正为默认值")
		assert.Equal(t, "warn", cfg.LogLevel, "LogLevel 应该被修正为默认值")
		assert.Equal(t, 200*time.Millisecond, cfg.SlowThreshold, "SlowThreshold 应该被修正为默认值")

		// 确保值确实发生了变化
		assert.NotEqual(t, originalOpenConns, cfg.MaxOpenConns)
		assert.NotEqual(t, originalIdleConns, cfg.MaxIdleConns)
		assert.NotEqual(t, originalConnMaxLifetime, cfg.ConnMaxLifetime)
		assert.NotEqual(t, originalConnMaxIdleTime, cfg.ConnMaxIdleTime)
		assert.NotEqual(t, originalLogLevel, cfg.LogLevel)
		assert.NotEqual(t, originalSlowThreshold, cfg.SlowThreshold)
	})

	t.Run("IdleConnectionsGreaterThanOpen", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.MaxOpenConns = 10
		cfg.MaxIdleConns = 15 // 空闲连接数大于最大连接数

		err := db.ValidateConfig(&cfg)
		assert.NoError(t, err)
		assert.Equal(t, 10, cfg.MaxIdleConns) // 应该被修正为等于最大连接数
	})
}

func TestValidateShardingConfig(t *testing.T) {
	t.Run("ValidShardingConfig", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.Sharding = db.NewShardingConfig("user_id", 16)

		err := db.ValidateConfig(&cfg)
		assert.NoError(t, err)
	})

	t.Run("EmptyShardingKey", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.Sharding = db.NewShardingConfig("", 16)

		err := db.ValidateConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sharding key cannot be empty")
	})

	t.Run("InvalidNumberOfShards", func(t *testing.T) {
		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.Sharding = db.NewShardingConfig("user_id", 0)

		err := db.ValidateConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "number of shards must be greater than 0")
	})
}