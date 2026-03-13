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

	// closeConns := sync.OnceFunc(func() {
	// 	// handle tls connections
	// 	// b := make([]byte, 0)
	// 	// srcConn.Read(b)
	// 	// dstConn.Read(b)

	// 	srcConn.Close()
	// 	dstConn.Close()
	// })

	// proxyConns := func(a, b net.Conn) error {
	// 	_, err := io.Copy(a, b)
	// 	if errors.Is(err, io.EOF) ||
	// 		errors.Is(err, net.ErrClosed) {
	// 		err = nil
	// 	}

	// 	closeConns()

	// 	return err
	// }

	// var g errgroup.Group

	// g.Go(func() error {
	// 	return proxyConns(srcConn, dstConn)
	// })

	// g.Go(func() error {
	// 	return proxyConns(dstConn, srcConn)
	// })

	// err := g.Wait()

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(srcConn, dstConn)
		// slog.Debug(fmt.Sprintf("Done proxying 1 srcConn, dstConn %v", err))

		srcConn.Close()

		return err
	})

	g.Go(func() error {
		_, err := io.Copy(dstConn, srcConn)
		// slog.Debug(fmt.Sprintf("Done proxying 2 dstConn, srcConn %v", err))

		dstConn.Close()

		return err
	})

	err := g.Wait()

	slog.Debug(fmt.Sprintf("Done proxying %s <--> %s %v", srcConn.RemoteAddr(), dstConn.RemoteAddr(), err))

	return nil
}
