package connections

import (
	"net"
	"ntun/internal/cfg"

	"github.com/conduitio/bwlimit"
)

func NewRateLimitedConn(conn net.Conn, rate *cfg.Rate) net.Conn {
	if rate.Value == 0 {
		return conn
	}

	rateBytesPerSecond := int(float64(rate.Value) / rate.Interval.Seconds())

	return bwlimit.NewConn(conn, bwlimit.Byte(rateBytesPerSecond), bwlimit.Byte(rateBytesPerSecond))
}

// TODO test own realisation

// const tick = 50 * time.Millisecond

// type Limiter struct {
// 	time     time.Time // время начала интервала
// 	value    int       // сколько уже в интервале с его начала
// 	maxValue int       // сколько максимум для tick
// }

// func (l *Limiter) WaitForNextTick() time.Duration {
// 	return max(0, tick-time.Since(l.time))
// }

// func (l *Limiter) Process(b []byte, action func([]byte) (int, error)) (int, error) {
// 	if time.Since(l.time) > tick {
// 		l.time = time.Now()
// 		l.value = 0
// 	}

// 	n := 0
// 	offset := 0
// 	for {
// 		ln := int(l.maxValue - l.value)
// 		rn := min(len(b), offset+ln) - offset
// 		if rn == 0 {
// 			return n, fmt.Errorf("can't process %d real %d", rn, n)
// 		}

// 		w, err := action(b[offset : offset+rn])
// 		n += w
// 		if err != nil {
// 			return n, err
// 		}
// 		if w != rn {
// 			return n, fmt.Errorf("can't process %d real %d", rn, w)
// 		}

// 		offset += w
// 		l.value += w

// 		if l.value < l.maxValue ||
// 			offset >= len(b) {
// 			break
// 		}

// 		d := tick - time.Since(l.time)
// 		// fmt.Printf("n %d total %d sleep %f", w, n, d.Seconds())
// 		time.Sleep(d)
// 		l.time = time.Now()
// 		l.value = 0
// 	}

// 	return n, nil
// }

// type RateLimitedConn struct {
// 	net.Conn
// 	readLimiter  Limiter
// 	writeLimiter Limiter
// }

// func NewRateLimitedConn(conn net.Conn, rate *cfg.Rate) *RateLimitedConn {
// 	maxValue := int(float64(rate.Value) / rate.Interval.Seconds() / float64(time.Second) * float64(tick))

// 	return &RateLimitedConn{
// 		Conn:         conn,
// 		readLimiter:  Limiter{maxValue: maxValue},
// 		writeLimiter: Limiter{maxValue: maxValue},
// 	}
// }

// func (r *RateLimitedConn) Read(b []byte) (int, error) {
// 	return r.readLimiter.Process(b, r.Conn.Read)
// }

// func (r *RateLimitedConn) Write(b []byte) (int, error) {
// 	return r.readLimiter.Process(b, r.Conn.Write)
// }
