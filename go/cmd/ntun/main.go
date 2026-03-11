package main

import (
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	app.InitEnv()
	log.Init()

	slog.Info(fmt.Sprintf("%s v%s (%s)", app.Name, app.Version, runtime.Version()))
	if app.IsDevelopment() {
		slog.Warn("Development mode")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
