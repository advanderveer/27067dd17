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
	b1.AppendWrite(&onl.Write{TxData: &ssi.TxData{}})
	test.Equals(t, "fffffffffffffffe9443880e", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(1), b1.Hash().Round())

	//expect the hash to change on every field manipulation
	b1.FinalizedPrev[0] = 0x01
	test.Equals(t, "fffffffffffffffe71974c3e", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "fffffffffffffffe5afeebb2", fmt.Sprintf("%.12x", b1.Hash()))

	b1.PK[0] = 0x01
	test.Equals(t, "fffffffffffffffe4f83959b", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Proof[0] = 0x01
	test.Equals(t, "fffffffffffffffe9257ac8b", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Token[0] = 0x01
	test.Equals(t, "fffffffffffffffe11ad84f8", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Timestamp += 1
	test.Equals(t, "fffffffffffffffe10aa6ee1", fmt.Sprintf("%.12x", b1.Hash()))

	b1.Round = 100
	test.Equals(t, "ffffffffffffff9bf8c446ed", fmt.Sprintf("%.12x", b1.Hash()))
	test.Equals(t, uint64(100), b1.Hash().Round())

	b1.AppendWrite(&onl.Write{TxData: &ssi.TxData{}})
	test.Equals(t, "ffffffffffffff9b0656fdc7", fmt.Sprintf("%.12x", b1.Hash()))
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
	b1.AppendWrite(w1)
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
	test.Equals(t, "7d77be7b", fmt.Sprintf("%.4x", b1.Signature))
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

func TestBlockRanking(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(testClock(1), bid1, bid2, 1)

	test.Equals(t, "0", b1.Rank(0).Text(10)) //should equal exactly 0
	test.Equals(t, "97552841951904930067318056973531093736152646717171940036091344696617083484138", b1.Rank(1).Text(10))
	test.Equals(t, "195105683903809860134636113947062187472305293434343880072182689393234166968276", b1.Rank(2).Text(10))
}
