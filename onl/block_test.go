package onl_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

var (
	//some test ids
	bid1 = onl.ID{}
	bid2 = onl.ID{}
	bid3 = onl.ID{}
	bid4 = onl.ID{}
)

func init() {
	bid1[0] = 0x01
	bid2[0] = 0x02
	bid3[0] = 0x03
	bid4[0] = 0x04
}

func TestBlockHashing(t *testing.T) {
	c1 := testClock(1)
	idn1 := onl.NewIdentity([]byte{0x01})

	b1 := idn1.Mint(c1, bid1, bid2, 1)
	b1.AppendOps(idn1.Join(1))
	test.Equals(t, "00000000000000015a259acd", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(1), b1.Hash().Round())

	//expect the hash to change on every field manipulation
	b1.FinalizedPrev[0] = 0x01
	test.Equals(t, "000000000000000182887e46", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "000000000000000157c2d528", fmt.Sprintf("%.12x", b1.Hash()))

	b1.PK[0] = 0x01
	test.Equals(t, "0000000000000001388021a7", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Proof[0] = 0x01
	test.Equals(t, "00000000000000010b3d8f30", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Token[0] = 0x01
	test.Equals(t, "0000000000000001d8831006", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Timestamp += 1
	test.Equals(t, "00000000000000015b946ef8", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Round = 100
	test.Equals(t, "000000000000006437a0c978", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(100), b1.Hash().Round())

	b1.AppendOps(idn1.Join(2))
	test.Equals(t, "0000000000000064042e268c", fmt.Sprintf("%.12x", b1.Hash()))
}

func TestBlockMintingSigningVerification(t *testing.T) {
	c1 := testClock(1)
	idn1 := onl.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(c1, bid1, bid2, 1)
	test.Equals(t, false, b1.VerifySignature())

	idn1.Sign(b1)
	test.Equals(t, "20781440", fmt.Sprintf("%.4x", b1.Signature))
	test.Equals(t, true, b1.VerifySignature())

	//crypto should verify
	test.Equals(t, true, b1.VerifyToken(idn1.TokenPK()))

	//different round should invalidate the vrf
	b1.Round = 2
	test.Equals(t, false, b1.VerifyToken(idn1.TokenPK()))

	//changing a field should invalidate the block signature
	b1.Timestamp += 1
	test.Equals(t, false, b1.VerifySignature())
}
