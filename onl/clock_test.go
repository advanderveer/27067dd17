package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestWallClock(t *testing.T) {
	c1 := onl.NewWallClock()
	test.Assert(t, c1.ReadUs() > 129609265633051333, "should give a time after some time in the past")
}
