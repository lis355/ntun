package dev

import (
	"fmt"
	"hash/crc32"
	"log/slog"
	"net"
	"ntun/internal/utils"
	"sync"
)

var mu sync.Mutex

type SnifferHexDumpDebugConn struct {
	net.Conn
	prefix                        string
	sniffRead, sniffWrite         bool
	printReadDump, printWriteDump bool
}

func NewSnifferHexDumpDebugConn(conn net.Conn, prefix string, sniffRead, sniffWrite, printReadDump, printWriteDump bool) *SnifferHexDumpDebugConn {
	return &SnifferHexDumpDebugConn{
		Conn:           conn,
		prefix:         prefix,
		sniffRead:      sniffRead,
		sniffWrite:     sniffWrite,
		printReadDump:  printReadDump,
		printWriteDump: printWriteDump,
	}
}

func (s *SnifferHexDumpDebugConn) Read(b []byte) (n int, err error) {
	n, err = s.Conn.Read(b)
	if s.sniffRead &&
		n > 0 {
		mu.Lock()
		slog.Debug(fmt.Sprintf("%s r %d bytes [%d]", s.prefix, n, crc32.ChecksumIEEE(b[:n])))
		// slog.Debug(fmt.Sprintf("%s r %d bytes %s", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n])))
		if s.printReadDump {
			slog.Debug("\n" + utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}

func (s *SnifferHexDumpDebugConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if s.sniffWrite &&
		n > 0 {
		mu.Lock()
		slog.Debug(fmt.Sprintf("%s w %d bytes [%d]", s.prefix, n, crc32.ChecksumIEEE(b[:n])))
		// slog.Debug(fmt.Sprintf("%s w %d bytes %s", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n])))
		if s.printWriteDump {
			slog.Debug("\n" + utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}
