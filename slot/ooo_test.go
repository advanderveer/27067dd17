package slot_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

var _ slot.MsgHandler = &slot.OutOfOrder{}

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
	b1 := &slot.Vote{Block: slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}

	var handles []slot.ID
	o := slot.NewOutOfOrder(make(testbr), slot.HandlerFunc(func(msg *slot.Msg) error {
		handles = append(handles, msg.Vote.BlockHash())
		return nil
	}))

	msg1 := &slot.Msg{Vote: b1}
	test.Ok(t, o.Handle(msg1))
	test.Equals(t, []slot.ID{b1.BlockHash()}, handles)
}

func TestOutOfOrderSimple(t *testing.T) {
	br := make(testbr)
	v1 := &slot.Vote{Block: slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}
	v2 := &slot.Vote{Block: slot.NewBlock(2, v1.BlockHash(), slot.NilTicket, slot.NilProof, slot.NilPK)}
	v3 := &slot.Vote{Block: slot.NewBlock(3, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}
	br[v3.BlockHash()] = v3.Block //stored through a side-channel
	v4 := &slot.Vote{Block: slot.NewBlock(4, v3.BlockHash(), slot.NilTicket, slot.NilProof, slot.NilPK)}
	v5 := &slot.Vote{Block: slot.NewBlock(5, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}
	v6 := &slot.Vote{Block: slot.NewBlock(6, v5.BlockHash(), slot.NilTicket, slot.NilProof, slot.NilPK)}

	var handles []slot.ID
	o := slot.NewOutOfOrder(br, slot.HandlerFunc(func(msg *slot.Msg) error {
		if msg.Vote.BlockHash() == v6.BlockHash() {
			return fmt.Errorf("foo")
		}

		handles = append(handles, msg.Vote.BlockHash())
		return nil
	}))

	msg2 := &slot.Msg{Vote: v2}
	test.Ok(t, o.Handle(msg2)) //msg2 arrives before msg1

	msg1 := &slot.Msg{Vote: v1}
	test.Ok(t, o.Handle(msg1)) //then msg1 arrives, goes trough right away
	test.Equals(t, []slot.ID{v1.BlockHash()}, handles)

	n, err := o.Resolve(v1.Block)
	test.Ok(t, err)
	test.Equals(t, 1, n)
	test.Equals(t, []slot.ID{v1.BlockHash(), v2.BlockHash()}, handles)

	//resolving again should not cause another handle
	n, err = o.Resolve(v1.Block)
	test.Ok(t, err)
	test.Equals(t, 0, n)
	test.Equals(t, []slot.ID{v1.BlockHash(), v2.BlockHash()}, handles)

	//v4 should succeed because v3 was already written before any handling
	test.Ok(t, o.Handle(&slot.Msg{Vote: v4})) //then msg1 arrives, goes trough right away
	test.Equals(t, []slot.ID{v1.BlockHash(), v2.BlockHash(), v4.BlockHash()}, handles)

	t.Run("error resolve", func(t *testing.T) {
		test.Ok(t, o.Handle(&slot.Msg{Vote: v6}))
		_, err = o.Resolve(v5.Block)
		test.Equals(t, "failed to resolve out-of-order messages: foo", err.Error())
	})
}
