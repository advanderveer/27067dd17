package broadcast_test

import (
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/go-test"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

var _ engine.Broadcast = &broadcast.Injector{}

func TestInjectorVoting(t *testing.T) {
	bc1 := broadcast.NewMem(1)
	inj1 := broadcast.NewInjector([]byte{0x01}, 2)
	inj1.WithLatency(time.Millisecond*100, time.Millisecond*101)
	inj1.To(bc1)

	id1 := onl.ID{}
	id1[0] = 0x01

	t.Run("injector minting", func(t *testing.T) {
		t0 := time.Now()
		b1 := inj1.Mint(testClock(1), id1, id1, 1)

		msg := &engine.Msg{}
		test.Ok(t, bc1.Read(msg))

		dur := time.Now().Sub(t0)
		test.Assert(t, dur > time.Millisecond*100, "should have taken at least min latency")

		test.Equals(t, b1, msg.Block)
	})
}

func TestInjectorCollection(t *testing.T) {
	inj1 := broadcast.NewInjector([]byte{0x01}, 0)
	inj2 := broadcast.NewInjector([]byte{0x02}, 0)
	inj2.To(inj1.Mem)

	test.Ok(t, inj2.Write(&engine.Msg{}))
	time.Sleep(time.Millisecond)

	msgs := inj1.Collect()
	test.Equals(t, 1, len(msgs))
}
