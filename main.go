package main

import (
	"context"
	"flag"
	"fmt"
	"gochat/api"
	"gochat/clog"
	"gochat/connect"
	"gochat/logic"
	"gochat/site"
	"gochat/task"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ModuleRunner 定义模块运行接口
type ModuleRunner interface {
	Run() error
	Shutdown(ctx context.Context) error
}

func main() {
	// 设置日志级别为 Debug
	clog.SetLogLevel(clog.LevelDebug)

	// 解析命令行参数
	var module string
	var logDir string
	flag.StringVar(&module, "module", "", "assign run module (logic, task, api, connect, site)")
	flag.StringVar(&logDir, "logdir", "", "log output directory")
	flag.Parse()

	// 设置日志输出路径，如果指定了logDir
	if logDir != "" {
		if err := clog.SetLogPath(logDir); err != nil {
			clog.Error("Failed to set log directory: %v", err)
			os.Exit(1)
		}
		clog.Info("Log files will be saved to: %s", logDir)
	}

	if module == "" {
		clog.Error("Error: Module must be specified, use -module=<module_name>")
		flag.Usage()
		os.Exit(1)
	}

	clog.Info("Starting %s module", module)

	// 获取模块实例
	runner, err := getModuleRunner(module)
	if err != nil {
		clog.Error("Module initialization failed: %v", err)
		os.Exit(1)
	}

	// 启动模块
	if err := runner.Run(); err != nil {
		clog.Error("%s module failed to start: %v", module, err)
		os.Exit(1)
	}
	clog.Info("%s module started successfully!", module)

	// 优雅关闭处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quit

	clog.Warning("Received signal %v, graceful shutdown initiated...", sig)

	// 创建关闭上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭模块
	if err := runner.Shutdown(ctx); err != nil {
		clog.Error("Error during shutdown: %v", err)
		os.Exit(1)
	}

	clog.Info("Service has been safely terminated")

	// 关闭日志系统
	if err := clog.Close(); err != nil {
		fmt.Printf("Error closing log system: %v\n", err)
	}
}

// getModuleRunner 根据模块名返回对应的运行器
func getModuleRunner(module string) (ModuleRunner, error) {
	clog.Debug("Creating runner for module: %s", module)

	switch module {
	case "logic":
		return logic.New(), nil
	case "task":
		return task.New(), nil
	case "api":
		return api.New(), nil
	case "connect":
		return connect.New(), nil
	case "site":
		return site.New(), nil
	default:
		return nil, fmt.Errorf("unknown module: %s", module)
	}
}
