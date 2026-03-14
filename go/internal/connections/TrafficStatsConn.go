package connections

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	intervalWindowDuration = 250 * time.Millisecond
)

type interval struct {
	time  time.Time
	value int
}

type speed struct {
	intervals []interval
	sum       int
	lock      sync.Mutex
}

func (s *speed) add(value int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now()
	s.intervals = append(s.intervals, interval{now, value})
	s.sum += value

	s.validate()
}

func (s *speed) validate() {
	now := time.Now()
	index := 0
	for index < len(s.intervals) &&
		now.Sub(s.intervals[index].time) > intervalWindowDuration {
		s.sum -= s.intervals[index].value
		index++
	}

	if index > 0 {
		s.intervals = (s.intervals)[index:]
	}
}

func (s *speed) speed() float64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.validate()

	return float64(s.sum) / intervalWindowDuration.Seconds()
}

type TrafficStatsConn struct {
	net.Conn
	readed, written       int32
	readSpeed, writeSpeed speed
}

func NewTrafficStatsConn(conn net.Conn) *TrafficStatsConn {
	return &TrafficStatsConn{
		Conn: conn,
	}
}

func (c *TrafficStatsConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	atomic.AddInt32(&c.readed, int32(n))
	c.readSpeed.add(n)
	return n, err
}

func (c *TrafficStatsConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	atomic.AddInt32(&c.written, int32(n))
	c.writeSpeed.add(n)
	return n, err
}

func (c *TrafficStatsConn) Readed() int {
	return int(c.readed)
}

func (c *TrafficStatsConn) Written() int {
	return int(c.written)
}

func (c *TrafficStatsConn) ResetStats() {
	atomic.StoreInt32(&c.readed, 0)
	atomic.StoreInt32(&c.written, 0)
}

func (c *TrafficStatsConn) WriteSpeed() float64 {
	return c.writeSpeed.speed()
}

func (c *TrafficStatsConn) ReadSpeed() float64 {
	return c.readSpeed.speed()
}
