package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
	"github.com/spaolacci/murmur3"
)

// bloomFilterOperations 实现了 BloomFilterOperations 接口
// 它使用 Redis 的位图（Bitmaps）和哈希（Hashes）来自行实现布隆过滤器，不依赖 RedisBloom 模块。
type bloomFilterOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// bloomFilterMetadata 存储布隆过滤器的元数据
type bloomFilterMetadata struct {
	M uint   `json:"m"` // 位图大小 (number of bits)
	K uint   `json:"k"` // 哈希函数数量 (number of hash functions)
	N uint64 `json:"n"` // 预期容量 (capacity)
}

// newBloomFilterOperations 创建一个新的 bloomFilterOperations 实例
func newBloomFilterOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *bloomFilterOperations {
	return &bloomFilterOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatBitmapKey 格式化位图的键名
func (b *bloomFilterOperations) formatBitmapKey(key string) string {
	if b.keyPrefix == "" {
		return key + ":bf_bitmap"
	}
	return b.keyPrefix + ":" + key + ":bf_bitmap"
}

// formatMetaKey 格式化元数据的键名
func (b *bloomFilterOperations) formatMetaKey(key string) string {
	if b.keyPrefix == "" {
		return key + ":bf_meta"
	}
	return b.keyPrefix + ":" + key + ":bf_meta"
}

// BFInit 初始化一个布隆过滤器
func (b *bloomFilterOperations) BFInit(ctx context.Context, key string, errorRate float64, capacity int64) error {
	metaKey := b.formatMetaKey(key)

	// 检查元数据是否已存在，如果存在则不执行任何操作
	exists, err := b.client.Exists(ctx, metaKey).Result()
	if err != nil {
		b.logger.Error("检查布隆过滤器元数据失败", clog.String("key", metaKey), clog.Err(err))
		return fmt.Errorf("failed to check bloom filter metadata: %w", err)
	}
	if exists > 0 {
		b.logger.Info("布隆过滤器已存在，跳过初始化", clog.String("key", key))
		return nil
	}

	// 计算最优的 m 和 k
	m := optimalM(uint64(capacity), errorRate)
	k := optimalK(m, uint64(capacity))

	meta := bloomFilterMetadata{
		M: m,
		K: k,
		N: uint64(capacity),
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		b.logger.Error("序列化布隆过滤器元数据失败", clog.Err(err))
		return fmt.Errorf("failed to marshal bloom filter metadata: %w", err)
	}

	// 将元数据存储在哈希中
	err = b.client.Set(ctx, metaKey, metaJSON, 0).Err()
	if err != nil {
		b.logger.Error("存储布隆过滤器元数据失败", clog.String("key", metaKey), clog.Err(err))
		return fmt.Errorf("failed to store bloom filter metadata: %w", err)
	}

	b.logger.Info("成功初始化布隆过滤器",
		clog.String("key", key),
		clog.Uint64("capacity", uint64(capacity)),
		clog.Float64("errorRate", errorRate),
		clog.Uint("m_bits", m),
		clog.Uint("k_hashes", k),
	)

	return nil
}

// BFAdd 向布隆过滤器中添加一个元素
func (b *bloomFilterOperations) BFAdd(ctx context.Context, key string, item string) error {
	meta, err := b.getMetadata(ctx, key)
	if err != nil {
		return err
	}

	bitmapKey := b.formatBitmapKey(key)
	locations := b.getLocations([]byte(item), meta)

	pipe := b.client.Pipeline()
	for _, loc := range locations {
		pipe.Do(ctx, "SETBIT", bitmapKey, loc, 1)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		b.logger.Error("添加到布隆过滤器失败", clog.String("key", key), clog.String("item", item), clog.Err(err))
		return fmt.Errorf("failed to add to bloom filter: %w", err)
	}
	return nil
}

// BFExists 检查一个元素是否存在于布隆过滤器中
func (b *bloomFilterOperations) BFExists(ctx context.Context, key string, item string) (bool, error) {
	meta, err := b.getMetadata(ctx, key)
	if err != nil {
		// 如果元数据不存在，说明过滤器未初始化，元素肯定不存在
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	bitmapKey := b.formatBitmapKey(key)
	locations := b.getLocations([]byte(item), meta)

	pipe := b.client.Pipeline()
	results := make([]*redis.Cmd, meta.K)
	for i, loc := range locations {
		results[i] = pipe.Do(ctx, "GETBIT", bitmapKey, loc)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		b.logger.Error("检查布隆过滤器失败", clog.String("key", key), clog.String("item", item), clog.Err(err))
		return false, fmt.Errorf("failed to check bloom filter: %w", err)
	}

	for _, res := range results {
		bit, err := res.Int()
		if err != nil {
			return false, fmt.Errorf("failed to get bit result: %w", err)
		}
		if bit == 0 {
			return false, nil // 只要有一个位是 0，就说明元素肯定不存在
		}
	}

	return true, nil
}

// getMetadata 获取布隆过滤器的元数据
func (b *bloomFilterOperations) getMetadata(ctx context.Context, key string) (*bloomFilterMetadata, error) {
	metaKey := b.formatMetaKey(key)
	metaJSON, err := b.client.Get(ctx, metaKey).Result()
	if err != nil {
		if err == redis.Nil {
			b.logger.Error("布隆过滤器未初始化", clog.String("key", key))
			return nil, fmt.Errorf("bloom filter not initialized for key: %s", key)
		}
		b.logger.Error("获取布隆过滤器元数据失败", clog.String("key", metaKey), clog.Err(err))
		return nil, fmt.Errorf("failed to get bloom filter metadata: %w", err)
	}

	var meta bloomFilterMetadata
	err = json.Unmarshal([]byte(metaJSON), &meta)
	if err != nil {
		b.logger.Error("反序列化布隆过滤器元数据失败", clog.String("key", metaKey), clog.Err(err))
		return nil, fmt.Errorf("failed to unmarshal bloom filter metadata: %w", err)
	}
	return &meta, nil
}

// getLocations 计算一个元素在位图中的所有位置
func (b *bloomFilterOperations) getLocations(data []byte, meta *bloomFilterMetadata) []uint {
	locations := make([]uint, meta.K)
	// 使用两个哈希函数模拟 k 个哈希函数
	h1 := murmur3.New64WithSeed(0)
	h1.Write(data)
	hash1 := h1.Sum64()

	h2 := fnv.New64a()
	h2.Write(data)
	hash2 := h2.Sum64()

	for i := uint(0); i < meta.K; i++ {
		// Double-hashing: h(i) = (h1 + i * h2) mod m
		loc := (uint(hash1) + i*uint(hash2)) % uint(meta.M)
		locations[i] = loc
	}
	return locations
}

// optimalM 计算最优的位图大小 (m)
// m = - (n * ln(p)) / (ln(2)^2)
func optimalM(n uint64, p float64) uint {
	return uint(math.Ceil(-1 * float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)))
}

// optimalK 计算最优的哈希函数数量 (k)
// k = (m / n) * ln(2)
func optimalK(m uint, n uint64) uint {
	return uint(math.Ceil((float64(m) / float64(n)) * math.Ln2))
}
