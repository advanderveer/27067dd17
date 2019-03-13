package slot_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

type testbw struct{ msgs []*slot.Msg }

func (bw *testbw) Write(m *slot.Msg) (err error) {
	bw.msgs = append(bw.msgs, m)
	return nil
}

type testbr map[slot.ID]*slot.Block

func (br testbr) Read(id slot.ID) (b *slot.Block) {
	b = br[id]
	return
}

func TestOutOfOrderNoPrev(t *testing.T) {
	b1 := slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)

	var handles []slot.ID
	o := slot.NewOutOfOrder(&testbw{}, make(testbr), func(msg *slot.Msg, bw slot.BroadcastWriter) error {
		handles = append(handles, msg.Vote.Hash())
		return nil
	})

	msg1 := &slot.Msg{Vote: b1}
	test.Ok(t, o.Handle(msg1))
	test.Equals(t, []slot.ID{b1.Hash()}, handles)
}

func TestOutOfOrderSimple(t *testing.T) {
	br := make(testbr)
	b1 := slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
	b2 := slot.NewBlock(2, b1.Hash(), slot.NilTicket, slot.NilProof, slot.NilPK)
	b3 := slot.NewBlock(3, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
	br[b3.Hash()] = b3
	b4 := slot.NewBlock(4, b3.Hash(), slot.NilTicket, slot.NilProof, slot.NilPK)

	b5 := slot.NewBlock(5, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
	b6 := slot.NewBlock(6, b5.Hash(), slot.NilTicket, slot.NilProof, slot.NilPK)

	var handles []slot.ID
	o := slot.NewOutOfOrder(&testbw{}, br, func(msg *slot.Msg, bw slot.BroadcastWriter) error {
		if msg.Vote.Hash() == b6.Hash() {
			return fmt.Errorf("foo")
		}

		handles = append(handles, msg.Vote.Hash())
		return nil
	})

	msg2 := &slot.Msg{Vote: b2}
	test.Ok(t, o.Handle(msg2)) //msg2 arrives before msg1

	msg1 := &slot.Msg{Vote: b1}
	test.Ok(t, o.Handle(msg1)) //then msg1 arrives, goes trough right away
	test.Equals(t, []slot.ID{b1.Hash()}, handles)

	test.Ok(t, o.Resolve(b1))
	test.Equals(t, []slot.ID{b1.Hash(), b2.Hash()}, handles)

	//resolving again should not cause another handle
	test.Ok(t, o.Resolve(b1))
	test.Equals(t, []slot.ID{b1.Hash(), b2.Hash()}, handles)

	//b4 should succeed because b3 was already written before any handling
	test.Ok(t, o.Handle(&slot.Msg{Vote: b4})) //then msg1 arrives, goes trough right away
	test.Equals(t, []slot.ID{b1.Hash(), b2.Hash(), b4.Hash()}, handles)

	t.Run("error resolve", func(t *testing.T) {
		test.Ok(t, o.Handle(&slot.Msg{Vote: b6}))
		err := o.Resolve(b5)
		test.Equals(t, "failed to resolve out-of-order messages: foo", err.Error())
	})
}
