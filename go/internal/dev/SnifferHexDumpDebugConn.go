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
		fmt.Printf("%s r %d bytes\n", s.Prefix, n)
		// fmt.Printf("%s r %d bytes %s\n", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n]))
		if s.PrintDump {
			fmt.Println(utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}

func (s *SnifferHexDumpDebugConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if n > 0 {
		mu.Lock()
		fmt.Printf("%s w %d bytes\n", s.Prefix, n)
		// fmt.Printf("%s w %d bytes %s\n", s.Prefix, n, utils.BytesToASCIIHexDumpString(b[:n]))
		if s.PrintDump {
			fmt.Println(utils.HexDump(b[:n]))
		}
		mu.Unlock()
	}
	return n, err
}
