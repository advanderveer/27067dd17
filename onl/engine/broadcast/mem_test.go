package broadcast_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/go-test"
)

var _ engine.Broadcast = &broadcast.Mem{}

func TestBroadcast(t *testing.T) {
	bc1 := broadcast.NewMem(1)
	bc2 := broadcast.NewMem(1)
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
