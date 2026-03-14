package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/conf"
	"ntun/internal/log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	app.Init()
	log.Init()

	slog.Info(fmt.Sprintf("%s v%s (%s)", app.Name, app.Version, runtime.Version()))

	var configPath string
	flag.StringVar(&configPath, "c", "", "config file path (short)")
	flag.StringVar(&configPath, "config", "", "config file path (long)")

	flag.Parse()

	cfg := parseConfig(configPath)
	slog.Debug(fmt.Sprintf("%+v", cfg))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func parseConfig(configPath string) *conf.Config {
	if configPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		configPath = path.Join(cwd, "config.yaml")
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Errorf("config file %s does not exist", configPath))
		}
		if os.IsPermission(err) {
			panic(fmt.Errorf("config file %s permission denied", configPath))
		}
		panic(fmt.Errorf("config file %s error", configPath))
	}

	configFile, err := os.OpenFile(configPath, os.O_RDONLY, 0)
	if err != nil {
		panic(fmt.Errorf("config file %s error %v", configPath, err))
	}
	defer configFile.Close()

	configBuf, err := io.ReadAll(configFile)
	if err != nil {
		panic(fmt.Errorf("config file %s error %v", configPath, err))
	}

	var cfg conf.Config
	err = cfg.Parse([]byte(strings.ReplaceAll(string(configBuf), "\t", "  ")))
	if err != nil {
		panic(fmt.Errorf("config file %s parse error %v", configPath, err))
	}

	return &cfg
}
