package topn_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/topn"
	"github.com/advanderveer/go-test"
)

var (
	//some test ids
	bid1 = topn.ID{}
	bid2 = topn.ID{}
	bid3 = topn.ID{}
	bid4 = topn.ID{}
)

func init() {
	bid1[0] = 0x01
	bid2[0] = 0x02
	bid3[0] = 0x03
	bid4[0] = 0x04
}

func TestTxHashing(t *testing.T) {
	tx := &topn.Tx{FromPK: []byte{0x01}, ToPK: []byte{0x02}, Amount: 1}
	test.Equals(t, "5e90e8fc309c", fmt.Sprintf("%.6x", tx.Hash()))

	tx.FromPK = []byte{0x02}
	test.Equals(t, "9a3cecbd9cff", fmt.Sprintf("%.6x", tx.Hash()))
	tx.ToPK = []byte{0x01}
	test.Equals(t, "3c51fa98fc45", fmt.Sprintf("%.6x", tx.Hash()))
	tx.Amount = 2
	test.Equals(t, "06bb465e3930", fmt.Sprintf("%.6x", tx.Hash()))
}

func TestBlockHashing(t *testing.T) {
	idn1 := topn.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(1, bid1)
	tx1 := &topn.Tx{FromPK: []byte{0x01}, ToPK: []byte{0x02}, Amount: 1}
	b1.AppendTx(tx1)

	test.Equals(t, uint64(1), b1.Hash().Round())
	test.Equals(t, "0000000000000001f7de863295e8", fmt.Sprintf("%.14x", b1.Hash().Bytes()))
	b1.Round = 2

	test.Equals(t, uint64(2), b1.Hash().Round())
	test.Equals(t, "00000000000000026f7fef30a237", fmt.Sprintf("%.14x", b1.Hash().Bytes()))

	b1.Prev = bid2
	test.Equals(t, "0000000000000002347e426e2fb5", fmt.Sprintf("%.14x", b1.Hash().Bytes()))

	b1.Txs[0].Amount = 2
	test.Equals(t, "00000000000000021e8f825e333d", fmt.Sprintf("%.14x", b1.Hash().Bytes()))

	b1.Token[0] = 0x01
	test.Equals(t, "00000000000000026864580e4d42", fmt.Sprintf("%.14x", b1.Hash().Bytes()))

	b1.Proof[0] = 0x01
	test.Equals(t, "0000000000000002c98af3f407d8", fmt.Sprintf("%.14x", b1.Hash().Bytes()))

	b1.PK[0] = 0x01
	test.Equals(t, "0000000000000002f945717908f4", fmt.Sprintf("%.14x", b1.Hash().Bytes()))
}

func TestBlockPrinting(t *testing.T) {
	idn1 := topn.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(1, bid1)
	test.Equals(t, "1-9f426a5e", fmt.Sprint(b1))

	b1.Round = 2
	test.Equals(t, "2-e9ad842b", fmt.Sprint(b1))
}

func TestBlockRanking(t *testing.T) {
	idn1 := topn.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(1, bid1)

	test.Equals(t, "0", b1.Rank(0).Text(10)) //should equal exactly 0
	test.Equals(t, "21135040306902344126042764464942834682251382364299118602526470152441247899096", b1.Rank(1).Text(10))
	test.Equals(t, "42270080613804688252085528929885669364502764728598237205052940304882495798192", b1.Rank(2).Text(10))

}
