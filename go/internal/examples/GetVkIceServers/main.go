package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/log"
	"ntun/internal/ntun/transport/vk"
	"os"
)

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()

	iceServer, err := vk.GetIceServerFromJoinIdOrLink(os.Getenv("DEVELOP_VK_JOIN_ID_OR_LINK"))
	if err != nil {
		panic(fmt.Errorf("get iceServer from vk server error %v", err))
	}

	jsonBuf, _ := json.Marshal(iceServer)

	slog.Info(fmt.Sprintf("iceServer %s", jsonBuf))
}
