package onl_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/ssi"
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
	b1.Append(&onl.Write{TxData: &ssi.TxData{}})
	test.Equals(t, "0000000000000001ab30a254", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(1), b1.Hash().Round())

	//expect the hash to change on every field manipulation
	b1.FinalizedPrev[0] = 0x01
	test.Equals(t, "000000000000000109961974", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "0000000000000001ee4b94b9", fmt.Sprintf("%.12x", b1.Hash()))

	b1.PK[0] = 0x01
	test.Equals(t, "000000000000000117352713", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Proof[0] = 0x01
	test.Equals(t, "00000000000000015a79c45e", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Token[0] = 0x01
	test.Equals(t, "0000000000000001ca5215a7", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Timestamp += 1
	test.Equals(t, "0000000000000001091060cb", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Round = 100
	test.Equals(t, "0000000000000064db7b1249", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(100), b1.Hash().Round())

	b1.Append(&onl.Write{TxData: &ssi.TxData{}})
	test.Equals(t, "0000000000000064b451f880", fmt.Sprintf("%.12x", b1.Hash()))
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
