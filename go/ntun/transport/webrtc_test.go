package transport

import (
	"encoding/json"
	"ntun/internal/app"
	"os"
	"testing"
)

func TestWebRTCTransport(t *testing.T) {
	app.InitEnv()

	var turnServers struct {
		URLs     []string `json:"urls"`
		Username string   `json:"username"`
		Password string   `json:"credential"`
	}

	json.Unmarshal([]byte(os.Getenv("DEVELOP_WEB_RTC_SERVERS")), &turnServers)

	peer1 := NewWebRTCTransport(&TurnServerInfo{
		URL:      turnServers.URLs[0],
		Username: turnServers.Username,
		Password: turnServers.Password,
	})

	peer1.CreateOffer()
}
