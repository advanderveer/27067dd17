package slot_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

func TestMemNetwork(t *testing.T) {
	netw := slot.NewMemNetwork()
	b1 := slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
	b2 := slot.NewBlock(2, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)

	msg1 := &slot.Msg{Proposal: b1}
	msg2 := &slot.Msg{Proposal: b2}

	err := netw.Write(msg1) //no endpoints, message is discarded
	test.Ok(t, err)

	ep1 := netw.Endpoint()
	err = netw.Write(msg2) //stored in the single element msg buffer
	test.Ok(t, err)

	msg3 := &slot.Msg{}
	err = ep1.Read(msg3)
	test.Ok(t, err)

	test.Equals(t, msg3.Proposal.Hash(), b2.Hash())
	test.Equals(t, uint64(2), msg3.Proposal.Round)
	test.Assert(t, msg3.Proposal != b2, "should be copied")

	err = netw.Write(msg2) //write again should be fine since last one was read
	test.Ok(t, err)

	test.Ok(t, ep1.Close())

	err = ep1.Read(msg3) //read should still succeed, one was buffered
	test.Ok(t, err)

	err = ep1.Read(msg3) //read should now return EOF
	test.Equals(t, io.EOF, err)

	err = netw.Write(msg2) //writes should still work, just not written to the closed ep
	test.Ok(t, err)

}
