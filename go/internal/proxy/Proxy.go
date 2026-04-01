package proxy

import (
	"io"
	"net"

	"golang.org/x/sync/errgroup"
)

func Proxy(srcConn, dstConn net.Conn) {
	// slog.Debug(fmt.Sprintf("%s: proxying %s <--> %s", log.ObjName(p), srcConn.RemoteAddr(), dstConn.RemoteAddr()))

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(srcConn, dstConn)

		tcpConn, ok := srcConn.(*net.TCPConn)
		if ok {
			tcpConn.CloseWrite()
			srcConn.Read(make([]byte, 1))
		}

		srcConn.Close()

		// slog.Debug(fmt.Sprintf("Done proxying 1  srcConn,  dstConn %v", err))

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

		// slog.Debug(fmt.Sprintf("Done proxying 2  dstConn,  srcConn %v", err))

		return err
	})

	err := g.Wait()

	srcConn.Close()
	dstConn.Close()

	_ = err

	// slog.Debug(fmt.Sprintf("%s: done proxying %s <--> %s err=%v", log.ObjName(p), srcConn.RemoteAddr(), dstConn.RemoteAddr(), err))
}
