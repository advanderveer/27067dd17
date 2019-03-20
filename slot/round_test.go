package slot_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

type testRound struct {
	pk []byte
}

func newTestRound(pk byte) (tr *testRound) {
	tr = &testRound{pk: make([]byte, slot.PKSize)}

	return
}

func (t *testRound) propose(prev slot.ID, rank byte, d []byte) *slot.Msg2 {
	token := make([]byte, slot.TicketSize)
	token[0] = rank

	return &slot.Msg2{
		Proposal: &slot.Proposal2{
			Proof: make([]byte, slot.ProofSize),
			PK:    t.pk,
			Token: token,
			Block: &slot.Block2{
				Prev: prev,
				Data: d,
			},
		},
	}
}

func (t *testRound) vote(p *slot.Proposal2) *slot.Msg2 {
	return &slot.Msg2{
		Vote: &slot.Vote2{
			Proof: make([]byte, slot.ProofSize),
			Token: make([]byte, slot.TicketSize),
			PK:    t.pk,

			Proposal: p,
		},
	}
}

func TestBlockHash(t *testing.T) {
	b1 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x01}}
	test.Equals(t, "1fd4247443c9", fmt.Sprintf("%.6x", b1.Hash()))

	b2 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x02}}
	test.Equals(t, "58cc2f44d3a2", fmt.Sprintf("%.6x", b2.Hash()))

	b3 := &slot.Block2{Prev: b2.Hash(), Data: []byte{0x02}}
	test.Equals(t, "ae1b2a0343c2", fmt.Sprintf("%.6x", b3.Hash()))

}

func TestMsgByMsgRoundComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	// protocol params
	bt := time.Millisecond * 10

	// broadcast: ep2 -> ep1 -> coll1
	coll1, done1 := slot.Collect2(1)
	ep1 := slot.NewMemBroadcast2()
	ep1.Relay(coll1)
	ep2 := slot.NewMemBroadcast2()
	ep2.Relay(ep1)

	// rounds
	tr1 := newTestRound(0x01)
	gen := slot.GenesisBlock()
	r1 := slot.NewRound(ep1, 1, gen, bt)

	// --- proposals ---

	//@TODO write a to round 2 message should not show up until round 2 starts
	msg1 := tr1.propose(gen.Hash(), 0x01, []byte{0x01})
	test.Ok(t, ep2.Write(1, msg1))

	time.Sleep(bt * 2)
	//@TODO send lower proposal again, should not broadcast as vote because
	//we have not reset top score block state for the voter.
	//@TODO wait for bt and write a higher ranking proposal, should broadcast
	//as vote also and immediately.

	// --- votes ---

	test.Ok(t, ep2.Write(1, tr1.vote(msg1.Proposal)))

	r2, err := r1.Run(ctx)
	test.Ok(t, err)

	msgs := <-done1()

	//@TODO should have made proposal to collector
	//@TODO should have relayed first proposal (high rank) to collector
	//@TODO should not have relayed second proposal (rank too low) to collector
	//@TODO should have cast vote for round 1 to collector

	fmt.Println(msgs)
	_ = r2

}

// @TODO test in ring network
