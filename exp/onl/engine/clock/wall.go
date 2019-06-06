package clock

import (
	"io"
	"sync"
	"time"
)

//WallClock provides synced rounds using just a local clock with a fixed round time.
//It assumes the local clock is reasonably synced with all other clocks in the network
//using something like NTP
type WallClock struct {
	c      chan *round
	curr   int64
	mu     sync.RWMutex
	ticker *time.Ticker
	done   chan struct{}
}

type round struct {
	nr uint64
	ts uint64
}

// NewWallClock creates a wall clock
func NewWallClock(trunc time.Duration) (c *WallClock) {
	c = &WallClock{
		c:      make(chan *round, 1),
		ticker: time.NewTicker(time.Millisecond * 10),
		done:   make(chan struct{}),
	}

	//observe current absolute round
	_, c.curr = c.observe(trunc)

	go func() {
		defer close(c.done)

		for {
			select {
			case <-c.done:
				return //stop rounds
			case <-c.ticker.C:

				//observe new absolute round
				ts, r := c.observe(trunc)

				//if new round larger then previous, send out new round
				c.mu.Lock()
				if r > c.curr {
					c.curr = r
					c.c <- &round{
						nr: uint64(c.curr),
						ts: uint64(ts), //milliseconds since epoch
					}
				}

				c.mu.Unlock()
			}
		}
	}()

	return
}

// Round returns the current round nr as observed by the last clock observation
func (c *WallClock) Round() (nr uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return uint64(c.curr)
}

//observe millisecond timestamp and absolute round
func (c *WallClock) observe(trunc time.Duration) (ts, round int64) {
	t := time.Now()
	round = t.Truncate(trunc).UnixNano() / trunc.Nanoseconds()
	return t.UnixNano() / 1e6, round
}

// Next returns the next round and an observed timestamp
func (c *WallClock) Next() (round, ts uint64, err error) {
	rmsg := <-c.c
	if rmsg == nil {
		return 0, 0, io.EOF
	}

	return rmsg.nr, rmsg.ts, nil
}

// Close the wall clock
func (c *WallClock) Close() (err error) {
	c.done <- struct{}{}
	<-c.done
	c.ticker.Stop()
	close(c.c)
	return
}
