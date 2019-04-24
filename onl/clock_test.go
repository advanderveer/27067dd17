package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

func TestWallClock(t *testing.T) {
	c1 := onl.NewWallClock()
	test.Assert(t, c1.ReadUs() > 129609265633051333, "should give a time after some time in the past")
}
