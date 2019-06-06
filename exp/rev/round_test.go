package rev_test

import (
	"bytes"
	"testing"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

func TestRoundObservation(t *testing.T) {
	rnd1 := bytes.NewReader(make([]byte, 32))
	pk1, sk1, err := vrf.GenerateKey(rnd1)
	test.Ok(t, err)
	p0 := rev.NewProposal(pk1, sk1, 1)
	p0.Block = &rev.Block{}

	c1 := rev.NewChain(3)
	r1 := rev.NewRound(c1, p0)

	//new proposal, that is lower in rank
	p1 := rev.NewProposal(pk1, sk1, 1)
	p1.Block = &rev.Block{Data: []byte{0x01}}

	_, _, wit1, tip1 := r1.Observe(p1)
	test.Assert(t, wit1 == nil, "witness should be nill")
	// test.Equals(t, p1.Block.Hash(), tip1) //doesn't rank high enough

	//new proposal, that is higher in rank
	p2 := rev.NewProposal(pk1, sk1, 3)
	p2.Block = &rev.Block{Data: []byte{0x01}}

	_, _, wit1, tip1 = r1.Observe(p2)
	test.Assert(t, wit1 != nil, "witness should not be nill")
	test.Equals(t, 3, len(wit1)) //should bring all observed proposals
	_ = tip1
	// test.Equals(t, p2.Block.Hash(), tip1) //should be high enough rankin //@TODO re-enable
}

func TestRoundQuestion(t *testing.T) {
	rnd1 := bytes.NewReader(make([]byte, 32))
	pk1, sk1, err := vrf.GenerateKey(rnd1)
	test.Ok(t, err)

	p0 := rev.NewProposal(pk1, sk1, 1)
	p0.Block = &rev.Block{Data: []byte{0x01}}

	p1 := rev.NewProposal(pk1, sk1, 2)
	p1.Block = &rev.Block{Prev: p0.Block.Hash(), Data: []byte{0x02}}
	p1.Witness = rev.PSet(p0.Hash())

	p2 := rev.NewProposal(pk1, sk1, 2)
	p2.Block = &rev.Block{Prev: p0.Block.Hash(), Data: []byte{0x03}}
	p2.Witness = rev.PSet(p1.Hash())

	p3 := rev.NewProposal(pk1, sk1, 2)
	p3.Block = &rev.Block{Prev: p1.Block.Hash(), Data: []byte{0x04}}
	p3.Witness = rev.PSet(p0.Hash(), p2.Hash())

	c1 := rev.NewChain(2)
	r1 := rev.NewRound(c1, p0)

	//p1 only has 1 witness
	ok, err := r1.Question(p1.Block.Prev, p1.Witness)
	test.Equals(t, rev.ErrNotEnoughWitness, err)
	test.Equals(t, false, ok)

	//p2 has witness p1 which is not observed by this round
	ok, err = r1.Question(p2.Block.Prev, p2.Witness)
	test.Equals(t, rev.ErrProposalWitnessUnknown, err)
	test.Equals(t, false, ok)

	//@TODO re-enable tests below
	// r1.Observe(p2)

	//p3 has enough witness but prev block refers to a non-existing proposal
	// ok, err = r1.Question(p3.Block.Prev, p3.Witness)
	// test.Equals(t, rev.ErrPrevProposalNotFound, err)
	// test.Equals(t, false, ok)

	// r1.Observe(p1)

	// //all witness are observed but the prev block is not in the top witness
	// ok, err = r1.Question(p3.Block.Prev, p3.Witness)
	// test.Equals(t, rev.ErrPrevProposalNotTopWitness, err)
	// test.Equals(t, false, ok)

}
