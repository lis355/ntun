package node

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/internal/app"
	"ntun/internal/connections"
	"ntun/internal/log"
	ntunConnections "ntun/internal/ntun/connections"
	"ntun/internal/proxy"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-yamux"
)

const (
	transportRetryingTimeout = 3 * time.Second
)

type ConnectionManager struct {
	node          *Node
	outputDialer  ntunConnections.Dialer // TODO hack сделать абстракцию
	transportConn net.Conn
	inHs, outHs   *TransportHandshake
	client        bool
	// mux           *mux.Mux
	session      *yamux.Session
	wasConnected bool
}

func NewConnManager(node *Node, outputDialer ntunConnections.Dialer) *ConnectionManager {
	return &ConnectionManager{
		node:         node,
		outputDialer: outputDialer,
	}
}

func (m *ConnectionManager) Start() error {
	go m.process()

	return nil
}

func (m *ConnectionManager) Stop() error {
	return nil
}

func (m *ConnectionManager) clear() {
	// if m.mux != nil {
	// 	m.mux.Close()
	// 	m.mux = nil
	// }

	if m.session != nil {
		m.session.Close()
		m.session = nil
	}

	if m.transportConn != nil {
		m.transportConn.Close()
		m.transportConn = nil
	}
}

func (m *ConnectionManager) process() {
	if err := m.node.Transporter.Listen(); err != nil {
		slog.Warn(fmt.Sprintf("%s: transport listen error: %v", log.ObjName(m), err))
	}

	for {
		transportConn, err := m.node.Transporter.Transport()
		if err == nil {
			m.handleTransportConn(transportConn)
		}

		if !m.wasConnected {
			slog.Debug(fmt.Sprintf("%s: can't get transport, waiting %s", log.ObjName(m), transportRetryingTimeout))

			time.Sleep(transportRetryingTimeout)
		}

		m.wasConnected = false
	}
}

func (m *ConnectionManager) handleTransportConn(transportConn net.Conn) {
	m.clear()

	m.transportConn = transportConn

	// m.transportConn = dev.NewSnifferHexDumpDebugConn(m.transportConn, fmt.Sprintf("[%s:transportConn]", log.ObjName(m)), true)

	err := m.cipherTransportConn()
	if err != nil {
		m.clear()

		return
	}

	err = m.doTransportHandshake()
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: bad hs connection %s %v", log.ObjName(m), transportConn.RemoteAddr().String(), err))

		m.clear()

		return
	}

	// DEBUG turn off yamux warnings about tcp resets
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard

	var session *yamux.Session
	if m.client {
		session, err = yamux.Client(m.transportConn, config)
	} else {
		session, err = yamux.Server(m.transportConn, config)
	}
	if err != nil {
		m.clear()

		return
	}

	m.wasConnected = true

	slog.Info(fmt.Sprintf("%s: node %s connected", log.ObjName(m), m.inHs.Id.String()))

	m.session = session

	// m.mux = mux.NewMux(m.transportConn)
	// muxListener, err := m.mux.Listen()
	// if err != nil {
	// 	m.clear()

	// 	return
	// }

	for {
		conn, err := m.session.Accept()
		// conn, err := muxListener.Accept()
		if err != nil {
			slog.Info(fmt.Sprintf("%s: node %s disconnected %v", log.ObjName(m), m.inHs.Id.String(), err))

			m.clear()

			return
		}

		go m.handleMuxConn(conn)
	}
}

func (m *ConnectionManager) cipherTransportConn() error {
	cipherAesGcmConn, err := connections.NewCipherAesGcmConn(m.transportConn, []byte(m.node.Config.CipherKey))
	if err != nil {
		return err
	}

	m.transportConn = cipherAesGcmConn

	return nil
}

type TransportHandshake struct {
	Version string
	Id      uuid.UUID
}

func WriteMsg[T any](conn net.Conn, msg *T) error {
	msgBuf, err := json.Marshal(&msg)
	if err != nil {
		return err
	}

	msgLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBuf, uint32(len(msgBuf)))

	n, err := conn.Write(msgLenBuf)
	if err != nil {
		return err
	}
	if n != len(msgLenBuf) {
		return fmt.Errorf("bad write %d bytes, expected %d", n, len(msgLenBuf))
	}

	n, err = conn.Write(msgBuf)
	if err != nil {
		return err
	}
	if n != len(msgBuf) {
		return fmt.Errorf("bad write %d bytes, expected %d", n, len(msgBuf))
	}

	return nil
}

func ReadMsg[T any](conn net.Conn, msg *T) error {
	msgLenBuf := make([]byte, 4)
	n, err := conn.Read(msgLenBuf)
	if err != nil {
		return err
	}
	if n != len(msgLenBuf) {
		return fmt.Errorf("bad read %d bytes, expected %d", n, len(msgLenBuf))
	}

	msgLen := binary.BigEndian.Uint32(msgLenBuf)

	msgBuf := make([]byte, msgLen)
	n, err = conn.Read(msgBuf)
	if err != nil {
		return err
	}
	if n != len(msgBuf) {
		return fmt.Errorf("bad read %d bytes, expected %d", n, len(msgBuf))
	}

	err = json.Unmarshal(msgBuf, &msg)
	if err != nil {
		return err
	}

	return nil
}

func CmpUUID(a, b uuid.UUID) int {
	for i := 0; i < 16; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func (m *ConnectionManager) doTransportHandshake() error {
	m.outHs = &TransportHandshake{Version: app.Version, Id: m.node.Config.Id}

	err := WriteMsg(m.transportConn, m.outHs)
	// slog.Debug(fmt.Sprintf("%s: written transport hs %+v", log.ObjName(m), m.outHs))
	if err != nil {
		return err
	}

	err = ReadMsg(m.transportConn, &m.inHs)
	// slog.Debug(fmt.Sprintf("%s: readed transport hs %+v", log.ObjName(m), m.inHs))
	if err != nil {
		return err
	}

	if m.inHs.Version != m.outHs.Version {
		return fmt.Errorf("handshake version mismatch %s != %s", m.inHs.Version, m.outHs.Version)
	}

	cmpId := CmpUUID(m.outHs.Id, m.inHs.Id)

	if cmpId == 0 {
		return fmt.Errorf("handshake bad ids %s %s", m.outHs.Id, m.inHs.Id)
	}

	if !m.node.HasAllowedToConnectNodeId(m.inHs.Id) {
		return fmt.Errorf("handshake not allowed node with id %s", m.inHs.Id)
	}

	// NOTE нужно для создания yamux, т.к. айдишники нод всегда разные,
	// какой-то из них математически меньше другого - пусть он будет yamux клиентом
	m.client = cmpId < 0

	return nil
}

type ConnectMsg struct {
	Address string
}

func (m *ConnectionManager) handleMuxConn(conn net.Conn) {
	var msg ConnectMsg
	err := ReadMsg(conn, &msg)
	if err != nil {
		conn.Close()

		return
	}

	if m.outputDialer == nil {
		conn.Close()

		slog.Debug(fmt.Sprintf("%s: current node has not outputs", log.ObjName(m)))

		return
	}

	// slog.Debug(fmt.Sprintf("%s: mux stream accepted connect to %s", log.ObjName(m), msg.Address))

	outConn, err := m.outputDialer.Dial(msg.Address)
	if err != nil {
		conn.Close()

		return
	}

	// DEBUG
	// conn = dev.NewSnifferHexDumpDebugConn(conn, fmt.Sprintf("direct"), false)

	// DEBUG
	protocolDetectorConn := connections.NewProtocolDetectorConn(outConn)
	outConn = protocolDetectorConn

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		protocol := <-protocolDetectorConn.Detected
		switch pr := protocol.(type) {
		case *connections.HttpProtocol:
			slog.Info(fmt.Sprintf("%s: detected %s protocol", log.ObjName(m), pr.Protocol()))
		case *connections.HttpsProtocol:
			slog.Info(fmt.Sprintf("%s: detected %s protocol %s", log.ObjName(m), pr.Protocol(), pr.Domain))
		}
	}()

	proxy.Proxy(conn, outConn)

	wg.Wait()
}

func (m *ConnectionManager) Dial(dstAddress string) (net.Conn, error) {
	if m.session == nil {
		return nil, net.ErrClosed
	}

	dstConn, err := m.session.Open()
	if err != nil {
		return nil, err
	}

	// // if mux created, it means hs passed and connection established
	// if m.mux == nil {
	// 	return nil, net.ErrClosed
	// }

	// dstConn, err := m.mux.CreateStream()
	// if err != nil {
	// 	return nil, err
	// }

	// slog.Debug(fmt.Sprintf("%s: mux stream created %s <--> %s", log.ObjName(m), srcAddress, dstAddress))

	err = WriteMsg(dstConn, &ConnectMsg{Address: dstAddress})
	if err != nil {
		return nil, err
	}

	return dstConn, nil
}
