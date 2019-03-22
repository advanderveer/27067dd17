package rev_test

import (
	"bytes"
	"testing"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

var _ rev.Handler = &rev.OutOfOrder{}

func p(round uint64, r byte) *rev.Proposal {
	rnd := make([]byte, 32)
	rnd[0] = r
	pk, sk, err := vrf.GenerateKey(bytes.NewReader(rnd))
	if err != nil {
		panic(err)
	}

	return rev.NewProposal(pk, sk, round)
}

func TestOutOfOrder(t *testing.T) {
	var handled []*rev.Proposal
	o1 := rev.NewOutOfOrder(rev.HandlerFunc(func(p *rev.Proposal) {
		handled = append(handled, p)
		return
	}))

	p1 := p(1, 0x01)
	p2 := p(2, 0x02)
	p2.Witness.Add(p1.Hash())
	p3 := p(3, 0x03)
	p3.Witness.Add(p2.Hash())
	p4 := p(4, 0x04)
	p4.Witness.Add(p3.Hash(), p1.Hash())
	p5 := p(5, 0x05)
	p5.Witness.Add(p1.Hash())

	//  /---------------------p5
	// p1 <- p2 <- p3 <- p4
	//  \-------<- -----/

	o1.Handle(p5)
	test.Equals(t, 0, len(handled)) //p5 still waiting on p1
	o1.Handle(p1)
	test.Equals(t, []*rev.Proposal{p1, p5}, handled) //p1, p5 should be handled

	o1.Handle(p4)
	test.Equals(t, 2, len(handled)) //waits for p3
	o1.Handle(p3)
	test.Equals(t, 2, len(handled)) //waits for p2
	o1.Handle(p2)
	test.Equals(t, 5, len(handled)) //resolved 3 and 4

	test.Equals(t, []*rev.Proposal{p1, p5, p2, p3, p4}, handled) //p1, p5 should be handled

	no, nh := o1.Size()
	test.Equals(t, 0, no)
	test.Equals(t, 5, nh)
}
