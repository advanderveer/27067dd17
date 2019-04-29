package clock_test

import (
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/clock"
	"github.com/advanderveer/go-test"
)

var _ engine.Clock = &clock.WallClock{}

func TestBasicWallClock(t *testing.T) {

	c1 := clock.NewWallClock(time.Second * 1)
	test.Assert(t, c1.Round() > 10000, "should set initial absolute round")

	nr1, ts1, err := c1.Next()
	fmt.Println(ts1)
	test.Assert(t, ts1 > 1556525961000, "should get timestamp in truncated milliseconds since epoch")
	test.Ok(t, err)

	t0 := time.Now()
	nr2, _, _ := c1.Next()
	test.Equals(t, uint64(1), nr2-nr1)
	test.Equals(t, nr2, c1.Round())

	//should round to about 1 second
	test.Equals(t, 1.0, math.Round(time.Now().Sub(t0).Seconds()))

	test.Ok(t, c1.Close())

	//next after close should return eof
	nr3, _, err := c1.Next()
	test.Equals(t, uint64(0), nr3)
	test.Equals(t, io.EOF, err)

}
