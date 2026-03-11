package ntun

import (
	"errors"
	"io"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

func Proxy(srcConn, dstConn net.Conn) error {
	// slog.Debug(fmt.Sprintf("Proxying %s<-->%s", srcConn.RemoteAddr(), dstConn.RemoteAddr()))

	closeConns := sync.OnceFunc(func() {
		srcConn.Close()
		dstConn.Close()
	})

	proxyConns := func(a, b net.Conn) error {
		_, err := io.Copy(a, b)
		if errors.Is(err, io.EOF) ||
			errors.Is(err, net.ErrClosed) {
			err = nil
		}

		closeConns()

		return err
	}

	var g errgroup.Group

	g.Go(func() error {
		return proxyConns(srcConn, dstConn)
	})

	g.Go(func() error {
		return proxyConns(dstConn, srcConn)
	})

	err := g.Wait()

	// slog.Debug(fmt.Sprintf("Done proxying %s<-->%s", srcConn.RemoteAddr(), dstConn.RemoteAddr()))

	return err
}
