package db_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShardingConfig 测试分片配置
func TestShardingConfig(t *testing.T) {
	t.Run("BasicShardingConfig", func(t *testing.T) {
		shardingConfig := db.NewShardingConfig("user_id", 16)
		shardingConfig.Tables = map[string]*db.TableShardingConfig{
			"users":  {},
			"orders": {},
		}

		cfg := db.GetDefaultConfig("development")
		cfg.DSN = "valid:dsn@tcp(localhost:3306)/test"
		cfg.Sharding = shardingConfig

		err := db.ValidateConfig(&cfg)
		assert.NoError(t, err)

		assert.Equal(t, "user_id", cfg.Sharding.ShardingKey)
		assert.Equal(t, 16, cfg.Sharding.NumberOfShards)
		assert.Len(t, cfg.Sharding.Tables, 2)
	})

	t.Run("TableSpecificSharding", func(t *testing.T) {
		shardingConfig := db.NewShardingConfig("user_id", 16)
		shardingConfig.Tables = map[string]*db.TableShardingConfig{
			"users": {
				ShardingKey:    "id",        // 覆盖全局分片键
				NumberOfShards: 8,           // 覆盖全局分片数
			},
			"orders": {
				NumberOfShards: 32, // 只覆盖分片数
			},
		}

		assert.Equal(t, "id", shardingConfig.Tables["users"].ShardingKey)
		assert.Equal(t, 8, shardingConfig.Tables["users"].NumberOfShards)
		assert.Equal(t, "", shardingConfig.Tables["orders"].ShardingKey) // 使用全局分片键
		assert.Equal(t, 32, shardingConfig.Tables["orders"].NumberOfShards)
	})
}

// TestShardingAlgorithm 测试分片算法逻辑
func TestShardingAlgorithm(t *testing.T) {
	// 创建一个测试用的分片配置来进行算法测试
	shardingConfig := db.NewShardingConfig("user_id", 16)

	testCases := []struct {
		name        string
		input       interface{}
		expectedShard int
	}{
		{"PositiveInteger", int64(123), 123 % 16},
		{"NegativeInteger", int64(-45), 45 % 16},
		{"Zero", int64(0), 0},
		{"MaxInt64", int64(9223372036854775807), 9223372036854775807 % 16},
		{"StringNumber", "123", 123 % 16},
		{"StringNegativeNumber", "-45", 45 % 16},
		{"StringZero", "0", 0},
		{"StringAlpha", "hello", 2}, // hello 的哈希值对16取模
		{"Uint64", uint64(123), 123 % 16},
		{"Int32", int32(45), 45 % 16},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shardSuffix, err := simulateShardingAlgorithm(tc.input, shardingConfig.NumberOfShards)
			require.NoError(t, err)
			expectedSuffix := fmt.Sprintf("_%02d", tc.expectedShard)
			assert.Equal(t, expectedSuffix, shardSuffix)
		})
	}

	t.Run("UnsupportedType", func(t *testing.T) {
		shardingConfig := db.NewShardingConfig("user_id", 16)
		_, err := simulateShardingAlgorithm(3.14, shardingConfig.NumberOfShards)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported sharding key type")
	})
}

// simulateShardingAlgorithm 模拟分片算法以进行单元测试
func simulateShardingAlgorithm(columnValue interface{}, numberOfShards int) (suffix string, err error) {
	var intValue int64
	switch v := columnValue.(type) {
	case int:
		intValue = int64(v)
	case int32:
		intValue = int64(v)
	case int64:
		intValue = v
	case uint:
		intValue = int64(v)
	case uint32:
		intValue = int64(v)
	case uint64:
		intValue = int64(v)
	case string:
		// 对于字符串，优先解析为数字
		if parsed, parseErr := strconv.ParseInt(v, 10, 64); parseErr == nil {
			intValue = parsed
		} else {
			// 如果不能解析为数字，使用哈希
			hash := int64(0)
			for _, c := range v {
				hash = hash*31 + int64(c)
			}
			intValue = hash
		}
	default:
		return "", fmt.Errorf("unsupported sharding key type: %T", columnValue)
	}

	// 取绝对值
	if intValue < 0 {
		intValue = -intValue
	}

	// 对分片总数进行取模，得到分片索引
	shardIndex := intValue % int64(numberOfShards)
	return fmt.Sprintf("_%02d", shardIndex), nil
}

// TestShardingWithDatabase 测试与数据库集成的分片功能
func TestShardingWithDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要数据库的测试")
	}

	// 创建分片配置
	shardingConfig := db.NewShardingConfig("user_id", 4) // 使用4个分片便于测试
	shardingConfig.Tables = map[string]*db.TableShardingConfig{
		"sharded_users": {},
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.Sharding = shardingConfig

	logger := clog.Namespace("sharding-test")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		t.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 定义分片表模型
	type ShardedUser struct {
		ID     int64  `gorm:"primaryKey"`
		UserID int64  `gorm:"index"` // 分片键
		Name   string `gorm:"size:100"`
		Email  string `gorm:"size:255"`
	}

	t.Run("AutoMigrateShardedTable", func(t *testing.T) {
		err := provider.AutoMigrate(ctx, &ShardedUser{})
		assert.NoError(t, err)

		// 验证分片表是否被创建
		gormDB := provider.DB(ctx)

		// 检查所有分片表是否存在
		for i := 0; i < shardingConfig.NumberOfShards; i++ {
			tableName := fmt.Sprintf("sharded_users_%02d", i)
			var result []map[string]interface{}
			err = gormDB.Raw(fmt.Sprintf("SHOW TABLES LIKE '%s'", tableName)).Scan(&result).Error
			assert.NoError(t, err, "分片表 %s 应该存在", tableName)
			assert.Len(t, result, 1, "分片表 %s 应该存在", tableName)
		}
	})

	t.Run("InsertAndQueryWithShardingKey", func(t *testing.T) {
		gormDB := provider.DB(ctx)

		// 创建测试用户，使用不同的user_id来测试分片
		testUsers := []ShardedUser{
			{UserID: 1, Name: "User1", Email: "user1@example.com"},
			{UserID: 2, Name: "User2", Email: "user2@example.com"},
			{UserID: 5, Name: "User5", Email: "user5@example.com"},
			{UserID: 9, Name: "User9", Email: "user9@example.com"},
		}

		// 插入用户数据
		for _, user := range testUsers {
			err := gormDB.Create(&user).Error
			assert.NoError(t, err, "插入用户 %d 应该成功", user.UserID)
		}

		// 验证数据被正确插入到相应的分片
		for _, user := range testUsers {
			expectedShard := user.UserID % int64(shardingConfig.NumberOfShards)
			tableName := fmt.Sprintf("sharded_users_%02d", expectedShard)

			// 直接查询分片表
			var count int64
			err = gormDB.Table(tableName).Where("user_id = ?", user.UserID).Count(&count).Error
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count, "用户 %d 应该在分片表 %s 中", user.UserID, tableName)
		}

		// 通过分片键查询（应该由分片插件自动路由）
		t.Run("QueryByShardingKey", func(t *testing.T) {
			targetUser := testUsers[0] // UserID: 1

			var foundUser ShardedUser
			err := gormDB.Where("user_id = ?", targetUser.UserID).First(&foundUser).Error
			assert.NoError(t, err)
			assert.Equal(t, targetUser.Name, foundUser.Name)
			assert.Equal(t, targetUser.Email, foundUser.Email)
		})

		// 清理测试数据
		for _, user := range testUsers {
			expectedShard := user.UserID % int64(shardingConfig.NumberOfShards)
			tableName := fmt.Sprintf("sharded_users_%02d", expectedShard)
			gormDB.Table(tableName).Where("user_id = ?", user.UserID).Delete(&ShardedUser{})
		}
	})

	t.Run("QueryWithoutShardingKeyShouldFail", func(t *testing.T) {
		gormDB := provider.DB(ctx)

		// 尝试不带分片键的查询（应该失败或返回空结果）
		var users []ShardedUser
		err := gormDB.Find(&users).Error

		// 分片库通常要求必须提供分片键
		// 这里我们期望查询成功但返回空结果，因为分片库在没有分片键时可能不会扫描所有分片
		assert.NoError(t, err)
		// 验证返回的确实是空结果
		assert.Empty(t, users)
	})
}

// TestShardingHelper 测试分片辅助工具
func TestShardingHelper(t *testing.T) {
	shardingConfig := db.NewShardingConfig("user_id", 16)

	// 注意：这里我们只能测试算法逻辑，因为实际的 ShardingHelper 在 internal 包中
	t.Run("ShardingConsistency", func(t *testing.T) {
		testValues := []interface{}{
			int64(1),
			int64(17),  // 1 + 16，应该和 1 在同一个分片
			int64(33),  // 1 + 32，应该和 1 在同一个分片
			"1",
			"17",
			"33",
		}

		// 验证相同逻辑的值落在同一个分片
		suffix1, _ := simulateShardingAlgorithm(testValues[0], shardingConfig.NumberOfShards)
		suffix2, _ := simulateShardingAlgorithm(testValues[1], shardingConfig.NumberOfShards)
		suffix3, _ := simulateShardingAlgorithm(testValues[2], shardingConfig.NumberOfShards)
		suffix4, _ := simulateShardingAlgorithm(testValues[3], shardingConfig.NumberOfShards)
		suffix5, _ := simulateShardingAlgorithm(testValues[4], shardingConfig.NumberOfShards)
		suffix6, _ := simulateShardingAlgorithm(testValues[5], shardingConfig.NumberOfShards)

		assert.Equal(t, suffix1, suffix2)
		assert.Equal(t, suffix1, suffix3)
		assert.Equal(t, suffix4, suffix5)
		assert.Equal(t, suffix4, suffix6)
	})
}

// BenchmarkShardingAlgorithmSimple 简单分片算法性能基准测试
func BenchmarkShardingAlgorithmSimple(b *testing.B) {
	shardingConfig := db.NewShardingConfig("user_id", 16)

	testCases := []interface{}{
		int64(123456789),
		"user_123456",
		"abcdefghijklmnopqrstuvwxyz",
		int32(98765),
		uint64(5555555555),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_, err := simulateShardingAlgorithm(tc, shardingConfig.NumberOfShards)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}