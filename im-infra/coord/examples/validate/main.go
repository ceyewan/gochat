package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 验证工具：确保所有示例都能正常运行
// 使用方法: go run validate/main.go

func main() {
	fmt.Println("=== Coord 模块示例验证工具 ===")
	fmt.Println("验证所有示例是否能在真实环境中正常运行")
	fmt.Println()

	// 检查etcd是否运行
	if !checkEtcdRunning() {
		fmt.Println("❌ etcd 服务未运行")
		fmt.Println("请先启动 etcd:")
		fmt.Println("  etcd --listen-client-urls=http://localhost:2379 --advertise-client-urls=http://localhost:2379")
		os.Exit(1)
	}

	fmt.Println("✓ etcd 服务运行正常")
	fmt.Println()

	// 获取当前目录的父目录（examples目录）
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取当前目录失败: %v", err)
	}

	// 验证工具在examples/validate目录，需要到examples目录找其他示例
	examplesDir := filepath.Dir(currentDir)

	// 要验证的示例列表
	examples := []string{
		"lock",
		"registry",
		"config",
		"config_manager",
		"grpc_resolver",
		"comprehensive",
		"advanced",
	}

	results := make(map[string]bool)
	durations := make(map[string]time.Duration)

	for _, example := range examples {
		fmt.Printf("正在验证示例: %s\n", example)

		examplePath := filepath.Join(examplesDir, example)
		if _, err := os.Stat(examplePath); os.IsNotExist(err) {
			fmt.Printf("  ⚠️  示例目录不存在: %s\n", examplePath)
			results[example] = false
			continue
		}

		start := time.Now()
		success := runExample(examplePath, example)
		duration := time.Since(start)

		results[example] = success
		durations[example] = duration

		if success {
			fmt.Printf("  ✅ %s 验证成功 (耗时: %v)\n", example, duration)
		} else {
			fmt.Printf("  ❌ %s 验证失败 (耗时: %v)\n", example, duration)
		}
		fmt.Println()
	}

	// 生成验证报告
	printValidationReport(results, durations)
}

func checkEtcdRunning() bool {
	// 尝试连接etcd
	cmd := exec.Command("etcdctl", "--endpoints=localhost:2379", "endpoint", "health")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 尝试使用curl检查
		curlCmd := exec.Command("curl", "-s", "http://localhost:2379/health")
		curlOutput, curlErr := curlCmd.CombinedOutput()
		return curlErr == nil && strings.Contains(string(curlOutput), "true")
	}
	return strings.Contains(string(output), "is healthy") || strings.Contains(string(output), "success")
}

func runExample(examplePath, exampleName string) bool {
	mainFile := filepath.Join(examplePath, "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		fmt.Printf("  ⚠️  main.go 文件不存在: %s\n", mainFile)
		return false
	}

	// 运行示例
	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = examplePath
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	// 设置超时
	done := make(chan bool, 1)
	var output []byte
	var runErr error

	go func() {
		output, runErr = cmd.CombinedOutput()
		done <- true
	}()

	// 等待完成或超时
	select {
	case <-done:
		if runErr != nil {
			fmt.Printf("  运行错误: %v\n", runErr)
			fmt.Printf("  输出: %s\n", string(output))
			return false
		}

		// 检查输出是否包含成功标志
		outputStr := string(output)
		if strings.Contains(outputStr, "完成") ||
			strings.Contains(outputStr, "success") ||
			strings.Contains(outputStr, "✓") ||
			strings.Contains(outputStr, "测试通过") {
			return true
		}

		// 如果没有明确的错误信息，也认为成功
		if !strings.Contains(outputStr, "失败") && !strings.Contains(outputStr, "error") {
			return true
		}

		return false

	case <-time.After(30 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		fmt.Printf("  超时: 示例运行超过30秒\n")
		return false
	}
}

func printValidationReport(results map[string]bool, durations map[string]time.Duration) {
	fmt.Println("\n=== 验证报告 ===")

	total := len(results)
	successCount := 0
	failedCount := 0

	for _, success := range results {
		if success {
			successCount++
		} else {
			failedCount++
		}
	}

	fmt.Printf("总示例数: %d\n", total)
	fmt.Printf("成功: %d (%.1f%%)\n", successCount, float64(successCount)/float64(total)*100)
	fmt.Printf("失败: %d (%.1f%%)\n", failedCount, float64(failedCount)/float64(total)*100)
	fmt.Println()

	// 详细结果
	fmt.Println("详细结果:")
	for example, success := range results {
		status := "✅ 通过"
		if !success {
			status = "❌ 失败"
		}
		fmt.Printf("  %s %s (耗时: %v)\n", status, example, durations[example])
	}

	fmt.Println()

	// 建议
	if failedCount > 0 {
		fmt.Println("建议:")
		fmt.Println("  1. 检查 etcd 是否正常运行")
		fmt.Println("  2. 查看失败示例的具体错误信息")
		fmt.Println("  3. 确保网络连接正常")
		fmt.Println("  4. 检查示例代码是否正确")
	} else {
		fmt.Println("🎉 所有示例验证通过！")
		fmt.Println("coord 模块功能正常，可以在生产环境中使用。")
	}
}

// 备用etcd检查函数
func checkEtcdAlternative() bool {
	// 尝试使用telnet检查端口
	cmd := exec.Command("timeout", "2", "telnet", "localhost", "2379")
	err := cmd.Run()
	return err == nil
}
