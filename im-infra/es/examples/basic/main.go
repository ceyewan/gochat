package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/es"
)

// MyMessage 是一个示例结构体，实现了 es.Indexable 接口
type MyMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetID 返回消息的唯一标识符
func (m MyMessage) GetID() string {
	return m.ID
}

func main() {
	// 1. 初始化日志记录器
	logger := clog.Namespace("es-example")
	ctx := context.Background()

	// 2. 获取默认配置并创建新的 es provider
	cfg := es.GetDefaultConfig("development")
	// 使用真实的 ES 地址
	cfg.Addresses = []string{"http://localhost:9200"}
	cfg.BulkIndexer.FlushInterval = 2 * time.Second // 加快刷新速度用于示例

	log.Println("正在连接到 Elasticsearch...")
	esProvider, err := es.New[MyMessage](ctx, cfg, es.WithLogger(logger))
	if err != nil {
		log.Fatalf("创建 es provider 失败: %v", err)
	}
	defer esProvider.Close()
	log.Println("成功连接到 Elasticsearch")

	// 3. 创建一些示例消息
	messages := make([]MyMessage, 0, 10)
	for i := 0; i < 10; i++ {
		messages = append(messages, MyMessage{
			ID:        fmt.Sprintf("msg-%d", i+1),
			SessionID: "session-123",
			Content:   fmt.Sprintf("Hello, Elasticsearch! This is message %d with some test content", i+1),
			Timestamp: time.Now(),
		})
	}

	// 添加不同会话的消息
	messages = append(messages, MyMessage{
		ID:        "msg-other-1",
		SessionID: "session-456",
		Content:   "This is a message from different session and should not be found in main session search",
		Timestamp: time.Now(),
	})

	// 4. 批量索引这些消息
	indexName := "example-messages" + fmt.Sprintf("-%d", time.Now().Unix())
	logger.Info("正在索引文档...", clog.Int("count", len(messages)), clog.String("index", indexName))
	if err := esProvider.BulkIndex(ctx, indexName, messages); err != nil {
		log.Fatalf("批量索引消息失败: %v", err)
	}

	logger.Info("成功索引文档。等待批量索引器刷新...")

	// 等待批量索引器完成 - 增加等待时间
	log.Println("等待批量索引器刷新...")
	time.Sleep(5 * time.Second)

	// 5. 搜索消息
	log.Println("\n=== 测试 1: 全局搜索包含 'message' 的消息 ===")
	searchResult, err := esProvider.SearchGlobal(ctx, indexName, "message", 1, 10)
	if err != nil {
		log.Fatalf("搜索消息失败: %v", err)
	}

	log.Printf("找到 %d 条消息:\n", searchResult.Total)
	for _, msg := range searchResult.Items {
		log.Printf("  - ID: %s, 会话: %s, 内容: %s\n", (*msg).GetID(), (*msg).SessionID, (*msg).Content)
	}

	// 6. 测试会话内搜索
	log.Println("\n=== 测试 2: 在会话 'session-123' 中搜索消息 ===")
	sessionResult, err := esProvider.SearchInSession(ctx, indexName, "session-123", "message", 1, 5)
	if err != nil {
		log.Fatalf("在会话中搜索消息失败: %v", err)
	}

	log.Printf("在会话 'session-123' 中找到 %d 条消息:\n", sessionResult.Total)
	for _, msg := range sessionResult.Items {
		log.Printf("  - ID: %s, 内容: %s\n", (*msg).GetID(), (*msg).Content)
	}

	// 7. 测试搜索不存在的关键词
	log.Println("\n=== 测试 3: 搜索不存在的关键词 ===")
	noResult, err := esProvider.SearchGlobal(ctx, indexName, "nonexistentkeyword", 1, 10)
	if err != nil {
		log.Fatalf("搜索失败: %v", err)
	}

	log.Printf("搜索不存在的关键词找到 %d 条消息\n", noResult.Total)

	// 8. 测试分页
	log.Println("\n=== 测试 4: 分页搜索 ===")
	for page := 1; page <= 3; page++ {
		pageResult, err := esProvider.SearchGlobal(ctx, indexName, "hello", page, 3)
		if err != nil {
			log.Fatalf("分页搜索失败: %v", err)
		}
		log.Printf("第 %d 页 (每页 3 条): 找到 %d 条记录\n", page, len(pageResult.Items))
		for _, msg := range pageResult.Items {
			log.Printf("  - ID: %s\n", (*msg).GetID())
		}
		if len(pageResult.Items) == 0 {
			break
		}
	}

	log.Println("\n=== 示例完成 ===")
	log.Printf("使用的索引: %s\n", indexName)
	log.Printf("总计索引了 %d 条消息\n", len(messages))
}
