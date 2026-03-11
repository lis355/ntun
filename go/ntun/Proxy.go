package ntun

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

func Proxy(srcConn, destConn net.Conn) error {
	slog.Debug(fmt.Sprintf("Proxying %s<-->%s", srcConn.RemoteAddr(), destConn.RemoteAddr()))

	closeConns := sync.OnceFunc(func() {
		srcConn.Close()
		destConn.Close()
	})

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(srcConn, destConn)
		if err != nil {
			closeConns()
		}
		return err
	})

	g.Go(func() error {
		_, err := io.Copy(destConn, srcConn)
		if err != nil {
			closeConns()
		}
		return err
	})

	err := g.Wait()

	closeConns()

	slog.Debug(fmt.Sprintf("Done proxying %s<-->%s", srcConn.RemoteAddr(), destConn.RemoteAddr()))

	return err
}
