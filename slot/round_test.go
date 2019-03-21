package slot_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/go-test"
)

type injector struct {
	pk []byte
}

func newInjector(pk byte) (tr *injector) {
	tr = &injector{pk: make([]byte, slot.PKSize)}
	tr.pk[0] = pk
	return
}

func (t *injector) propose(prev slot.ID, rank byte, d []byte) *slot.Msg2 {
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

func (t *injector) vote(p *slot.Proposal2) *slot.Msg2 {
	return &slot.Msg2{
		Vote: &slot.Vote2{
			Proof: make([]byte, slot.ProofSize),
			Token: make([]byte, slot.TicketSize),
			PK:    t.pk,

			Proposal: p,
		},
	}
}

func TestProposerFilter(t *testing.T) {
	pf1 := slot.NewProposerFilter()

	n1 := pf1.Add([]byte{0x01}, []byte{0x01})
	test.Equals(t, 1, n1)

	n2 := pf1.Add([]byte{0x01}, []byte{0x02})
	test.Equals(t, 2, n2)

	n3 := pf1.Add([]byte{0x01}, []byte{0x01})
	test.Equals(t, 2, n3)

}

func TestBlockHash(t *testing.T) {
	b1 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x01}}
	test.Equals(t, "1fd4247443c9", fmt.Sprintf("%.6x", b1.Hash()))

	b2 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x02}}
	test.Equals(t, "58cc2f44d3a2", fmt.Sprintf("%.6x", b2.Hash()))

	b3 := &slot.Block2{Prev: b2.Hash(), Data: []byte{0x02}}
	test.Equals(t, "ae1b2a0343c2", fmt.Sprintf("%.6x", b3.Hash()))
}

func TestLottery(t *testing.T) {
	lt1 := slot.NewLottery(bytes.NewBuffer(make([]byte, 32)))
	rnd2 := make([]byte, 32)
	rnd2[0] = 0x01
	lt2 := slot.NewLottery(bytes.NewBuffer(rnd2))

	tk1, proof1, pk1 := lt1.Draw(1)
	test.Equals(t, "494326928e5a", fmt.Sprintf("%.6x", tk1))
	test.Equals(t, "72296e4cdeb3", fmt.Sprintf("%.6x", proof1))
	test.Equals(t, "d50ee45c5c2e", fmt.Sprintf("%.6x", pk1))

	tk1, proof1, pk1 = lt1.Draw(1) //must be deterministic for the same round nr
	test.Equals(t, "494326928e5a", fmt.Sprintf("%.6x", tk1))
	test.Equals(t, "72296e4cdeb3", fmt.Sprintf("%.6x", proof1))
	test.Equals(t, "d50ee45c5c2e", fmt.Sprintf("%.6x", pk1))

	tk2, proof2, pk2 := lt2.Draw(1) //must be different for other lottery
	test.Equals(t, "ac9765ebb028", fmt.Sprintf("%.6x", tk2))
	test.Equals(t, "e1f4afe8e7d5", fmt.Sprintf("%.6x", proof2))
	test.Equals(t, "4762ad6415ce", fmt.Sprintf("%.6x", pk2))

	tk2, proof2, pk2 = lt2.Draw(2) //must be different for another round
	test.Equals(t, "b02e361441c1", fmt.Sprintf("%.6x", tk2))
	test.Equals(t, "c8bdf013f1b9", fmt.Sprintf("%.6x", proof2))
	test.Equals(t, "4762ad6415ce", fmt.Sprintf("%.6x", pk2)) //same pk for different round
}

func TestMsgByMsgRoundComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	// protocol params
	bt := time.Millisecond * 10
	minv := uint64(1)

	// broadcast: ep2 -> ep1 -> coll1
	coll1, done1 := slot.Collect2(1)
	ep1 := slot.NewMemBroadcast2()
	ep1.Relay(coll1)
	ep2 := slot.NewMemBroadcast2()
	ep2.Relay(ep1)

	// rounds
	inj1 := newInjector(0x01)
	inj2 := newInjector(0x02)
	gen := slot.GenesisBlock().Hash()
	lt1 := slot.NewLottery(bytes.NewBuffer(make([]byte, 32)))
	r1 := slot.NewRound(lt1, ep1, 1, gen, bt, minv)

	// --- proposals ---

	msg1 := inj1.propose(gen, 0x01, []byte{0x01})
	test.Ok(t, ep2.Write(1, msg1))
	time.Sleep(bt * 2) //wait for timer to expire

	//@TODO send lower proposal again, should not broadcast as vote because
	//we have not reset top score block state for the voter.
	//@TODO wait for bt and write a higher ranking proposal, should broadcast
	//as vote also and immediately.

	// --- votes ---

	test.Ok(t, ep2.Write(1, inj1.vote(msg1.Proposal)))
	test.Ok(t, ep2.Write(1, inj2.vote(msg1.Proposal))) //second vote

	r2, err := r1.Run(ctx)
	test.Ok(t, err)

	msgs := <-done1()

	//@TODO should have made proposal to collector
	//@TODO should have relayed first proposal (high rank) to collector
	//@TODO should not have relayed second proposal (rank too low) to collector
	//@TODO should have cast vote for round 1 to collector

	//@TODO more assertion on message
	test.Equals(t, 5, len(msgs))
	test.Equals(t, uint64(2), r2.Num())
	test.Equals(t, msg1.Proposal.Block.Hash(), r2.Tip())
}

// @TODO test in ring network
