package connections

import (
	"net"
	"sync"
	"time"
)

// BufferedConn буферизирует данные и отправляет пакетами не меньше maxSize
// но не позже чем через maxDelay после первого байта в буфере
type BufferedConn struct {
	net.Conn
	mu       sync.Mutex
	buffer   []byte
	maxSize  int
	maxDelay time.Duration
	timer    *time.Timer
	closeCh  chan struct{}
	writeCh  chan []byte
}

const (
	defaultMaxSize  = 4096
	defaultMaxDelay = 10 * time.Millisecond
)

func NewBufferedConn(conn net.Conn, maxSize int, maxDelay time.Duration) *BufferedConn {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	if maxDelay <= 0 {
		maxDelay = defaultMaxDelay
	}

	bc := &BufferedConn{
		Conn:     conn,
		buffer:   make([]byte, 0, maxSize),
		maxSize:  maxSize,
		maxDelay: maxDelay,
		closeCh:  make(chan struct{}),
		writeCh:  make(chan []byte, 100),
	}

	go bc.writer()

	return bc
}

func (bc *BufferedConn) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.buffer = append(bc.buffer, data...)

	if len(bc.buffer) >= bc.maxSize {
		bc.flushLocked()

		return len(data), nil
	}

	if len(bc.buffer) == len(data) &&
		bc.timer == nil {
		bc.timer = time.AfterFunc(bc.maxDelay, bc.onTimeout)
	}

	return len(data), nil
}

func (bc *BufferedConn) flushLocked() error {
	if len(bc.buffer) == 0 {
		return nil
	}

	if bc.timer != nil {
		bc.timer.Stop()
		bc.timer = nil
	}

	data := make([]byte, len(bc.buffer))
	copy(data, bc.buffer)
	bc.buffer = bc.buffer[:0]

	select {
	case bc.writeCh <- data:
	case <-bc.closeCh:
		return net.ErrClosed
	}

	return nil
}

func (bc *BufferedConn) onTimeout() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.flushLocked()
}

func (bc *BufferedConn) writer() {
	for {
		select {
		case data := <-bc.writeCh:
			if _, err := bc.Conn.Write(data); err != nil {
				return
			}
		case <-bc.closeCh:
			bc.mu.Lock()
			if len(bc.buffer) > 0 {
				bc.flushLocked()
			}
			bc.mu.Unlock()

			return
		}
	}
}

func (bc *BufferedConn) Close() error {
	close(bc.closeCh)

	time.Sleep(bc.maxDelay + 10*time.Millisecond)

	return bc.Conn.Close()
}

func (bc *BufferedConn) Flush() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	return bc.flushLocked()
}
