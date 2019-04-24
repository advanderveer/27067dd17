package engine_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

func TestEngineSetupAndShutdown(t *testing.T) {
	s1, clean := onl.TempBadgerStore()
	defer clean()
	c1, _, _ := onl.NewChain(s1, func(kv *onl.KV) {

	})

	osc1 := engine.NewMemOscillator()
	bc1 := engine.NewMemBroadcast(100)
	idn1 := onl.NewIdentity([]byte{0x01})
	e1 := engine.New(os.Stderr, bc1, osc1.Pulse(), idn1, c1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	test.Ok(t, e1.Shutdown(ctx))
}

func TestEngineRoundTaking(t *testing.T) {
	// bc := engine.NewMemBroadcast(100)
	// e1 := engine.New(ioutil.Discard)

}
