package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/internal/log"

	"golang.org/x/sync/errgroup"
)

type proxy struct {
	srcConn, dstConn net.Conn
}

func (p *proxy) run() {
	slog.Debug(fmt.Sprintf("%s: proxying %s <--> %s", log.ObjName(p), p.srcConn.RemoteAddr(), p.dstConn.RemoteAddr()))

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(p.srcConn, p.dstConn)

		tcpConn, ok := p.srcConn.(*net.TCPConn)
		if ok {
			tcpConn.CloseWrite()
			p.srcConn.Read(make([]byte, 1))
		}

		p.srcConn.Close()

		// slog.Debug(fmt.Sprintf("Done proxying 1  p.srcConn,  p.dstConn %v", err))

		return err
	})

	g.Go(func() error {
		_, err := io.Copy(p.dstConn, p.srcConn)

		tcpConn, ok := p.dstConn.(*net.TCPConn)
		if ok {
			tcpConn.CloseWrite()
			p.dstConn.Read(make([]byte, 1))
		}

		p.dstConn.Close()

		// slog.Debug(fmt.Sprintf("Done proxying 2  p.dstConn,  p.srcConn %v", err))

		return err
	})

	err := g.Wait()

	p.srcConn.Close()
	p.dstConn.Close()

	slog.Debug(fmt.Sprintf("%s: done proxying %s <--> %s err=%v", log.ObjName(p), p.srcConn.RemoteAddr(), p.dstConn.RemoteAddr(), err))
}

func Proxy(srcConn, dstConn net.Conn) {
	proxy := &proxy{srcConn, dstConn}

	proxy.run()
}
