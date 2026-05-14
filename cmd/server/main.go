package main

import (
	"context"
	stdlog "log"

	"llmtrace/internal/bootstrap"
)

// main 启动服务进程，并将启动失败直接输出到标准日志。
func main() {
	if err := bootstrap.Run(context.Background()); err != nil {
		stdlog.Fatal(err)
	}
}
