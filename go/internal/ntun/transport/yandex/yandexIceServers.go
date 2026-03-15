package yandex

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

func getJoinId(joinIdOrLink string) (string, error) {
	if joinIdOrLink == "" {
		return "", fmt.Errorf("join id or link is required")
	}

	check := func(joinIdOrLink string) bool {
		matched, _ := regexp.MatchString(`^\d{14}$`, joinIdOrLink)

		return matched
	}

	if check(joinIdOrLink) {
		return joinIdOrLink, nil
	}

	joinIdOrLink = strings.TrimPrefix(joinIdOrLink, "https://telemost.yandex.ru/j/")
	if check(joinIdOrLink) {
		return joinIdOrLink, nil
	}

	return "", fmt.Errorf("bad join id or link")
}

type YandexTelemostWebSocketSignalServerInfo struct {
	ParticipantId       uuid.UUID `json:"peer_id"`
	RoomId              uuid.UUID `json:"room_id"`
	Credentials         string    `json:"credentials"`
	ClientConfiguration struct {
		MediaServerUrl string `json:"media_server_url"`
	} `json:"client_configuration"`
}

func getYandexTelemostWebSocketSignalServerInfoByJoinId(joinId string) (*YandexTelemostWebSocketSignalServerInfo, error) {
	clientInstanceId := uuid.New().String()
	username := "anon_" + clientInstanceId[0:4]

	url := "https://cloud-api.yandex.ru/telemost_front/v2/telemost/conferences/https%3A%2F%2Ftelemost.yandex.ru%2Fj%2F" + url.PathEscape(joinId) + "/connection"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("next_gen_media_platform_allowed", "true")
	q.Add("display_name", username)
	q.Add("waiting_room_supported", "true")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("client-instance-id", clientInstanceId)
	req.Header.Set("Referer", "https://telemost.yandex.ru/")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response status %d", res.StatusCode)
	}

	var info YandexTelemostWebSocketSignalServerInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}

	if info.ClientConfiguration.MediaServerUrl == "" {
		return nil, errors.New("bad response")
	}

	return &info, nil
}

const helloMessageFormatString = `{
	"uid": "%s",
    "hello": {
        "participantMeta": {
            "name": "super",
            "role": "SPEAKER",
            "description": "",
            "sendAudio": false,
            "sendVideo": false
        },
        "participantAttributes": {
            "name": "super",
            "role": "SPEAKER",
            "description": ""
        },
        "sendAudio": false,
        "sendVideo": false,
        "sendSharing": false,
        "participantId": "%s",
        "roomId": "%s",
        "serviceName": "telemost",
        "credentials": "%s",
        "capabilitiesOffer": {
            "offerAnswerMode": ["SEPARATE"],
            "initialSubscriberOffer": ["ON_HELLO"],
            "slotsMode": ["FROM_CONTROLLER"],
            "simulcastMode": ["DISABLED", "STATIC"],
            "selfVadStatus": ["FROM_SERVER", "FROM_CLIENT"],
            "dataChannelSharing": ["TO_RTP"],
            "videoEncoderConfig": ["NO_CONFIG", "ONLY_INIT_CONFIG", "RUNTIME_CONFIG"],
            "dataChannelVideoCodec": ["VP8", "UNIQUE_CODEC_FROM_TRACK_DESCRIPTION"],
            "bandwidthLimitationReason": ["BANDWIDTH_REASON_DISABLED", "BANDWIDTH_REASON_ENABLED"],
            "sdkDefaultDeviceManagement": ["SDK_DEFAULT_DEVICE_MANAGEMENT_DISABLED", "SDK_DEFAULT_DEVICE_MANAGEMENT_ENABLED"],
            "joinOrderLayout": ["JOIN_ORDER_LAYOUT_DISABLED", "JOIN_ORDER_LAYOUT_ENABLED"],
            "pinLayout": ["PIN_LAYOUT_DISABLED"],
            "sendSelfViewVideoSlot": ["SEND_SELF_VIEW_VIDEO_SLOT_DISABLED", "SEND_SELF_VIEW_VIDEO_SLOT_ENABLED"],
            "serverLayoutTransition": ["SERVER_LAYOUT_TRANSITION_DISABLED"],
            "sdkPublisherOptimizeBitrate": ["SDK_PUBLISHER_OPTIMIZE_BITRATE_DISABLED", "SDK_PUBLISHER_OPTIMIZE_BITRATE_FULL", "SDK_PUBLISHER_OPTIMIZE_BITRATE_ONLY_SELF"],
            "sdkNetworkLostDetection": ["SDK_NETWORK_LOST_DETECTION_DISABLED"],
            "sdkNetworkPathMonitor": ["SDK_NETWORK_PATH_MONITOR_DISABLED"],
            "publisherVp9": ["PUBLISH_VP9_DISABLED", "PUBLISH_VP9_ENABLED"],
            "svcMode": ["SVC_MODE_DISABLED", "SVC_MODE_L3T3", "SVC_MODE_L3T3_KEY"],
            "subscriberOfferAsyncAck": ["SUBSCRIBER_OFFER_ASYNC_ACK_DISABLED", "SUBSCRIBER_OFFER_ASYNC_ACK_ENABLED"],
            "svcModes": ["FALSE"],
            "reportTelemetryModes": ["TRUE"],
            "keepDefaultDevicesModes": ["TRUE"]
        },
        "sdkInfo": {
            "implementation": "browser",
            "version": "5.15.0",
            "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
            "hwConcurrency": 8
        },
        "sdkInitializationId": "%s",
        "disablePublisher": false,
        "disableSubscriber": false,
        "disableSubscriberAudio": false
    }
}`

func getIceServerFromYandexTelemostWebSocketSignalServer(info *YandexTelemostWebSocketSignalServerInfo) (*webrtc.ICEServer, error) {
	webSocketConn, _, err := websocket.DefaultDialer.Dial(info.ClientConfiguration.MediaServerUrl, nil)
	if err != nil {
		return nil, err
	}
	defer webSocketConn.Close()

	webSocketConn.WriteMessage(websocket.TextMessage, []byte(
		fmt.Sprintf(helloMessageFormatString,
			uuid.New().String(),
			info.ParticipantId,
			info.RoomId,
			info.Credentials,
			uuid.New().String(),
		),
	))

	waitMessagesAmount := 5
	for {
		if waitMessagesAmount == 0 {
			return nil, errors.New("Bad response")
		}

		err := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			return nil, err
		}

		_, buf, err := webSocketConn.ReadMessage()
		if err != nil {
			return nil, err
		}

		var msg struct {
			ServerHello struct {
				RtcConfiguration struct {
					IceServers []*webrtc.ICEServer `json:"iceServers"`
				} `json:"rtcConfiguration"`
			} `json:"serverHello"`
		}

		err = json.Unmarshal(buf, &msg)
		if err == nil {
			for _, iceServer := range msg.ServerHello.RtcConfiguration.IceServers {
				for _, url := range iceServer.URLs {
					if strings.HasPrefix(url, "turn:") &&
						strings.Contains(url, "transport=tcp") {

						return iceServer, nil
					}
				}
			}
		}

		waitMessagesAmount--
	}
}

func GetIceServerFromJoinIdOrLink(joinIdOrLink string) (*webrtc.ICEServer, error) {
	joinId, err := getJoinId(joinIdOrLink)
	if err != nil {
		return nil, err
	}

	info, err := getYandexTelemostWebSocketSignalServerInfoByJoinId(joinId)
	if err != nil {
		return nil, err
	}

	iceServer, err := getIceServerFromYandexTelemostWebSocketSignalServer(info)
	if err != nil {
		return nil, err
	}

	return iceServer, nil
}
