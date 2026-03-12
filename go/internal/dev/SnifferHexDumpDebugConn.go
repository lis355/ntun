package dev

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

var mu sync.Mutex

type SnifferHexDumpDebugConn struct {
	net.Conn
	Prefix    string
	PrintDump bool
}

func (s *SnifferHexDumpDebugConn) Read(b []byte) (n int, err error) {
	n, err = s.Conn.Read(b)
	if n > 0 {
		fmt.Printf("READ %s %d\n", s.Prefix, n)
		if s.PrintDump {
			printHexDump(b[:n])
		}
	}
	return n, err
}

func (s *SnifferHexDumpDebugConn) Write(b []byte) (n int, err error) {
	n, err = s.Conn.Write(b)
	if n > 0 {
		fmt.Printf("WRITE %s %d\n", s.Prefix, n)
		if s.PrintDump {
			printHexDump(b[:n])
		}
	}
	return n, err
}

func printHexDump(data []byte) {
	mu.Lock()
	defer mu.Unlock()

	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("%02X ", i)

	// 	if (i+1)%8 == 0 {
	// 		fmt.Print(" ")
	// 	}
	// }
	// fmt.Print("\n")

	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("--")

	// 	if i < 31 {
	// 		fmt.Print("-")
	// 	}

	// 	if (i+1)%8 == 0 &&
	// 		i+1 != 32 {
	// 		fmt.Print("-")
	// 	}
	// }
	// fmt.Print("\n")

	for i := 0; i < len(data); i += 32 {
		fmt.Printf("%08X| ", i)

		ascii := strings.Builder{}
		ascii.WriteString("|")

		for j := 0; j < 32; j++ {
			if i+j < len(data) {
				fmt.Printf("%02X ", data[i+j])

				b := data[i+j]
				if b >= 32 &&
					b <= 126 {
					ascii.WriteByte(b)
				} else {
					ascii.WriteByte('.')
				}
			} else {
				fmt.Print("   ")
				ascii.WriteByte(' ')
			}

			if (j+1)%8 == 0 &&
				j+1 != 32 {
				fmt.Print(" ")
			}
		}

		fmt.Print(ascii.String())
		fmt.Print("\n")
	}
}
