package ntun

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"ntun/internal/app"
	"ntun/internal/dev"
	"ntun/internal/mux"

	"github.com/google/uuid"
)

type ConnManager struct {
	node          *Node
	transporter   Transporter
	transportConn net.Conn
	mux           *mux.Mux
}

func NewConnManager(node *Node) *ConnManager {
	return &ConnManager{
		node:        node,
		transporter: node.Transporter,
	}
}

func (m *ConnManager) Start() error {
	// DEBUG
	//go m.process()

	return nil
}

func (m *ConnManager) Stop() error {
	return nil
}

func (m *ConnManager) clear() {
	if m.mux != nil {
		m.mux.Close()
		m.mux = nil
	}

	if m.transportConn != nil {
		m.transportConn.Close()
		m.transportConn = nil
	}
}

func (m *ConnManager) process() {
	for transportConn := range m.transporter.Transport() {
		m.handleTransportConn(transportConn)
	}
}

func (m *ConnManager) handleTransportConn(transportConn net.Conn) {
	m.clear()

	m.transportConn = transportConn

	err := m.cipherTransportConn()
	if err != nil {
		m.clear()

		return
	}

	// m.transportConn = dev.NewSnifferHexDumpDebugConn(m.transportConn, fmt.Sprintf("[%s:ConnManager:transportConn]", m.node.String()), false)

	err = m.doTransportHandshake()
	if err != nil {
		slog.Warn(fmt.Sprintf("[%s:ConnManager] bad hs connection %s", m.node.String(), transportConn.RemoteAddr().String()))

		m.clear()

		return
	}

	m.mux = mux.NewMux(m.transportConn)
	muxListener, err := m.mux.Listen()
	if err != nil {
		m.clear()

		return
	}

	go func() {
		for {
			conn, err := muxListener.Accept()
			if err != nil {
				m.clear()

				return
			}

			go m.handleMuxConn(conn)
		}
	}()
}

func (m *ConnManager) cipherTransportConn() error {
	// cipherAesGcmConn, err := connections.NewCipherAesGcmConn(m.transportConn, []byte(m.node.Config.CipherKey))
	// if err != nil {
	// 	return err
	// }

	// m.transportConn = cipherAesGcmConn

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
		return fmt.Errorf("Bad write %d bytes, expected %d", n, len(msgLenBuf))
	}

	n, err = conn.Write(msgBuf)
	if err != nil {
		return err
	}
	if n != len(msgBuf) {
		return fmt.Errorf("Bad write %d bytes, expected %d", n, len(msgBuf))
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
		return fmt.Errorf("Bad read %d bytes, expected %d", n, len(msgLenBuf))
	}

	msgLen := binary.BigEndian.Uint32(msgLenBuf)

	msgBuf := make([]byte, msgLen)
	n, err = conn.Read(msgBuf)
	if err != nil {
		return err
	}
	if n != len(msgBuf) {
		return fmt.Errorf("Bad read %d bytes, expected %d", n, len(msgBuf))
	}

	err = json.Unmarshal(msgBuf, &msg)
	if err != nil {
		return err
	}

	return nil
}

func (m *ConnManager) doTransportHandshake() error {
	outHs := &TransportHandshake{Version: app.Version, Id: m.node.Config.Id}

	err := WriteMsg(m.transportConn, outHs)
	slog.Debug(fmt.Sprintf("[%s:ConnManager] written transport hs %+v", m.node.String(), outHs))
	if err != nil {
		return err
	}

	var inHs TransportHandshake
	err = ReadMsg(m.transportConn, &inHs)
	slog.Debug(fmt.Sprintf("[%s:ConnManager] readed transport hs %+v", m.node.String(), &inHs))
	if err != nil {
		return err
	}

	if inHs.Version != outHs.Version {
		return fmt.Errorf("Handshake version mismatch %s != %s", inHs.Version, outHs.Version)
	}

	if !m.node.HasAllowedToConnectNodeId(inHs.Id) {
		return fmt.Errorf("Handshake not allowed node with id %s", inHs.Id.String())
	}

	slog.Info(fmt.Sprintf("[%s:ConnManager] node %s connected", m.node.String(), inHs.Id.String()))

	return nil
}

type ConnectMsg struct {
	Address string
}

func (m *ConnManager) handleMuxConn(conn net.Conn) {
	var msg ConnectMsg
	err := ReadMsg(conn, &msg)
	if err != nil {
		conn.Close()

		return
	}

	// slog.Debug(fmt.Sprintf("[%s:ConnManager] mux stream accepted connect to %s", m.node.String(), msg.Address))

	outConn, err := net.Dial("tcp", msg.Address)
	if err != nil {
		conn.Close()

		return
	}

	// DEBUG
	conn = dev.NewSnifferHexDumpDebugConn(conn, fmt.Sprintf("direct"), false)

	// DEBUG
	// protocolDetectorConn := connections.NewProtocolDetectorConn(outConn)
	// outConn = protocolDetectorConn

	// var wg sync.WaitGroup
	// wg.Add(1)

	// go func() {
	// 	defer wg.Done()

	// 	protocol := <-protocolDetectorConn.Detected
	// 	switch pr := protocol.(type) {
	// 	case *connections.HttpProtocol:
	// 		slog.Info(fmt.Sprintf("[%s:ConnManager] detected %s protocol", m.node.String(), pr.Protocol()))
	// 	case *connections.HttpsProtocol:
	// 		slog.Info(fmt.Sprintf("[%s:ConnManager] detected %s protocol %s", m.node.String(), pr.Protocol(), pr.Domain))
	// 	}
	// }()

	err = Proxy(conn, outConn)
	if err != nil {
		return
	}

	// wg.Wait()
}

func (m *ConnManager) Dial(srcAddress, dstAddress string) (net.Conn, error) {
	// DEBUG
	return net.Dial("tcp", dstAddress)

	// if mux created, it means hs passed and connection established
	if m.mux == nil {
		return nil, net.ErrClosed
	}

	dstConn, err := m.mux.CreateStream()
	if err != nil {
		return nil, err
	}

	// slog.Debug(fmt.Sprintf("[%s:ConnManager] mux stream created %s <--> %s", m.node.String(), srcAddress, dstAddress))

	err = WriteMsg(dstConn, &ConnectMsg{Address: dstAddress})
	if err != nil {
		return nil, err
	}

	return dstConn, nil
}
