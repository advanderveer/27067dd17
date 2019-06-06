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

// injector allows for easy messages construction for the broadcast network
type injector struct {
	lt *slot.Lottery
}

func newInjector(pk byte) (tr *injector) {
	tr = &injector{}

	rnd := make([]byte, 32)
	rnd[0] = pk

	tr.lt = slot.NewLottery(bytes.NewReader(rnd))

	return
}

func (t *injector) propose(round uint64, prev slot.ID, rank byte, d []byte) *slot.Msg2 {
	token, proof, pk := t.lt.Draw(round)

	return &slot.Msg2{
		Proposal: &slot.Proposal2{
			Proof: proof,
			PK:    pk,
			Token: token,
			Block: &slot.Block2{
				Prev: prev,
				Data: d,
			},
		},
	}
}

func (t *injector) vote(round uint64, p *slot.Proposal2) *slot.Msg2 {
	token, proof, pk := t.lt.Draw(round)

	return &slot.Msg2{
		Vote: &slot.Vote2{
			Proof: proof,
			Token: token,
			PK:    pk,

			Proposal: p,
		},
	}
}

func TestValidation(t *testing.T) {
	b1 := &slot.Block2{}
	ok, err := b1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidBlockNilPrev, err)

	//proposal
	p1 := &slot.Proposal2{}
	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidProposalTokenLen, err)

	p1 = &slot.Proposal2{Token: make([]byte, slot.TicketSize)}
	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidProposalPKLen, err)

	p1 = &slot.Proposal2{Token: make([]byte, slot.TicketSize), PK: make([]byte, slot.PKSize)}
	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidProposalProofLen, err)

	p1 = &slot.Proposal2{Token: make([]byte, slot.TicketSize), PK: make([]byte, slot.PKSize), Proof: make([]byte, slot.ProofSize)}
	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidProposalNoBlock, err)

	//vote
	v1 := &slot.Vote2{}
	ok, err = v1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidVoteTokenLen, err)

	v1 = &slot.Vote2{Token: make([]byte, slot.TicketSize)}
	ok, err = v1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidVotePKLen, err)

	v1 = &slot.Vote2{Token: make([]byte, slot.TicketSize), PK: make([]byte, slot.PKSize)}
	ok, err = v1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidVoteProofLen, err)

	v1 = &slot.Vote2{Token: make([]byte, slot.TicketSize), PK: make([]byte, slot.PKSize), Proof: make([]byte, slot.ProofSize)}
	ok, err = v1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, slot.ErrInvalidVoteNoProposal, err)

	//valid
	b1.Prev[0] = 0x01
	p1.Block = b1
	v1.Proposal = p1
	ok, err = v1.Validate()
	test.Equals(t, true, ok)
	test.Ok(t, err)
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

func TestProposalRanking(t *testing.T) {
	p1 := &slot.Proposal2{Token: []byte{0x01}}
	p2 := &slot.Proposal2{Token: []byte{0x02}}
	p3 := &slot.Proposal2{Token: []byte{0x02}}

	test.Equals(t, "1", p1.Rank().String())
	test.Equals(t, "2", p2.Rank().String())
	test.Equals(t, "2", p3.Rank().String())

	test.Equals(t, true, p2.RanksGtOrEqThen(p1))
	test.Equals(t, true, p2.RanksGtOrEqThen(p3))
	test.Equals(t, false, p1.RanksGtOrEqThen(p2))
}

func TestProposalBlockHash(t *testing.T) {
	b1 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x01}}
	test.Equals(t, "1fd4247443c9", fmt.Sprintf("%.6x", b1.Hash()))

	b2 := &slot.Block2{Prev: slot.NilID, Data: []byte{0x02}}
	test.Equals(t, "58cc2f44d3a2", fmt.Sprintf("%.6x", b2.Hash()))

	b3 := &slot.Block2{Prev: b2.Hash(), Data: []byte{0x02}}
	test.Equals(t, "ae1b2a0343c2", fmt.Sprintf("%.6x", b3.Hash()))

	p1 := &slot.Proposal2{Token: []byte{0x01}, Proof: []byte{0x01}, PK: []byte{0x01}, Block: b1}
	test.Equals(t, "ce9e9711b023", fmt.Sprintf("%.6x", p1.Hash()))

	p2 := &slot.Proposal2{Token: []byte{0x02}, Proof: []byte{0x01}, PK: []byte{0x01}, Block: b1}
	test.Equals(t, "f12091840931", fmt.Sprintf("%.6x", p2.Hash()))

	p3 := &slot.Proposal2{Token: []byte{0x02}, Proof: []byte{0x02}, PK: []byte{0x01}, Block: b1}
	test.Equals(t, "6f0f8ab25f5e", fmt.Sprintf("%.6x", p3.Hash()))

	p4 := &slot.Proposal2{Token: []byte{0x02}, Proof: []byte{0x02}, PK: []byte{0x02}, Block: b1}
	test.Equals(t, "bc3292f69116", fmt.Sprintf("%.6x", p4.Hash()))

	p5 := &slot.Proposal2{Token: []byte{0x02}, Proof: []byte{0x02}, PK: []byte{0x02}, Block: b2}
	test.Equals(t, "17196dc4a370", fmt.Sprintf("%.6x", p5.Hash()))
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
	test.Equals(t, true, lt1.Verify(1, pk1, tk1, proof1))
	test.Equals(t, true, lt2.Verify(1, pk1, tk1, proof1)) //anyone can validate

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

func TestVoter(t *testing.T) {
	coll1, done1 := slot.Collect2(1)
	ep1 := slot.NewMemBroadcast2()
	ep1.Relay(coll1)
	lt1 := slot.NewLottery(bytes.NewBuffer(make([]byte, 32)))
	rnd2 := make([]byte, 32)
	rnd2[0] = 0x01

	top := &slot.Proposal2{Block: &slot.Block2{Prev: slot.NilID}}

	v1 := slot.NewVoter2(ep1, lt1, 1, top, time.Millisecond*100)

	//invalid proposal syntax
	res := v1.Handle(&slot.Proposal2{})
	test.Equals(t, true, res.SyntaxInvalid)
	test.Equals(t, slot.ErrInvalidProposalTokenLen, res.SyntaxValidationErr)

	//proposal token invalid
	b1 := &slot.Block2{Data: []byte{0x01}}
	b1.Prev[0] = 0x01
	res = v1.Handle(&slot.Proposal2{Token: make([]byte, slot.TicketSize), PK: make([]byte, slot.PKSize), Proof: make([]byte, slot.ProofSize), Block: b1})
	test.Equals(t, true, res.ProposalTokenInvalid)
	test.Equals(t, false, res.SyntaxInvalid)

	//non winning tip, @TODO test that a new vote's prev proposal rank is lower
	//then the incoming proposal's prev rank
	// token, proof, pk := lt1.Draw(1)
	// p1 := &slot.Proposal2{Token: token, PK: pk, Proof: proof, Block: b1}
	//
	// token2, proof2, pk2 := lt2.Draw(1)
	// p2 := &slot.Proposal2{Token: token2, PK: pk2, Proof: proof2, Block: b1}
	// test.Equals(t, true, p2.RanksGtOrEqThen(p1))
	//
	// res = v1.Handle(p2)
	// test.Equals(t, false, res.ProposalTokenInvalid)
	// test.Equals(t, false, res.NonWinningTip)
	//
	// res = v1.Handle(p1)
	// test.Equals(t, true, res.NonWinningTip)

	//@TODO this sometimes times out
	fmt.Println("waiting 1...")
	msgs := <-done1()
	fmt.Println(msgs)
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
	gen := &slot.Proposal2{Block: slot.GenesisBlock()}
	lt1 := slot.NewLottery(bytes.NewBuffer(make([]byte, 32)))
	r1 := slot.NewRound(lt1, ep1, 1, gen, bt, minv)

	// --- proposals ---

	msg1 := inj1.propose(1, gen.Block.Hash(), 0x01, []byte{0x01})
	test.Ok(t, ep2.Write(1, msg1))
	time.Sleep(bt * 2) //wait for timer to expire

	//@TODO send lower proposal again, should not broadcast as vote because
	//we have not reset top score block state for the voter.
	//@TODO wait for bt and write a higher ranking proposal, should broadcast
	//as vote also and immediately.

	// --- votes ---

	test.Ok(t, ep2.Write(1, inj1.vote(1, msg1.Proposal)))
	test.Ok(t, ep2.Write(1, inj2.vote(1, msg1.Proposal))) //second vote

	r2, err := r1.Run(ctx)
	test.Ok(t, err)

	fmt.Println("waiting 2...")
	msgs := <-done1()

	//@TODO should have made proposal to collector
	//@TODO should have relayed first proposal (high rank) to collector
	//@TODO should not have relayed second proposal (rank too low) to collector
	//@TODO should have cast vote for round 1 to collector

	//@TODO more assertion on message
	test.Equals(t, 5, len(msgs))
	test.Equals(t, uint64(2), r2.Num())
	test.Equals(t, msg1.Proposal.Block.Hash(), r2.Top().Block.Hash())
}

// @TODO test in ring network
