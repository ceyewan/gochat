package main

import (
	"flag"
	"fmt"
	"gochat/api"
	"gochat/logic"
	"gochat/task"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var module string
	flag.StringVar(&module, "module", "", "assign run module")
	flag.Parse()
	fmt.Printf("start run %s module\n", module)
	switch module {
	case "logic":
		logic.New().Run()
	case "task":
		task.New().Run()
	case "api":
		api.New().Run()
	default:
		fmt.Println("exiting,module param error!")
		return
	}
	fmt.Printf("run %s module done!\n", module)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	fmt.Println("Server exiting")
}
