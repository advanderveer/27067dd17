package onl

import "time"

//Clock represents a time keeping device
type Clock interface {
	ReadUs() (ts uint64)
}

//WallClock implements time keeping by looking at the local wallclock
type WallClock struct{}

//NewWallClock creates a clock
func NewWallClock() *WallClock {
	return &WallClock{}
}

//ReadUs reads the microseconds since the unix epoch, overflows in the year 584940447
func (c *WallClock) ReadUs() uint64 {
	return uint64(time.Now().UnixNano() / (10 ^ 6))
}
