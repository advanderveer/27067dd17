package tt_test

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestBlockHashing(t *testing.T) {
	b1 := tt.B(tt.NilID, []byte{0x01})
	test.Equals(t, "1fd4247443c9", fmt.Sprintf("%.6x", b1.Hash()))

	b1.Data[0] = 0x02
	test.Equals(t, "58cc2f44d3a2", fmt.Sprintf("%.6x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "a3b7219472e4", fmt.Sprintf("%.6x", b1.Hash()))

	b1.Votes = append(b1.Votes, tt.Vote{})
	test.Equals(t, "dceccb1ee6a8", fmt.Sprintf("%.6x", b1.Hash()))
}

func TestPowerSet(t *testing.T) {
	test.Equals(t, [][]tt.VID{[]tt.VID(nil)}, tt.PowerSet())
	test.Equals(t, 2, len(tt.PowerSet(vid1))) //empty set and itself
	test.Equals(t, 4, len(tt.PowerSet(vid1, vid2)))
}

func TestBlockMine(t *testing.T) {

	b := &tt.Block{Prev: bid1, Data: []byte{0x01}}

	//@See https://jeiwan.cc/posts/building-blockchain-in-go-part-2/
	t.Run("difficulty 6 with 4 votes", func(t *testing.T) {
		difc := 6
		target := big.NewInt(1)
		target.Lsh(target, uint(256-difc))

		test.Equals(t, true, b.Mine(target, map[tt.VID]tt.Vote{
			vid1: tt.Vote{Token: []byte{0x01}},
			vid2: tt.Vote{Token: []byte{0x02}},
			vid3: tt.Vote{Token: []byte{0x03}},
			vid4: tt.Vote{Token: []byte{0x04}}}))
	})

	t.Run("difficulty 6 with 3 votes", func(t *testing.T) {
		difc := 6
		target := big.NewInt(1)
		target.Lsh(target, uint(256-difc))

		test.Equals(t, false, b.Mine(target, map[tt.VID]tt.Vote{
			vid1: tt.Vote{Token: []byte{0x01}},
			vid2: tt.Vote{Token: []byte{0x02}},
			vid3: tt.Vote{Token: []byte{0x03}}})) //not enough
	})

	t.Run("difficulty 7 with 4 votes", func(t *testing.T) {
		difc := 7
		target := big.NewInt(1)
		target.Lsh(target, uint(256-difc))

		test.Equals(t, false, b.Mine(target, map[tt.VID]tt.Vote{
			vid1: tt.Vote{Token: []byte{0x01}},
			vid2: tt.Vote{Token: []byte{0x02}},
			vid3: tt.Vote{Token: []byte{0x03}},
			vid4: tt.Vote{Token: []byte{0x04}}})) //nog enough
	})

	t.Run("difficulty 7 with 9 votes", func(t *testing.T) {
		difc := 7
		target := big.NewInt(1)
		target.Lsh(target, uint(256-difc))

		votes := map[tt.VID]tt.Vote{}
		for i := uint64(0); i < 9; i++ {
			v := &tt.Vote{Token: make([]byte, 32)}
			binary.LittleEndian.PutUint64(v.Token, i)
			votes[v.Hash()] = *v
		}

		test.Equals(t, true, b.Mine(target, votes)) //enough
	})
}
