package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/cfg"
	"ntun/internal/log"
	"ntun/internal/ntun/fabric"
	"ntun/internal/ntun/node"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

func main() {
	debugLogs := flag.Bool("v", false, "show debug logs")

	configPath := flag.String("c", "", "config file path")

	flag.Parse()

	app.InitEnv()

	if debugLogs != nil &&
		*debugLogs {
		os.Setenv("LOG_LEVEL", "debug")
	}

	log.Init()
	app.PrintLogo()
	app.PrintHeader()

	cfg := parseConfig(*configPath)

	node := node.NewNode(cfg)
	slog.Info(fmt.Sprintf("Node: %s", node.String()))

	input, err := fabric.CreateInput(node)
	if err != nil {
		panic(err)
	}

	output, err := fabric.CreateOutput(node)
	if err != nil {
		panic(err)
	}

	if input == nil && output == nil ||
		input != nil && output != nil {
		panic(errors.New("can't have none or both input and output"))
	}

	transporter, err := fabric.CreateTransporter(node)
	if err != nil {
		panic(err)
	}

	if transporter == nil {
		panic(errors.New("can't have not transport"))
	}

	node.AssignComponents(input, output, transporter)

	if input != nil {
		if err := input.Listen(); err != nil {
			panic(err)
		}
	}

	if output != nil {
		if err := output.Listen(); err != nil {
			panic(err)
		}
	}

	node.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func parseConfig(configPath string) *cfg.Config {
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

	var cfg cfg.Config
	err = cfg.Parse([]byte(strings.ReplaceAll(string(configBuf), "\t", "  ")))
	if err != nil {
		panic(fmt.Errorf("config file %s parse error %v", configPath, err))
	}

	return &cfg
}
