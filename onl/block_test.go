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
	test.Equals(t, "0000000000000001382ab918", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(1), b1.Hash().Round())

	//expect the hash to change on every field manipulation
	b1.FinalizedPrev[0] = 0x01
	test.Equals(t, "0000000000000001e48f6195", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "0000000000000001cc06c4ff", fmt.Sprintf("%.12x", b1.Hash()))

	b1.PK[0] = 0x01
	test.Equals(t, "000000000000000172979cb7", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Proof[0] = 0x01
	test.Equals(t, "0000000000000001bdeab958", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Token[0] = 0x01
	test.Equals(t, "0000000000000001a7018c85", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Timestamp += 1
	test.Equals(t, "0000000000000001c0a77287", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Round = 100
	test.Equals(t, "0000000000000064ba226bc6", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(100), b1.Hash().Round())

	b1.Append(&onl.Write{TxData: &ssi.TxData{}})
	test.Equals(t, "0000000000000064a51ff1c1", fmt.Sprintf("%.12x", b1.Hash()))
}

func TestConsistentWritesHashing(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	st1, err := onl.NewState(nil)
	test.Ok(t, err)
	w1 := st1.Update(func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)             //mint 1 current
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK()) //then deposit it
	})

	b1 := idn1.Mint(testClock(1), bid1, bid1, 1)
	b1.Append(w1)
	for i := 0; i < 100; i++ { //should hash consistently
		test.Equals(t, b1.Hash(), b1.Hash())
	}
}

func TestBlockMintingSigningVerification(t *testing.T) {
	c1 := testClock(1)
	idn1 := onl.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(c1, bid1, bid2, 1)
	test.Equals(t, false, b1.VerifySignature())

	idn1.Sign(b1)
	test.Equals(t, "b8474890", fmt.Sprintf("%.4x", b1.Signature))
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
