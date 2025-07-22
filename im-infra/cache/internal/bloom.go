package internal

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// BloomInit 初始化布隆过滤器
func (c *cache) BloomInit(ctx context.Context, key string, capacity uint64, errorRate float64) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if capacity == 0 {
		capacity = c.bloomConfig.DefaultCapacity
	}
	if errorRate <= 0 || errorRate >= 1 {
		errorRate = c.bloomConfig.DefaultErrorRate
	}

	// 计算布隆过滤器参数
	bitSize := calculateOptimalBitSize(capacity, errorRate)
	hashFunctions := calculateOptimalHashFunctions(bitSize, capacity)

	bloomLogger := clog.Module("cache.bloom")
	bloomLogger.Info("初始化布隆过滤器",
		clog.String("key", key),
		clog.Uint64("capacity", capacity),
		clog.Float64("errorRate", errorRate),
		clog.Uint64("bitSize", bitSize),
		clog.Int("hashFunctions", hashFunctions),
	)

	// 存储布隆过滤器的元数据
	metaKey := c.formatKey("bloom:meta:" + key)
	metadata := map[string]interface{}{
		"capacity":      capacity,
		"errorRate":     errorRate,
		"bitSize":       bitSize,
		"hashFunctions": hashFunctions,
		"initialized":   true,
	}

	return c.executeWithLogging(ctx, "BLOOM_INIT", key, func() error {
		err := c.client.HMSet(ctx, metaKey, metadata).Err()
		if err != nil {
			return c.handleRedisError("BLOOM_INIT", key, err)
		}

		bloomLogger.Info("布隆过滤器初始化成功",
			clog.String("key", key),
			clog.String("metaKey", metaKey),
		)

		return nil
	})
}

// BloomAdd 向布隆过滤器添加元素
func (c *cache) BloomAdd(ctx context.Context, key string, item string) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if item == "" {
		return fmt.Errorf("item cannot be empty")
	}

	// 获取布隆过滤器元数据
	metadata, err := c.getBloomMetadata(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get bloom filter metadata: %w", err)
	}

	// 计算哈希值并设置对应的位
	hashes := c.calculateHashes(item, metadata.HashFunctions)
	bloomKey := c.formatKey("bloom:bits:" + key)

	bloomLogger := clog.Module("cache.bloom")

	return c.executeWithLogging(ctx, "BLOOM_ADD", key, func() error {
		// 使用 pipeline 批量设置位
		pipe := c.client.Pipeline()
		for _, hash := range hashes {
			bitOffset := hash % metadata.BitSize
			pipe.SetBit(ctx, bloomKey, int64(bitOffset), 1)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return c.handleRedisError("BLOOM_ADD", key, err)
		}

		bloomLogger.Debug("添加元素到布隆过滤器",
			clog.String("key", key),
			clog.String("item", item),
			clog.Int("hashCount", len(hashes)),
		)

		return nil
	})
}

// BloomExists 检查元素是否可能存在于布隆过滤器中
func (c *cache) BloomExists(ctx context.Context, key string, item string) (bool, error) {
	if err := c.validateContext(ctx); err != nil {
		return false, err
	}
	if err := c.validateKey(key); err != nil {
		return false, err
	}
	if item == "" {
		return false, fmt.Errorf("item cannot be empty")
	}

	// 获取布隆过滤器元数据
	metadata, err := c.getBloomMetadata(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to get bloom filter metadata: %w", err)
	}

	// 计算哈希值并检查对应的位
	hashes := c.calculateHashes(item, metadata.HashFunctions)
	bloomKey := c.formatKey("bloom:bits:" + key)

	bloomLogger := clog.Module("cache.bloom")

	var exists bool
	err = c.executeWithLogging(ctx, "BLOOM_EXISTS", key, func() error {
		// 使用 pipeline 批量检查位
		pipe := c.client.Pipeline()
		cmds := make([]*redis.IntCmd, len(hashes))

		for i, hash := range hashes {
			bitOffset := hash % metadata.BitSize
			cmds[i] = pipe.GetBit(ctx, bloomKey, int64(bitOffset))
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return c.handleRedisError("BLOOM_EXISTS", key, err)
		}

		// 检查所有位是否都为 1
		for _, cmd := range cmds {
			if cmd.Val() == 0 {
				exists = false
				bloomLogger.Debug("元素不存在于布隆过滤器",
					clog.String("key", key),
					clog.String("item", item),
				)
				return nil
			}
		}

		exists = true
		bloomLogger.Debug("元素可能存在于布隆过滤器",
			clog.String("key", key),
			clog.String("item", item),
		)

		return nil
	})

	return exists, err
}

// bloomMetadata 布隆过滤器元数据结构
type bloomMetadata struct {
	Capacity      uint64
	ErrorRate     float64
	BitSize       uint64
	HashFunctions int
	Initialized   bool
}

// getBloomMetadata 获取布隆过滤器元数据
func (c *cache) getBloomMetadata(ctx context.Context, key string) (*bloomMetadata, error) {
	metaKey := c.formatKey("bloom:meta:" + key)

	result, err := c.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("bloom filter not initialized for key %s", key)
	}

	metadata := &bloomMetadata{}

	if val, ok := result["capacity"]; ok {
		if capacity, err := parseUint64(val); err == nil {
			metadata.Capacity = capacity
		}
	}

	if val, ok := result["errorRate"]; ok {
		if errorRate, err := parseFloat64(val); err == nil {
			metadata.ErrorRate = errorRate
		}
	}

	if val, ok := result["bitSize"]; ok {
		if bitSize, err := parseUint64(val); err == nil {
			metadata.BitSize = bitSize
		}
	}

	if val, ok := result["hashFunctions"]; ok {
		if hashFunctions, err := parseInt(val); err == nil {
			metadata.HashFunctions = hashFunctions
		}
	}

	if val, ok := result["initialized"]; ok {
		metadata.Initialized = val == "true"
	}

	if !metadata.Initialized {
		return nil, fmt.Errorf("bloom filter not properly initialized for key %s", key)
	}

	return metadata, nil
}

// calculateHashes 计算多个哈希值
func (c *cache) calculateHashes(item string, hashFunctions int) []uint64 {
	hashes := make([]uint64, hashFunctions)

	// 使用 SHA-256 作为基础哈希函数
	h := sha256.Sum256([]byte(item))

	// 使用双重哈希技术生成多个哈希值
	hash1 := binary.BigEndian.Uint64(h[:8])
	hash2 := binary.BigEndian.Uint64(h[8:16])

	for i := 0; i < hashFunctions; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}

	return hashes
}

// calculateOptimalBitSize 计算最优位数组大小
func calculateOptimalBitSize(capacity uint64, errorRate float64) uint64 {
	// m = -n * ln(p) / (ln(2)^2)
	// 其中 n 是预期元素数量，p 是错误率
	m := -float64(capacity) * math.Log(errorRate) / (math.Log(2) * math.Log(2))
	return uint64(math.Ceil(m))
}

// calculateOptimalHashFunctions 计算最优哈希函数数量
func calculateOptimalHashFunctions(bitSize, capacity uint64) int {
	// k = (m/n) * ln(2)
	// 其中 m 是位数组大小，n 是预期元素数量
	k := float64(bitSize) / float64(capacity) * math.Log(2)
	return int(math.Round(k))
}

// parseUint64 解析 uint64
func parseUint64(s string) (uint64, error) {
	var result uint64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseFloat64 解析 float64
func parseFloat64(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// parseInt 解析 int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// BloomClear 清空布隆过滤器
func (c *cache) BloomClear(ctx context.Context, key string) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}

	metaKey := c.formatKey("bloom:meta:" + key)
	bloomKey := c.formatKey("bloom:bits:" + key)

	bloomLogger := clog.Module("cache.bloom")

	return c.executeWithLogging(ctx, "BLOOM_CLEAR", key, func() error {
		// 删除元数据和位数组
		_, err := c.client.Del(ctx, metaKey, bloomKey).Result()
		if err != nil {
			return c.handleRedisError("BLOOM_CLEAR", key, err)
		}

		bloomLogger.Info("清空布隆过滤器",
			clog.String("key", key),
		)

		return nil
	})
}
