package engine_test

import (
	"io"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

func TestBroadcast(t *testing.T) {
	bc1 := engine.NewMemBroadcast(1)
	bc2 := engine.NewMemBroadcast(1)
	bc1.To(bc2)

	msg1 := &engine.Msg{}
	test.Ok(t, bc1.Write(msg1))

	msg2 := &engine.Msg{}
	test.Ok(t, bc2.Read(msg2))

	test.Equals(t, msg1, msg2)

	t.Run("close should return EOF", func(t *testing.T) {
		test.Ok(t, bc2.Close())
		test.Equals(t, io.EOF, bc2.Read(msg2))

		test.Ok(t, bc1.Write(msg1)) //should still work and not panic
	})
}

func TestInjectorVoting(t *testing.T) {
	bc1 := engine.NewMemBroadcast(1)
	inj1 := engine.NewInjector([]byte{0x01}, 2)
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
	inj1 := engine.NewInjector([]byte{0x01}, 0)
	inj2 := engine.NewInjector([]byte{0x02}, 0)
	inj2.To(inj1.MemBroadcast)

	test.Ok(t, inj2.Write(&engine.Msg{}))
	time.Sleep(time.Millisecond)

	msgs := inj1.Collect()
	test.Equals(t, 1, len(msgs))
}
