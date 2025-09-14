package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 日志轮转演示 ===")

	// 清理之前的日志文件
	logDir := "logs"
	os.RemoveAll(logDir)

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	// 配置: JSON 文件输出 + 轮转
	config := &clog.Config{
		Level:     "info",
		Format:    "json",
		Output:    filepath.Join(logDir, "app.log"),
		AddSource: true,
		RootPath:  filepath.Join(os.Getenv("PWD"), "im-infra", "clog"),
		Rotation: &clog.RotationConfig{
			MaxSize:    1,    // 1MB (小值以便演示, 实际用 100+)
			MaxBackups: 3,    // 保留 3 个备份
			MaxAge:     7,    // 7 天
			Compress:   true, // 压缩旧文件
		},
	}

	// 初始化全局 logger
	if err := clog.Init(context.Background(), config); err != nil {
		fmt.Printf("初始化 logger 失败: %v\n", err)
		return
	}

	ctx := clog.WithTraceID(context.Background(), "rotation-trace-001")

	fmt.Printf("日志输出到: %s (轮转配置: MaxSize=1MB, MaxBackups=3, MaxAge=7d, Compress=true)\n", config.Output)
	fmt.Println("生成大量日志以模拟轮转 (实际轮转需达到大小/时间阈值)...")

	// 生成日志以模拟文件增长和轮转
	for i := 1; i <= 50; i++ { // 足够日志来潜在触发小 MaxSize
		logger := clog.WithContext(ctx).Namespace(fmt.Sprintf("batch.%d", i))

		logger.Info("批量日志消息",
			clog.Int("batch_id", i),
			clog.String("content", fmt.Sprintf("这是第 %d 条日志消息，包含一些填充内容来增加文件大小。", i)),
			clog.Duration("timestamp_offset", time.Duration(i)*time.Millisecond),
			clog.Float64("simulated_size", float64(i*10)), // 模拟数据
		)

		if i%10 == 0 {
			logger.Warn("批次警告", clog.Int("batch_id", i))
		}

		time.Sleep(50 * time.Millisecond) // 轻微延迟模拟真实场景
	}

	// 检查轮转文件
	fmt.Println("\n检查轮转结果:")
	files, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	logFiles := 0
	rotatedFiles := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if name == "app.log" {
			logFiles++
			fmt.Printf("- 当前日志: %s (大小: %d bytes)\n", name, getFileSize(filepath.Join(logDir, name)))
		} else if name != ".DS_Store" && (name[:len(name)-4] == "app.log." || name[:len(name)-7] == "app.log.gz") {
			rotatedFiles++
			fmt.Printf("- 轮转文件: %s (大小: %d bytes)\n", name, getFileSize(filepath.Join(logDir, name)))
		}
	}

	fmt.Printf("当前日志文件数: %d\n", logFiles)
	fmt.Printf("轮转备份文件数: %d (预期最多 %d)\n", rotatedFiles, config.Rotation.MaxBackups)

	if rotatedFiles > 0 {
		fmt.Println("✅ 轮转演示成功: 检测到备份文件 (注: 实际触发取决于日志量/时间)")
	} else {
		fmt.Println("ℹ️ 未触发轮转: 日志量不足或 MaxSize 未达 (运行更长时间或增加日志)")
	}

	fmt.Println("=== 轮转演示完成 ===")
}

// getFileSize 获取文件大小 (bytes)
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
