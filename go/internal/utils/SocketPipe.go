package utils

import (
	"net"
)

func SocketPipe() (net.Conn, net.Conn) {
	listener, _ := net.Listen("tcp", "localhost:0")

	serverConnChan := make(chan net.Conn)

	go func() {
		serverConn, _ := listener.Accept()
		serverConnChan <- serverConn
	}()

	clientConn, _ := net.Dial("tcp", listener.Addr().String())
	serverConn := <-serverConnChan

	return clientConn, serverConn
}
