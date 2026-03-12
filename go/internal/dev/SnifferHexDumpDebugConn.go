package dev

import (
	"fmt"
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
		fmt.Printf("%s read %d bytes\n", s.Prefix, n)
		if s.PrintDump {
			utils.PrintHexDump(b[:n])
		}
		mu.Unlock()
	}
	return n, err
}

func (s *SnifferHexDumpDebugConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if n > 0 {
		mu.Lock()
		fmt.Printf("%s write %d bytes\n", s.Prefix, n)
		if s.PrintDump {
			utils.PrintHexDump(b[:n])
		}
		mu.Unlock()
	}
	return n, err
}
