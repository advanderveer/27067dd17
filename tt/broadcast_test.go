package tt_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestBroadcast(t *testing.T) {
	bc1 := tt.NewMemBroadcast(1)
	bc2 := tt.NewMemBroadcast(1)
	bc1.To(bc2)

	msg1 := &tt.Msg{}
	test.Ok(t, bc1.Write(msg1))

	msg2 := &tt.Msg{}
	test.Ok(t, bc2.Read(msg2))

	test.Equals(t, msg1, msg2)

	t.Run("close should return EOF", func(t *testing.T) {
		test.Ok(t, bc2.Close())
		test.Equals(t, io.EOF, bc2.Read(msg2))

		test.Ok(t, bc1.Write(msg1)) //should still work and not panic
	})
}

func TestInjectorVoting(t *testing.T) {
	bc1 := tt.NewMemBroadcast(1)
	inj1 := tt.NewInjector([]byte{0x01}, 2)
	inj1.WithLatency(time.Millisecond*100, time.Millisecond*101)
	inj1.To(bc1)

	id1 := tt.ID{}
	id1[0] = 0x01
	t0 := time.Now()
	v1 := inj1.Vote(id1)

	msg := &tt.Msg{}
	test.Ok(t, bc1.Read(msg))

	dur := time.Now().Sub(t0)
	test.Assert(t, dur > time.Millisecond*100, "should have taken at least min latency")

	test.Equals(t, v1, msg.Vote)
	test.Equals(t, "2eba0247", fmt.Sprintf("%.4x", msg.Vote.Token))
	test.Equals(t, "6a7a21d9", fmt.Sprintf("%.4x", msg.Vote.Proof))
	test.Equals(t, "4762ad64", fmt.Sprintf("%.4x", msg.Vote.PK))
}

func TestInjectorCollection(t *testing.T) {
	inj1 := tt.NewInjector([]byte{0x01}, 0)
	inj2 := tt.NewInjector([]byte{0x02}, 0)
	inj2.To(inj1.MemBroadcast)

	test.Ok(t, inj2.Write(&tt.Msg{}))
	time.Sleep(time.Millisecond)

	msgs := inj1.Collect()
	test.Equals(t, 1, len(msgs))

}
