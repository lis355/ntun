package mux

import (
	"io"
	"net"

	"github.com/libp2p/go-yamux"
)

// const uint32Len uint32 = 4

// type streamId uuid.UUID

// const streamIdLen uint32 = 16

// func (id streamId) String() string {
// 	return uuid.UUID(id).String()
// }

// type stream struct {
// 	mux    *Mux
// 	id     streamId
// 	closed bool

// 	readBuf []byte
// 	readCh  chan []byte
// }

// func newStream(mux *Mux, id streamId) *stream {
// 	return &stream{
// 		mux:    mux,
// 		id:     id,
// 		readCh: make(chan []byte),
// 	}
// }

type Mux struct {
	// conn    net.Conn
	// closed  bool
	// streams map[streamId]*stream
	// mu      sync.Mutex // TODO optimize over channels ?

	// acceptCh chan *stream
	// writeCh  chan messanger

	// TODO write own
	session *yamux.Session
}

func NewMux(conn net.Conn, client bool) (*Mux, error) {
	// NOTE turn off yamux warnings about tcp resets
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard

	var sessionCreator func(conn net.Conn, config *yamux.Config) (*yamux.Session, error)
	if client {
		sessionCreator = yamux.Client
	} else {
		sessionCreator = yamux.Server
	}

	session, err := sessionCreator(conn, config)
	if err != nil {
		return nil, err
	}

	m := &Mux{
		// conn:     conn,
		// streams:  make(map[streamId]*stream),
		// acceptCh: make(chan *stream),
		// writeCh:  make(chan messanger),
		session: session,
	}

	return m, nil
}

// type msgType byte

// const (
// 	msgTypeStreamConnect msgType = iota
// 	msgTypeStreamDisconnect
// 	msgTypeStreamData

// 	msgTypeLen   uint32 = 1
// 	msgHeaderLen uint32 = msgTypeLen + uint32Len
// )

// type messanger interface {
// 	Type() msgType
// 	Encode() ([]byte, error)
// 	Decode([]byte) error
// }

// type msg struct {
// 	msgType
// }

// func (m *msg) Type() msgType {
// 	return m.msgType
// }

// type msgStream struct {
// 	msg
// 	id streamId
// }

// type msgStreamConnect struct {
// 	msgStream
// }

// func (m *msgStream) EncodeStreamId(buf []byte, offset uint32) error {
// 	n := copy(buf[offset:offset+streamIdLen], m.id[:])
// 	if uint32(n) != streamIdLen {
// 		return fmt.Errorf("Bad encoding")
// 	}

// 	return nil
// }

// func (m *msgStream) DecodeStreamId(buf []byte, offset uint32) error {
// 	n := copy(m.id[:], buf[offset:offset+streamIdLen])
// 	if uint32(n) != streamIdLen {
// 		return fmt.Errorf("Bad decoding")
// 	}

// 	return nil
// }

// func (m *msgStreamConnect) Encode() ([]byte, error) {
// 	msgLen := streamIdLen
// 	buf := make([]byte, msgLen)

// 	err := m.EncodeStreamId(buf, 0)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return buf, nil
// }

// func (m *msgStreamConnect) Decode(buf []byte) error {
// 	err := m.DecodeStreamId(buf, 0)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// type msgStreamDisconnect struct {
// 	msgStream
// }

// func (m *msgStreamDisconnect) Encode() ([]byte, error) {
// 	msgLen := streamIdLen
// 	buf := make([]byte, msgLen)

// 	err := m.EncodeStreamId(buf, 0)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return buf, nil
// }

// func (m *msgStreamDisconnect) Decode(buf []byte) error {
// 	err := m.DecodeStreamId(buf, 0)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// type msgStreamData struct {
// 	msgStream
// 	data []byte
// }

// func (m *msgStreamData) Encode() ([]byte, error) {
// 	dataLen := uint32(len(m.data))
// 	msgLen := streamIdLen + uint32Len + dataLen
// 	buf := make([]byte, msgLen)
// 	var offset uint32 = 0

// 	err := m.EncodeStreamId(buf, offset)
// 	if err != nil {
// 		return nil, err
// 	}
// 	offset += streamIdLen

// 	binary.BigEndian.PutUint32(buf[offset:offset+uint32Len], dataLen)
// 	offset += uint32Len

// 	n := copy(buf[offset:], m.data)
// 	if uint32(n) != dataLen {
// 		return nil, fmt.Errorf("Bad encoding")
// 	}

// 	return buf, nil
// }

// func (m *msgStreamData) Decode(buf []byte) error {
// 	var offset uint32 = 0

// 	err := m.DecodeStreamId(buf, offset)
// 	if err != nil {
// 		return err
// 	}
// 	offset += streamIdLen

// 	dataLen := binary.BigEndian.Uint32(buf[offset : offset+uint32Len])
// 	offset += uint32Len

// 	m.data = buf[offset : offset+dataLen]
// 	if uint32(len(m.data)) != dataLen {
// 		return fmt.Errorf("Bad decoding")
// 	}

// 	return nil
// }

func (m *Mux) Listen() error {
	return nil

	// go m.doRead()
	// go m.doWrite()

	// return nil
}

// func (m *Mux) doRead() error {
// 	headerBuf := make([]byte, 5)
// 	for {
// 		_, err := io.ReadFull(m.conn, headerBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		var offset uint32 = 0
// 		msgType := msgType(headerBuf[offset])
// 		offset++

// 		msgLen := binary.BigEndian.Uint32(headerBuf[offset : offset+uint32Len])
// 		msgBuf := make([]byte, msgLen)
// 		_, err = io.ReadFull(m.conn, msgBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		err = m.processReadMessage(msgType, msgBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}
// 	}
// }

// func (m *Mux) processReadMessage(msgType msgType, msgBuf []byte) error {
// 	switch msgType {
// 	case msgTypeStreamConnect:
// 		var msg msgStreamConnect
// 		err := msg.Decode(msgBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		slog.Debug(fmt.Sprintf("%p mux recieve msgStreamConnect id=%s", m, msg.id))

// 		m.mu.Lock()
// 		_, ok := m.streams[msg.id]
// 		m.mu.Unlock()

// 		if ok {
// 			return fmt.Errorf("stream already exists")
// 		}

// 		m.mu.Lock()
// 		stream := newStream(m, msg.id)
// 		m.streams[msg.id] = stream
// 		m.mu.Unlock()

// 		m.acceptCh <- stream

// 	case msgTypeStreamDisconnect:
// 		var msg msgStreamDisconnect
// 		err := msg.Decode(msgBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		slog.Debug(fmt.Sprintf("%p mux recieve msgStreamDisconnect id=%s", m, msg.id))

// 		m.mu.Lock()
// 		stream, ok := m.streams[msg.id]
// 		m.mu.Unlock()

// 		if !ok {
// 			return m.shutdown(fmt.Errorf("stream not exists"))
// 		}

// 		m.doCloseStream(stream)

// 	case msgTypeStreamData:
// 		var msg msgStreamData
// 		err := msg.Decode(msgBuf)
// 		if err != nil {
// 			return err
// 		}

// 		// slog.Debug(fmt.Sprintf("%p mux recieve msgStreamData id=%s data=%s", m, msg.id, utils.BytesToASCIIHexDumpString(msg.data)))

// 		m.mu.Lock()
// 		stream, ok := m.streams[msg.id]
// 		m.mu.Unlock()

// 		if !ok {
// 			return m.shutdown(fmt.Errorf("stream not exists"))
// 		}

// 		stream.readCh <- msg.data

// 	default:
// 		return m.shutdown(fmt.Errorf("unknown message type"))
// 	}

// 	return nil
// }

// func (m *Mux) doWrite() error {
// 	for {
// 		msg, ok := <-m.writeCh
// 		if !ok {
// 			return nil
// 		}

// 		msgBuf, err := msg.Encode()
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		var offset uint32 = 0
// 		headerBuf := make([]byte, msgHeaderLen)
// 		headerBuf[offset] = byte(msg.Type())
// 		offset++

// 		msgLen := uint32(len(msgBuf))
// 		binary.BigEndian.PutUint32(headerBuf[offset:offset+uint32Len], msgLen)

// 		_, err = m.conn.Write(headerBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}

// 		_, err = m.conn.Write(msgBuf)
// 		if err != nil {
// 			return m.shutdown(err)
// 		}
// 	}
// }

func (m *Mux) CreateStream() (net.Conn, error) {
	return m.session.Open()

	// stream, err := m.doCreateStream()
	// if err != nil {
	// 	return nil, m.shutdown(err)
	// }

	// slog.Debug(fmt.Sprintf("%p mux send msgStreamConnect id=%s", m, stream.id))
	// m.writeMessage(&msgStreamConnect{msgStream{msg{msgTypeStreamConnect}, stream.id}})

	// return stream, nil
}

func (m *Mux) Accept() (net.Conn, error) {
	return m.session.Accept()

	// stream, ok := <-m.acceptCh
	// if !ok {
	// 	return nil, io.EOF
	// }

	// return stream, nil
}

func (m *Mux) Close() error {
	return m.session.Close()

	// return m.shutdown(nil)
}

// func (m *Mux) Addr() net.Addr {
// 	return m.conn.LocalAddr()
// }

// func (m *Mux) shutdown(err error) error {
// 	m.mu.Lock()
// 	if m.closed {
// 		m.mu.Unlock()
// 		return nil
// 	}

// 	m.closed = true
// 	m.mu.Unlock()

// 	m.closeAllStreams()
// 	close(m.writeCh)
// 	m.conn.Close()

// 	return err
// }

// func (m *Mux) writeMessage(msg messanger) {
// 	m.writeCh <- msg
// }

// func (m *Mux) doCreateStream() (*stream, error) {
// 	id := streamId(uuid.New())

// 	m.mu.Lock()
// 	_, ok := m.streams[id]
// 	m.mu.Unlock()

// 	if ok {
// 		return nil, fmt.Errorf("stream already exists")
// 	}

// 	m.mu.Lock()
// 	stream := newStream(m, id)
// 	m.streams[id] = stream
// 	m.mu.Unlock()

// 	return stream, nil
// }

// func (m *Mux) doCloseStream(stream *stream) {
// 	m.mu.Lock()
// 	stream.closed = true

// 	delete(m.streams, stream.id)

// 	stream.readBuf = nil
// 	m.mu.Unlock()

// 	close(stream.readCh)
// }

// func (m *Mux) closeAllStreams() {
// 	for _, stream := range m.streams {
// 		m.doCloseStream(stream)
// 	}

// 	close(m.acceptCh)
// }

// func (s *stream) Read(b []byte) (n int, err error) {
// 	if s.closed {
// 		return 0, io.EOF
// 	}

// 	if len(b) == 0 {
// 		return 0, nil
// 	}

// 	if len(s.readBuf) > 0 {
// 		n = copy(b, s.readBuf)
// 		s.readBuf = s.readBuf[n:]

// 		return n, nil
// 	}

// 	if s.readCh == nil {
// 		return 0, io.EOF
// 	}

// 	buf, ok := <-s.readCh
// 	if !ok {
// 		return 0, io.EOF
// 	}

// 	s.readBuf = append(s.readBuf, buf...)

// 	n = copy(b, s.readBuf)
// 	s.readBuf = s.readBuf[n:]

// 	return n, nil
// }

// func (s *stream) Write(b []byte) (n int, err error) {
// 	if s.closed {
// 		return 0, fmt.Errorf("write to closed stream")
// 	}

// 	if len(b) == 0 {
// 		return 0, nil
// 	}

// 	// slog.Debug(fmt.Sprintf("%p mux send msgTypeStreamData id=%s data=%s", s.mux, s.id, utils.BytesToASCIIHexDumpString(b)))
// 	s.mux.writeMessage(&msgStreamData{msgStream{msg{msgTypeStreamData}, s.id}, b})

// 	return len(b), nil
// }

// func (s *stream) Close() error {
// 	if s.closed {
// 		return nil
// 	}

// 	s.mux.doCloseStream(s)

// 	slog.Debug(fmt.Sprintf("%p mux send msgStreamDisconnect id=%s", s.mux, s.id))
// 	s.mux.writeMessage(&msgStreamDisconnect{msgStream{msg{msgTypeStreamDisconnect}, s.id}})

// 	return nil
// }

// func (s *stream) LocalAddr() net.Addr {
// 	return s.mux.conn.LocalAddr()
// }

// func (s *stream) RemoteAddr() net.Addr {
// 	return s.mux.conn.RemoteAddr()
// }

// func (s *stream) SetDeadline(t time.Time) error {
// 	return s.mux.conn.SetDeadline(t)
// }

// func (s *stream) SetReadDeadline(t time.Time) error {
// 	return s.mux.conn.SetReadDeadline(t)
// }

// func (s *stream) SetWriteDeadline(t time.Time) error {
// 	return s.mux.conn.SetWriteDeadline(t)
// }
