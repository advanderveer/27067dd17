package slot_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

func TestBroadcast2(t *testing.T) {
	bc1 := slot.NewMemBroadcast2()
	bc2 := slot.NewMemBroadcast2()
	bc3 := slot.NewMemBroadcast2()
	coll, done := slot.Collect2(1)
	bc1.Relay(bc2, bc3, coll) //bc1 will write to bc2 and bc3

	inj1 := newInjector(0x01)

	msg1 := inj1.propose(slot.NilID, 1, []byte{0x01})
	test.Ok(t, bc1.Write(1, msg1))

	msg2 := &slot.Msg2{}
	test.Ok(t, bc2.Read(1, msg2))
	msg3 := &slot.Msg2{}
	test.Ok(t, bc3.Read(1, msg3))

	test.Equals(t, []byte{0x01}, msg2.Proposal.Block.Data)
	test.Equals(t, []byte{0x01}, msg3.Proposal.Block.Data)

	//test collect
	msgs := <-done()
	test.Equals(t, []*slot.Msg2{msg1}, msgs)

	//close 1 endpoint
	test.Ok(t, bc3.Close())

	//reading should now return EOF
	err := bc3.Read(1, msg3)
	test.Equals(t, io.EOF, err)

	//Writing should still be possible
	msg4 := inj1.propose(slot.NilID, 1, []byte{0x01})
	test.Ok(t, bc1.Write(1, msg4)) //should not panic
}
