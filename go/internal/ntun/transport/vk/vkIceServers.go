package vk

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

func getJoinId(joinIdOrLink string) (string, error) {
	if joinIdOrLink == "" {
		return "", fmt.Errorf("join id or link is required")
	}

	joinIdOrLink = strings.TrimPrefix(joinIdOrLink, "https://vk.com/call/join/")

	return joinIdOrLink, nil
}

// type YandexTelemostWebSocketSignalServerInfo struct {
// 	ParticipantId       uuid.UUID `json:"peer_id"`
// 	RoomId              uuid.UUID `json:"room_id"`
// 	Credentials         string    `json:"credentials"`
// 	ClientConfiguration struct {
// 		MediaServerUrl string `json:"media_server_url"`
// 	} `json:"client_configuration"`
// }

func getValueByPathInJsonData[T any](data map[string]any, path string) (res T) {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return
			}
			current = v[idx]
		default:
			return
		}
	}

	res = current.(T)

	return
}

func GetIceServerFromJoinIdOrLink(joinIdOrLink string) (*webrtc.ICEServer, error) {
	joinId, err := getJoinId(joinIdOrLink)
	if err != nil {
		return nil, err
	}

	const (
		v              = "5.231"
		applicationKey = "CGMMEJLGDIHBABABA"
		clientSecret   = "QbYic1K3lEV5kTGiqlq2"
		clientId       = "6287487"
		appId          = "6287487"
	)

	var (
		deviceId = uuid.New().String()
		username = "Anonym " + deviceId[0:4]
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	doPost := func(rUrl string, query map[string]string, bodyParams map[string]string) (map[string]any, error) {
		var data map[string]any

		bodyQuery := url.Values{}
		for k, v := range bodyParams {
			bodyQuery.Set(k, v)
		}

		req, err := http.NewRequest("POST", rUrl, strings.NewReader(bodyQuery.Encode()))
		if err != nil {
			return data, err
		}

		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Del("User-Agent")

		slog.Debug(fmt.Sprintf("%s %s", req.Method, req.URL.String()))

		res, err := client.Do(req)
		if err != nil {
			return data, err
		}

		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return data, err
		}

		if res.StatusCode != http.StatusOK {
			return data, fmt.Errorf("bad response status %d", res.StatusCode)
		}

		if err := json.Unmarshal(body, &data); err != nil {
			return data, err
		}

		return data, nil
	}

	getAnonymToken1Data, err := doPost(
		"https://login.vk.com/",
		map[string]string{
			"act": "get_anonym_token",
		},
		map[string]string{
			"client_secret":           clientSecret,
			"client_id":               clientId,
			"app_id":                  appId,
			"version":                 "1",
			"scopes":                  "audio_anonymous,video_anonymous,photos_anonymous,profile_anonymous",
			"isApiOauthAnonymEnabled": "false",
		},
	)

	if err != nil {
		return nil, fmt.Errorf("vk error %w", err)
	}

	accessToken1 := getValueByPathInJsonData[string](getAnonymToken1Data, "data.access_token")
	if accessToken1 == "" {
		return nil, fmt.Errorf("vk error %w", err)
	}

	// responses["https://api.vk.com/method/calls.getAnonymousAccessTokenPayload"] = await postJson("https://api.vk.com/method/calls.getAnonymousAccessTokenPayload?v=5.265&client_id=6287487", {
	// 	"access_token": responses["https://login.vk.com/?act=get_anonym_token__1"]["data"]["access_token"]
	// });

	getAnonymToken2Data, err := doPost(
		"https://api.vk.com/method/calls.getAnonymousAccessTokenPayload",
		map[string]string{
			"v":         v,
			"client_id": clientId,
		},
		map[string]string{
			"access_token": accessToken1,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("vk error %w", err)
	}

	accessToken2 := getValueByPathInJsonData[string](getAnonymToken2Data, "response.payload")
	if accessToken2 == "" {
		return nil, fmt.Errorf("vk error %w", err)
	}

	getAnonymToken3Data, err := doPost(
		"https://login.vk.com/",
		map[string]string{
			"act": "get_anonym_token",
		},
		map[string]string{
			"client_secret": clientSecret,
			"client_id":     clientId,
			"app_id":        appId,
			"version":       "1",
			"token_type":    "messages",
			"payload":       accessToken2,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("vk error %w", err)
	}

	accessToken3 := getValueByPathInJsonData[string](getAnonymToken3Data, "data.access_token")
	if accessToken3 == "" {
		return nil, fmt.Errorf("vk error %w", err)
	}

	_ = joinId
	_ = username

	// responses["https://login.vk.com/?act=get_anonym_token__2"] = await postJson("https://login.vk.com/?act=get_anonym_token", {
	// 	"client_secret": clientSecret,
	// 	"client_id": clientId,
	// 	"app_id": appId,
	// 	"version": "1",

	// 	"token_type": "messages",
	// 	"payload": responses["https://api.vk.com/method/calls.getAnonymousAccessTokenPayload"]["response"]["payload"]
	// });

	// responses["https://api.vk.com/method/calls.getAnonymousToken"] = await postJson("https://api.vk.com/method/calls.getAnonymousToken?v=5.265&client_id=6287487", {
	// 	"vk_join_link": "https://vk.com/call/join/" + joinId,
	// 	"name": username,
	// 	"access_token": responses["https://login.vk.com/?act=get_anonym_token__2"]["data"]["access_token"]
	// });

	// responses["https://calls.okcdn.ru/fb.do__auth.anonymLogin"] = await postJson("https://calls.okcdn.ru/fb.do", {
	// 	"method": "auth.anonymLogin",
	// 	"format": "JSON",
	// 	"application_key": applicationKey,
	// 	"session_data": JSON.stringify({
	// 		"version": 2,
	// 		"device_id": deviceId,
	// 		"client_version": 1.1,
	// 		"client_type": "SDK_JS"
	// 	})
	// });

	// responses["https://calls.okcdn.ru/fb.do__vchat.joinConversationByLink"] = await postJson("https://calls.okcdn.ru/fb.do", {
	// 	"method": "vchat.joinConversationByLink",
	// 	"format": "JSON",
	// 	"application_key": applicationKey,
	// 	"session_key": responses["https://calls.okcdn.ru/fb.do__auth.anonymLogin"]["session_key"],
	// 	"joinLink": joinId,
	// 	"isVideo": false,
	// 	"protocolVersion": 5,
	// 	"anonymToken": responses["https://api.vk.com/method/calls.getAnonymousToken"]["response"]["token"]
	// });

	// const webSocketUrl = responses["https://calls.okcdn.ru/fb.do__vchat.joinConversationByLink"]["endpoint"] + "&platform=WEB&appVersion=1.1&version=5&device=browser&capabilities=2F7F&clientType=VK&tgt=join";

	// return webSocketUrl;

	return nil, nil
}

// const helloMessageFormatString = `{
// 	"uid": "%s",
//     "hello": {
//         "participantMeta": {
//             "name": "super",
//             "role": "SPEAKER",
//             "description": "",
//             "sendAudio": false,
//             "sendVideo": false
//         },
//         "participantAttributes": {
//             "name": "super",
//             "role": "SPEAKER",
//             "description": ""
//         },
//         "sendAudio": false,
//         "sendVideo": false,
//         "sendSharing": false,
//         "participantId": "%s",
//         "roomId": "%s",
//         "serviceName": "telemost",
//         "credentials": "%s",
//         "capabilitiesOffer": {
//             "offerAnswerMode": ["SEPARATE"],
//             "initialSubscriberOffer": ["ON_HELLO"],
//             "slotsMode": ["FROM_CONTROLLER"],
//             "simulcastMode": ["DISABLED", "STATIC"],
//             "selfVadStatus": ["FROM_SERVER", "FROM_CLIENT"],
//             "dataChannelSharing": ["TO_RTP"],
//             "videoEncoderConfig": ["NO_CONFIG", "ONLY_INIT_CONFIG", "RUNTIME_CONFIG"],
//             "dataChannelVideoCodec": ["VP8", "UNIQUE_CODEC_FROM_TRACK_DESCRIPTION"],
//             "bandwidthLimitationReason": ["BANDWIDTH_REASON_DISABLED", "BANDWIDTH_REASON_ENABLED"],
//             "sdkDefaultDeviceManagement": ["SDK_DEFAULT_DEVICE_MANAGEMENT_DISABLED", "SDK_DEFAULT_DEVICE_MANAGEMENT_ENABLED"],
//             "joinOrderLayout": ["JOIN_ORDER_LAYOUT_DISABLED", "JOIN_ORDER_LAYOUT_ENABLED"],
//             "pinLayout": ["PIN_LAYOUT_DISABLED"],
//             "sendSelfViewVideoSlot": ["SEND_SELF_VIEW_VIDEO_SLOT_DISABLED", "SEND_SELF_VIEW_VIDEO_SLOT_ENABLED"],
//             "serverLayoutTransition": ["SERVER_LAYOUT_TRANSITION_DISABLED"],
//             "sdkPublisherOptimizeBitrate": ["SDK_PUBLISHER_OPTIMIZE_BITRATE_DISABLED", "SDK_PUBLISHER_OPTIMIZE_BITRATE_FULL", "SDK_PUBLISHER_OPTIMIZE_BITRATE_ONLY_SELF"],
//             "sdkNetworkLostDetection": ["SDK_NETWORK_LOST_DETECTION_DISABLED"],
//             "sdkNetworkPathMonitor": ["SDK_NETWORK_PATH_MONITOR_DISABLED"],
//             "publisherVp9": ["PUBLISH_VP9_DISABLED", "PUBLISH_VP9_ENABLED"],
//             "svcMode": ["SVC_MODE_DISABLED", "SVC_MODE_L3T3", "SVC_MODE_L3T3_KEY"],
//             "subscriberOfferAsyncAck": ["SUBSCRIBER_OFFER_ASYNC_ACK_DISABLED", "SUBSCRIBER_OFFER_ASYNC_ACK_ENABLED"],
//             "svcModes": ["FALSE"],
//             "reportTelemetryModes": ["TRUE"],
//             "keepDefaultDevicesModes": ["TRUE"]
//         },
//         "sdkInfo": {
//             "implementation": "browser",
//             "version": "5.15.0",
//             "userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
//             "hwConcurrency": 8
//         },
//         "sdkInitializationId": "%s",
//         "disablePublisher": false,
//         "disableSubscriber": false,
//         "disableSubscriberAudio": false
//     }
// }`

// func getIceServerFromYandexTelemostWebSocketSignalServer(info *YandexTelemostWebSocketSignalServerInfo) (*webrtc.ICEServer, error) {
// 	webSocketConn, _, err := websocket.DefaultDialer.Dial(info.ClientConfiguration.MediaServerUrl, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer webSocketConn.Close()

// 	webSocketConn.WriteMessage(websocket.TextMessage, []byte(
// 		fmt.Sprintf(helloMessageFormatString,
// 			uuid.New().String(),
// 			info.ParticipantId,
// 			info.RoomId,
// 			info.Credentials,
// 			uuid.New().String(),
// 		),
// 	))

// 	waitMessagesAmount := 5
// 	for {
// 		if waitMessagesAmount == 0 {
// 			return nil, errors.New("Bad response")
// 		}

// 		err := webSocketConn.SetReadDeadline(time.Now().Add(5 * time.Second))
// 		if err != nil {
// 			return nil, err
// 		}

// 		_, buf, err := webSocketConn.ReadMessage()
// 		if err != nil {
// 			return nil, err
// 		}

// 		var msg struct {
// 			ServerHello struct {
// 				RtcConfiguration struct {
// 					IceServers []*webrtc.ICEServer `json:"iceServers"`
// 				} `json:"rtcConfiguration"`
// 			} `json:"serverHello"`
// 		}

// 		err = json.Unmarshal(buf, &msg)
// 		if err == nil {
// 			for _, iceServer := range msg.ServerHello.RtcConfiguration.IceServers {
// 				for _, url := range iceServer.URLs {
// 					if strings.HasPrefix(url, "turn:") &&
// 						strings.Contains(url, "transport=tcp") {

// 						return iceServer, nil
// 					}
// 				}
// 			}
// 		}

// 		waitMessagesAmount--
// 	}
// }

// func GetIceServerFromJoinIdOrLink(joinIdOrLink string) (*webrtc.ICEServer, error) {
// 	joinId, err := getJoinId(joinIdOrLink)
// 	if err != nil {
// 		return nil, err
// 	}

// 	info, err := getYandexTelemostWebSocketSignalServerInfoByJoinId(joinId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	iceServer, err := getIceServerFromYandexTelemostWebSocketSignalServer(info)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return iceServer, nil
// }
