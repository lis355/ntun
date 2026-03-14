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
	Prefix    string
	PrintDump bool
}

func NewSnifferHexDumpDebugConn(conn net.Conn, prefix string, printDump bool) *SnifferHexDumpDebugConn {
	return &SnifferHexDumpDebugConn{
		Conn:      conn,
		Prefix:    prefix,
		PrintDump: printDump,
	}
}

func (s *SnifferHexDumpDebugConn) Read(b []byte) (n int, err error) {
	n, err = s.Conn.Read(b)
	if n > 0 {
		mu.Lock()
		slog.Debug(fmt.Sprintf("%s r %d bytes [%d]", s.Prefix, n, crc32.ChecksumIEEE(b[:n])))
		// slog.Debug(fmt.Sprintf("%s r %d bytes %s", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n])))
		if s.PrintDump {
			slog.Debug("\n" + utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}

func (s *SnifferHexDumpDebugConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if n > 0 {
		mu.Lock()
		slog.Debug(fmt.Sprintf("%s w %d bytes [%d]", s.Prefix, n, crc32.ChecksumIEEE(b[:n])))
		// slog.Debug(fmt.Sprintf("%s w %d bytes %s", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n])))
		if s.PrintDump {
			slog.Debug("\n" + utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}
