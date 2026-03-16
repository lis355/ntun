package main

import (
	"fmt"
	"ntun/internal/cfg"
	"ntun/internal/connections"
	"ntun/internal/utils"
	"sync"
	"time"
)

func main() {
	aConn, bConn := utils.SocketPipe()

	aTrafficStatsConn, bTrafficStatsConn := connections.NewTrafficStatsConn(aConn), connections.NewTrafficStatsConn(bConn)
	aConn, bConn = aTrafficStatsConn, bTrafficStatsConn

	aConn = connections.NewRateLimitedConn(aConn, &cfg.Rate{Value: 1024 * 100, Interval: time.Second})
	bConn = connections.NewRateLimitedConn(bConn, &cfg.Rate{Value: 1024 * 100, Interval: time.Second})

	start := time.Now()

	go func() {
		for {
			fmt.Print("\033[1A\033[2K")
			fmt.Printf("T %04.2f U %6.2f kB | D %6.2f kB | ↑ %5.2f kB/s | ↓ %5.2f kB/s\n",
				time.Since(start).Seconds(),
				float64(aTrafficStatsConn.Written())/1024,
				float64(bTrafficStatsConn.Readed())/1024,
				aTrafficStatsConn.WriteSpeed()/1024,
				bTrafficStatsConn.ReadSpeed()/1024,
			)

			time.Sleep(100 * time.Millisecond)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		readBuf := make([]byte, 1024)
		for {
			_, err := bConn.Read(readBuf)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		payload := func(do bool) {
			for range 50 {
				if do {
					buf := utils.RandBytes(1024 * 200)
					_, err := aConn.Write(buf)
					if err != nil {
						return
					}
				}

				time.Sleep(100 * time.Millisecond)
			}
		}

		payload(true)
		payload(false)
		payload(true)

		aConn.Close()
	}()

	wg.Wait()
}
