package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"ntun/internal/app"
	"ntun/internal/log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
)

type loggingPacketConn struct {
	net.PacketConn
}

func (l *loggingPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = l.PacketConn.WriteTo(p, addr)
	if err == nil {
		slog.Info(fmt.Sprintf("OUT -> %s bytes=%d", addr.String(), n))
	}
	return n, err
}

func (l *loggingPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = l.PacketConn.ReadFrom(p)
	if err == nil {
		slog.Info(fmt.Sprintf("IN  <- %s bytes=%d", addr.String(), n))
	}
	return n, addr, err
}

type loggingConn struct {
	net.Conn
}

func (l *loggingConn) Read(b []byte) (n int, err error) {
	n, err = l.Conn.Read(b)
	if err == nil {
		slog.Info(fmt.Sprintf("IN  <- %s bytes=%d", l.RemoteAddr().String(), n))
	}
	return n, err
}

func (l *loggingConn) Write(b []byte) (n int, err error) {
	n, err = l.Conn.Write(b)
	if err == nil {
		slog.Info(fmt.Sprintf("OUT -> %s bytes=%d", l.RemoteAddr().String(), n))
	}
	return n, err
}

type loggingTCPListener struct {
	net.Listener
}

func (l *loggingTCPListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("TCP ACCEPT from %s", conn.RemoteAddr().String()))

	return &loggingConn{Conn: conn}, nil
}

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()

	publicIp := os.Getenv("DEVELOP_WEB_RTC_TURN_SERVER_IP")
	publicPort, err := strconv.Atoi(os.Getenv("DEVELOP_WEB_RTC_TURN_SERVER_PORT"))
	if err != nil {
		publicPort = 3478
	}

	serverLocalAdress := fmt.Sprintf("0.0.0.0:%d", publicPort)

	loggerFactory := logging.NewDefaultLoggerFactory()

	udpListener, err := net.ListenPacket("udp4", serverLocalAdress)
	if err != nil {
		panic(err)
	}
	udpListener = &loggingPacketConn{PacketConn: udpListener}

	tcpListener, err := net.Listen("tcp4", serverLocalAdress)
	if err != nil {
		panic(err)
	}
	tcpListener = &loggingTCPListener{Listener: tcpListener}

	user := "user"
	pass := "pass"
	realm := "realm"
	authKey := turn.GenerateAuthKey(user, pass, realm)

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: realm,
		AuthHandler: func(username, userRealm string, srcAddr net.Addr) ([]byte, bool) {
			if username == user &&
				userRealm == realm {
				return authKey, true
			}

			return nil, false
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIp),
					Address:      "0.0.0.0",
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIp),
					Address:      "0.0.0.0",
				},
			},
		},
		LoggerFactory: loggerFactory,
	})

	if err != nil {
		panic(err)
	}

	defer s.Close()

	slog.Info(fmt.Sprintf("TURN server started on %s", serverLocalAdress))

	iceServer := &webrtc.ICEServer{
		URLs: []string{
			fmt.Sprintf("turn:%s:%d?transport=udp", publicIp, publicPort),
		},
		Username:   user,
		Credential: pass,
	}

	jsonBuf, _ := json.Marshal(iceServer)

	slog.Info(fmt.Sprintf("%s", jsonBuf))

	iceServer = &webrtc.ICEServer{
		URLs: []string{
			fmt.Sprintf("turn:%s:%d?transport=tcp", publicIp, publicPort),
		},
		Username:   user,
		Credential: pass,
	}

	jsonBuf, _ = json.Marshal(iceServer)

	slog.Info(fmt.Sprintf("%s", jsonBuf))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
