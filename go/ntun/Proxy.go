package ntun

import (
	"fmt"
	"io"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"
)

func Proxy(srcConn, dstConn net.Conn) error {
	slog.Debug(fmt.Sprintf("Proxying %s <--> %s", srcConn.RemoteAddr(), dstConn.RemoteAddr()))

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(srcConn, dstConn)

		tcpConn, ok := srcConn.(*net.TCPConn)
		if ok {
			tcpConn.CloseWrite()
			srcConn.Read(make([]byte, 1))
		}

		srcConn.Close()

		// slog.Debug(fmt.Sprintf("Done proxying 1 srcConn, dstConn %v", err))

		return err
	})

	g.Go(func() error {
		_, err := io.Copy(dstConn, srcConn)

		tcpConn, ok := dstConn.(*net.TCPConn)
		if ok {
			tcpConn.CloseWrite()
			dstConn.Read(make([]byte, 1))
		}

		dstConn.Close()

		// slog.Debug(fmt.Sprintf("Done proxying 2 dstConn, srcConn %v", err))

		return err
	})

	err := g.Wait()

	srcConn.Close()
	dstConn.Close()

	slog.Debug(fmt.Sprintf("Done proxying %s <--> %s %v", srcConn.RemoteAddr(), dstConn.RemoteAddr(), err))

	return nil
}
