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

	//round times
	rt1s = uint64(1000000) //1second
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

	b1 := idn1.Mint(c1, rt1s, bid1, bid2)
	b1.AppendOps(idn1.Join(1))
	test.Equals(t, "de76b92f", fmt.Sprintf("%.4x", b1.Hash()))

	//expect the hash to change on every field manipulation
	b1.FinalizedPrev[0] = 0x01
	test.Equals(t, "0a5c6480", fmt.Sprintf("%.4x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "674795c0", fmt.Sprintf("%.4x", b1.Hash()))

	b1.PK[0] = 0x01
	test.Equals(t, "eef40c8e", fmt.Sprintf("%.4x", b1.Hash()))

	b1.Proof[0] = 0x01
	test.Equals(t, "14ee4ccb", fmt.Sprintf("%.4x", b1.Hash()))

	b1.Token[0] = 0x01
	test.Equals(t, "17dab4ef", fmt.Sprintf("%.4x", b1.Hash()))

	b1.Timestamp = 100
	test.Equals(t, "2e528c81", fmt.Sprintf("%.4x", b1.Hash()))

	b1.AppendOps(idn1.Join(2))
	test.Equals(t, "88bc9eef", fmt.Sprintf("%.4x", b1.Hash()))
}

func TestBlockMintingSigningVerification(t *testing.T) {
	c1 := testClock(1)
	idn1 := onl.NewIdentity([]byte{0x01})

	b1 := idn1.Mint(c1, rt1s, bid1, bid2)
	idn1.Sign(b1)
	test.Equals(t, "20732fa5", fmt.Sprintf("%.4x", b1.Signature))

	//crypto should verify
	ok, err := b1.VerifyCrypto(idn1.TokenPK(), rt1s)
	test.Ok(t, err)
	test.Equals(t, true, ok)

	//different roundtime should invalidate the vrf
	ok, err = b1.VerifyCrypto(idn1.TokenPK(), 1)
	test.Equals(t, false, ok)
	test.Equals(t, onl.ErrInvalidToken, err)

	//changing a field should invalidate the block signature
	b1.Timestamp += 1
	ok, err = b1.VerifyCrypto(idn1.TokenPK(), rt1s)
	test.Equals(t, false, ok)
	test.Equals(t, onl.ErrInvalidSignature, err)

	//@TODO test that the clock @.9999second still has the same vrf token as it
	//bucketed in the same round

}
