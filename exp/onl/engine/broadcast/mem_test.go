package broadcast_test

import (
	"io"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/go-test"
)

var _ engine.Broadcast = &broadcast.Mem{}

var (
	//some test ids
	bid1 = onl.ID{}
	bid2 = onl.ID{}
)

func init() {
	bid1[0] = 0x01
	bid2[0] = 0x02
}

func TestBroadcast(t *testing.T) {
	bc1 := broadcast.NewMem(1)
	bc2 := broadcast.NewMem(1)
	bc1.To(bc2)

	msg1 := &engine.Msg{}
	test.Ok(t, bc1.Write(msg1))

	msg2 := &engine.Msg{}
	test.Ok(t, bc2.Read(msg2))

	test.Equals(t, msg1, msg2)

	t.Run("with latency", func(t *testing.T) {
		bc1.WithLatency(time.Millisecond*50, time.Millisecond*100)

		msg3 := &engine.Msg{Block: &onl.Block{Round: 1}}
		test.Ok(t, bc1.Write(msg3))

		msg4 := &engine.Msg{}
		t0 := time.Now()
		test.Ok(t, bc2.Read(msg4))

		test.Assert(t, time.Now().Sub(t0) >= time.Millisecond*50, "latency should be at least 50ms")
	})

	t.Run("infinite latency, out-of-order ", func(t *testing.T) {
		bc1.WithLatency(time.Hour, time.Hour*1000) //really long time

		msg5 := &engine.Msg{Block: &onl.Block{Round: 2}}
		test.Ok(t, bc1.Write(msg5))

		bc1.WithLatency(0, 0)

		msg6 := &engine.Msg{Block: &onl.Block{Round: 3}}
		test.Ok(t, bc1.Write(msg6))

		msg7 := &engine.Msg{}
		test.Ok(t, bc2.Read(msg7))

		//msg6 should arrive before msg5
		test.Equals(t, msg7, msg6)
	})

	t.Run("close should return EOF", func(t *testing.T) {
		test.Ok(t, bc2.Close())
		test.Equals(t, io.EOF, bc2.Read(msg2))

		test.Ok(t, bc1.Write(msg1)) //should still work and not panic
	})
}

func TestSyncMessage(t *testing.T) {
	bc1 := broadcast.NewMem(1)
	bc2 := broadcast.NewMem(1)
	bc1.To(bc2)

	//bc1 writes sync request to b2
	msg1 := &engine.Msg{Sync: &engine.Sync{IDs: []onl.ID{bid1}}}
	test.Ok(t, bc1.Write(msg1))

	//bc2 reads sync request
	msg2 := &engine.Msg{}
	test.Ok(t, bc2.Read(msg2))

	//bc2 pushes block to sync back
	b1 := &onl.Block{Round: 1}
	test.Ok(t, msg2.Sync.Push(b1))

	//bc1 should read a block message
	msg3 := &engine.Msg{}
	test.Ok(t, bc1.Read(msg3))
	test.Equals(t, uint64(1), msg3.Block.Round)

	//after close writing a sync block should return closed error
	test.Ok(t, bc1.Close())
	b2 := &onl.Block{Round: 1}
	test.Equals(t, broadcast.ErrClosed, msg2.Sync.Push(b2))
}
