package rev_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

func TestProposalIDSet(t *testing.T) {
	pid1 := rev.PID{}
	pid2 := rev.PID{}
	pid2[0] = 0x03
	pid3 := rev.PID{}
	pid3[0] = 0x02

	ps1 := rev.PSet(pid1, pid2, pid3)
	test.Equals(t, pid1, ps1.Sorted()[0])
	test.Equals(t, pid3, ps1.Sorted()[1])
	test.Equals(t, pid2, ps1.Sorted()[2])

	for i := 0; i < 10; i++ {
		sum := ps1.Hash()
		test.Equals(t, "a650a6c9415b", fmt.Sprintf("%.6x", sum))
	}
}

func TestProposalHashing(t *testing.T) {
	rnd1 := bytes.NewReader(make([]byte, 32))
	pk1, sk1, err := vrf.GenerateKey(rnd1)
	test.Ok(t, err)
	p1 := rev.NewProposal(pk1, sk1, 1)
	p1.Block = rev.B(nil, rev.NilID)
	p1.Witness = rev.PSet()

	test.Equals(t, "0b53d964c421", fmt.Sprintf("%.6x", p1.Hash()))
	p1.Block.Prev[0] = 0x01
	test.Equals(t, "c861987c0f25", fmt.Sprintf("%.6x", p1.Hash()))
	p1.Witness[rev.PID{}] = struct{}{}
	test.Equals(t, "7094b3301b95", fmt.Sprintf("%.6x", p1.Hash()))
	p1.PK[0] = 0x01
	test.Equals(t, "5c89ad55442f", fmt.Sprintf("%.6x", p1.Hash()))
	p1.Proof[0] = 0x01
	test.Equals(t, "0a8350c2b537", fmt.Sprintf("%.6x", p1.Hash()))
	p1.Token[0] = 0x01
	test.Equals(t, "5ee6d23c6219", fmt.Sprintf("%.6x", p1.Hash()))
	p1.Round = 2
	test.Equals(t, "365de8b6fa5b", fmt.Sprintf("%.6x", p1.Hash()))
}

func TestProposalValidation(t *testing.T) {
	rnd1 := bytes.NewReader(make([]byte, 32))
	pk1, sk1, err := vrf.GenerateKey(rnd1)
	test.Ok(t, err)
	p1 := rev.NewProposal(pk1, sk1, 1)

	ok, err := p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, rev.ErrProposalHasNoBlock, err)
	p1.Block = rev.B([]byte{0x01}, rev.NilID)

	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, rev.ErrProposalHasNoWitness, err)
	p1.Witness = rev.PSet(p1.Hash())

	ok, err = p1.Validate()
	test.Equals(t, true, ok)
	test.Ok(t, err)

	p1.Round = 2
	ok, err = p1.Validate()
	test.Equals(t, false, ok)
	test.Equals(t, rev.ErrProposalTokenInvalid, err)
}
