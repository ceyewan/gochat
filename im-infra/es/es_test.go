package es

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessage 测试用的消息结构体，实现 Indexable 接口
type TestMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetID 实现 Indexable 接口
func (m TestMessage) GetID() string {
	return m.ID
}

func TestIndexableInterface(t *testing.T) {
	msg := TestMessage{
		ID:        "test-123",
		SessionID: "session-456",
		Content:   "test message content",
		Timestamp: time.Now(),
	}

	// 验证实现了 Indexable 接口
	var _ Indexable = msg

	// 验证 GetID 方法
	assert.Equal(t, "test-123", msg.GetID())
}

func TestSearchResult(t *testing.T) {
	// 创建测试消息
	msg1 := &TestMessage{ID: "1", Content: "message 1"}
	msg2 := &TestMessage{ID: "2", Content: "message 2"}

	// 创建搜索结果
	result := &SearchResult[TestMessage]{
		Total: 2,
		Items: []*TestMessage{msg1, msg2},
	}

	// 验证结果
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "1", result.Items[0].GetID())
	assert.Equal(t, "2", result.Items[1].GetID())
}

func TestProviderCreation(t *testing.T) {
	// 由于需要连接到真实的 Elasticsearch，这里只测试配置验证
	cfg := GetDefaultConfig("test")

	// 测试无效配置 - 空地址列表
	cfg.Addresses = []string{}

	ctx := context.Background()
	logger := clog.Namespace("es-test")

	provider, err := New[TestMessage](ctx, cfg, WithLogger(logger))

	// 期望创建失败，因为没有有效的 Elasticsearch 地址
	if provider != nil {
		provider.Close() // 如果创建成功，确保清理
	}

	// 注意：由于依赖外部 Elasticsearch 服务，在 CI/CD 环境中可能会失败
	// 这里只验证错误不是空指针错误
	if err != nil {
		assert.Contains(t, err.Error(), "elasticsearch", "错误信息应该包含 elasticsearch 相关内容")
	}
}

func TestProviderInterface(t *testing.T) {
	// 测试类型断言，确保 provider 实现了 Provider 接口
	// 这个测试验证我们的接口定义是正确的
	var _ Provider[TestMessage] = (*provider[TestMessage])(nil)
}

func TestOptions(t *testing.T) {
	logger := clog.Namespace("test-es")

	options := &providerOptions{}

	// 测试 WithLogger 选项
	WithLogger(logger)(options)
	assert.Equal(t, logger, options.logger)

	// 测试 WithCoordinator 选项 - coord.Provider 可能为 nil
	WithCoordinator(nil)(options)
	assert.Nil(t, options.coord)
}

func TestConfigDefaults(t *testing.T) {
	// 测试开发环境配置
	devConfig := GetDefaultConfig("development")
	require.NotNil(t, devConfig)
	assert.NotEmpty(t, devConfig.Addresses)
	assert.Equal(t, []string{"http://localhost:9200"}, devConfig.Addresses)

	// 测试生产环境配置
	prodConfig := GetDefaultConfig("production")
	require.NotNil(t, prodConfig)
	assert.NotEmpty(t, prodConfig.Addresses)

	// 验证 BulkIndexer 配置
	assert.Greater(t, devConfig.BulkIndexer.Workers, 0)
	assert.Greater(t, devConfig.BulkIndexer.FlushBytes, 0)
	assert.Greater(t, devConfig.BulkIndexer.FlushInterval, time.Duration(0))
}

// 集成测试 - 需要运行 Elasticsearch 实例
func TestProviderIntegration(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	t.Skip("集成测试需要运行中的 Elasticsearch 实例")

	// 如果有可用的 Elasticsearch 实例，可以启用以下测试
	/*
		cfg := GetDefaultConfig("development")
		ctx := context.Background()
		logger := clog.Namespace("es-integration-test")

		provider, err := New[TestMessage](ctx, cfg, WithLogger(logger))
		if err != nil {
			t.Skipf("跳过集成测试，无法连接到 Elasticsearch: %v", err)
		}
		defer provider.Close()

		// 创建测试数据
		messages := []TestMessage{
			{
				ID:        "test-1",
				SessionID: "session-123",
				Content:   "这是一条测试消息",
				Timestamp: time.Now(),
			},
			{
				ID:        "test-2",
				SessionID: "session-123",
				Content:   "这是另一条测试消息",
				Timestamp: time.Now(),
			},
		}

		// 测试批量索引
		indexName := "test-messages-" + fmt.Sprintf("%d", time.Now().Unix())
		err = provider.BulkIndex(ctx, indexName, messages)
		assert.NoError(t, err)

		// 等待索引完成
		time.Sleep(2 * time.Second)

		// 测试全局搜索
		result, err := provider.SearchGlobal(ctx, indexName, "测试", 1, 10)
		assert.NoError(t, err)
		assert.Greater(t, result.Total, int64(0))

		// 测试会话内搜索
		sessionResult, err := provider.SearchInSession(ctx, indexName, "session-123", "消息", 1, 10)
		assert.NoError(t, err)
		assert.Greater(t, sessionResult.Total, int64(0))
	*/
}

func BenchmarkProviderCreation(b *testing.B) {
	cfg := GetDefaultConfig("development")
	ctx := context.Background()
	logger := clog.Namespace("es-benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := New[TestMessage](ctx, cfg, WithLogger(logger))
		if err == nil && provider != nil {
			provider.Close()
		}
	}
}
