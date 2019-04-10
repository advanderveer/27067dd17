package topn

import (
	"testing"

	"github.com/advanderveer/go-test"
)

func TestRoundAdding(t *testing.T) {
	var (
		bid1 ID
		bid2 ID
		bid3 ID
	)

	bid1[0] = 0x01
	bid2[0] = 0x02
	bid3[0] = 0x03

	idn1 := NewIdentity([]byte{0x01})
	b1 := idn1.CreateBlock(1, bid1)

	r1 := newRound()
	r1.Set(b1.Hash(), b1, b1.Rank(1))

	test.Equals(t, true, r1.HasBlock(b1.Hash()))
	test.Equals(t, false, r1.HasBlock(bid1))

	test.Equals(t, true, r1.SawIdentity(b1.PK))
	test.Equals(t, false, r1.SawIdentity(PK{}))

	b2 := idn1.CreateBlock(1, bid2)
	b3 := idn1.CreateBlock(1, bid3)
	r1.Set(b2.Hash(), b2, b2.Rank(1))
	r1.Set(b3.Hash(), b3, b3.Rank(1))

	var ranking []ID
	r1.Ranked(func(pos int, id ID, b *Block) {
		test.Assert(t, pos > 0, "post must start at 1")
		ranking = append(ranking, id)
	})

	test.Equals(t, []ID{b3.Hash(), b1.Hash(), b2.Hash()}, ranking)

	r1.Set(b2.Hash(), b2, b2.Rank(10)) ///should rank higher after adding the stake

	ranking = []ID{}
	r1.Ranked(func(pos int, id ID, b *Block) {
		ranking = append(ranking, id)
	})

	test.Equals(t, []ID{b2.Hash(), b3.Hash(), b1.Hash()}, ranking)
}
