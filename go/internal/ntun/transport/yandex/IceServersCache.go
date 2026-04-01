package yandex

import (
	"errors"
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/log"
	"time"

	"github.com/pion/webrtc/v3"
	"go.yaml.in/yaml/v3"
)

const (
	iceServersCacheTimeout  = 1 * time.Hour
	iceServersCacheFileName = "iceServers.data"
)

type IceServersCache struct {
	Time       time.Time
	IceServers []webrtc.ICEServer
}

func (y *YandexWebRTCTransport) getIceServer() (*webrtc.ICEServer, error) {
	iceServers := y.loadIceServersCache()
	if iceServers == nil ||
		len(*iceServers) == 0 {
		slog.Debug(fmt.Sprintf("%s: trying to get iceServers from yandex server", log.ObjName(y)))

		iceServer, err := GetIceServerFromJoinIdOrLink(y.cfg.JoinId)
		if err != nil {
			slog.Debug(fmt.Sprintf("%s: get iceServers from yandex server error %v", log.ObjName(y), err))

			return nil, err
		}

		slog.Debug(fmt.Sprintf("%s: success got iceServers from yandex server, caching", log.ObjName(y)))

		iceServers = &[]webrtc.ICEServer{*iceServer}

		y.saveIceServersCache(iceServers)
	}

	if len(*iceServers) == 0 {
		return nil, errors.New("empty iceServers")
	}

	return &(*iceServers)[0], nil
}

func (y *YandexWebRTCTransport) loadIceServersCache() *[]webrtc.ICEServer {
	slog.Debug(fmt.Sprintf("%s: trying to read iceServers from cache", log.ObjName(y)))

	iceServersCacheBuf, err := app.ReadCacheFile(iceServersCacheFileName)
	if err != nil {
		return nil
	}

	iceServersCacheBuf, err = y.signaling.cipher.Decrypt(iceServersCacheBuf)
	if err != nil {
		return nil
	}

	var iceServersCache IceServersCache
	if err := yaml.Unmarshal(iceServersCacheBuf, &iceServersCache); err != nil {
		return nil
	}

	if time.Since(iceServersCache.Time) < iceServersCacheTimeout &&
		len(iceServersCache.IceServers) > 0 {
		slog.Debug(fmt.Sprintf("%s: iceServers readed from cache", log.ObjName(y)))

		return &iceServersCache.IceServers
	}

	slog.Debug(fmt.Sprintf("%s: iceServers in cache is obsolete", log.ObjName(y)))

	return nil
}

func (y *YandexWebRTCTransport) saveIceServersCache(iceServers *[]webrtc.ICEServer) {
	iceServersCache := &IceServersCache{Time: time.Now(), IceServers: *iceServers}
	iceServersCacheBuf, err := yaml.Marshal(&iceServersCache)
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}

	iceServersCacheBuf, err = y.signaling.cipher.Encrypt(iceServersCacheBuf)
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}

	if err := app.WriteCacheFile(iceServersCacheFileName, iceServersCacheBuf); err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}

	slog.Debug(fmt.Sprintf("%s: iceServers saved to cache", log.ObjName(y)))
}
