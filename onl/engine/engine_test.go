package engine_test

import (
	"testing"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

func TestEngineRoundTaking(t *testing.T) {
	// bc := engine.NewMemBroadcast(100)
	// e1 := engine.New(ioutil.Discard)

}
