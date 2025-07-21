package main

// import (
// 	"context"
// 	"flag"
// 	"fmt"
// 	"gochat/api"
// 	"gochat/clog"
// 	"gochat/connect"
// 	"gochat/logic"
// 	"gochat/site"
// 	"gochat/task"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"
// )

// // ModuleRunner 定义模块运行接口
// type ModuleRunner interface {
// 	Run() error
// 	Shutdown(ctx context.Context) error
// }

// func main() {
// 	// 使用默认配置初始化日志
// 	config := clog.DefaultConfig()
// 	config.ConsoleOutput = true // 同时输出到控制台

// 	err := clog.Init(config)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer clog.Sync() // 程序结束时确保日志刷新

// 	// 解析命令行参数
// 	var module string
// 	var logDir string
// 	flag.StringVar(&module, "module", "", "assign run module (logic, task, api, connect, site)")
// 	flag.StringVar(&logDir, "logdir", "", "log output directory")
// 	flag.Parse()

// 	if module == "" {
// 		clog.Error("Error: Module must be specified, use -module=<module_name>")
// 		flag.Usage()
// 		os.Exit(1)
// 	}

// 	clog.Infof("Starting %s module", module)

// 	// 获取模块实例
// 	runner, err := getModuleRunner(module)
// 	if err != nil {
// 		clog.Errorf("Module initialization failed: %v", err)
// 		os.Exit(1)
// 	}

// 	// 启动模块
// 	if err := runner.Run(); err != nil {
// 		clog.Errorf("%s module failed to start: %v", module, err)
// 		os.Exit(1)
// 	}
// 	clog.Infof("%s module started successfully!", module)

// 	// 优雅关闭处理
// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
// 	sig := <-quit

// 	clog.Warnf("Received signal %v, graceful shutdown initiated...", sig)

// 	// 创建关闭上下文，设置超时
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	// 关闭模块
// 	if err := runner.Shutdown(ctx); err != nil {
// 		clog.Errorf("Error during shutdown: %v", err)
// 		os.Exit(1)
// 	}

// 	clog.Infof("Service has been safely terminated")
// }

// // getModuleRunner 根据模块名返回对应的运行器
// func getModuleRunner(module string) (ModuleRunner, error) {
// 	clog.Debugf("Creating runner for module: %s", module)

// 	switch module {
// 	case "logic":
// 		return logic.New(), nil
// 	case "task":
// 		return task.New(), nil
// 	case "api":
// 		return api.New(), nil
// 	case "connect":
// 		return connect.New(), nil
// 	case "site":
// 		return site.New(), nil
// 	default:
// 		return nil, fmt.Errorf("unknown module: %s", module)
// 	}
// }
