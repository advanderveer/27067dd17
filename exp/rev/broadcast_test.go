package rev_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/go-test"
)

func TestBroadcast(t *testing.T) {
	bc1 := rev.NewMemBroadcast()
	bc2 := rev.NewMemBroadcast()
	bc1.To(bc2)

	msg1 := &rev.Msg{}
	test.Ok(t, bc1.Write(msg1))

	msg2 := &rev.Msg{}
	test.Ok(t, bc2.Read(msg2))

	test.Equals(t, msg1, msg2)

	t.Run("close should return EOF", func(t *testing.T) {
		test.Ok(t, bc2.Close())
		test.Equals(t, io.EOF, bc2.Read(msg2))

		test.Ok(t, bc1.Write(msg1)) //should still work and not panic
	})
}

func TestInjector(t *testing.T) {
	bc1 := rev.NewMemBroadcast()
	bc2 := rev.NewInjector([]byte{0x01})
	bc2.To(bc1)

	p1 := bc2.Propose(1, nil)

	msg := &rev.Msg{}
	test.Ok(t, bc1.Read(msg))
	test.Equals(t, p1, msg.Proposal)

	p2 := bc2.Propose(1, nil, p1)
	test.Ok(t, bc1.Read(msg))
	test.Equals(t, p2, msg.Proposal)
}
