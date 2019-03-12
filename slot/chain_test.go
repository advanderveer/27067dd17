package slot_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

//@TODO test concurrent operations

func TestChainCreationB(t *testing.T) {
	c1, err := slot.NewChain()
	test.Ok(t, err)

	test.Equals(t, uint64(1), c1.Round())

	tip1 := c1.Tip() //tip hash of genesis
	test.Equals(t, "6d9c54dee5660c46886f32d80e57e9dd0ffa57ee0cd2a762b036d9c8e0c3a33a", hex.EncodeToString(tip1[:]))

	b1, rank1 := c1.Read(tip1)
	test.Equals(t, 1, rank1)
	test.Equals(t, slot.NilID, b1.Prev)
	test.Equals(t, slot.NilTicket, b1.Ticket[:])
	test.Equals(t, slot.NilProof, b1.Proof[:])
	test.Equals(t, slot.NilPK, b1.PK[:])

	b2, rank2 := c1.Read(slot.NilID) //should not exist
	test.Equals(t, (*slot.Block)(nil), b2)
	test.Equals(t, 0, rank2)
}

func TestChainTicketDrawing(t *testing.T) {
	c1, err := slot.NewChain()
	test.Ok(t, err)

	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	ticket1, proof1, err := c1.Draw(pk, sk, c1.Tip())
	test.Ok(t, err)

	_, _, err = c1.Draw(pk, sk, slot.NilID)
	test.Equals(t, slot.ErrPrevNotExist, err)

	b1 := slot.NewBlock(c1.Round(), c1.Tip(), ticket1, proof1, pk)
	b11 := slot.NewBlock(c1.Round(), c1.Tip(), ticket1, proof1, pk)
	test.Equals(t, b1.Hash(), b11.Hash()) //must be deterministic

	ok, err := c1.Verify(b1)
	test.Ok(t, err)
	test.Equals(t, true, ok)

	c1.Progress(2)

	b2 := slot.NewBlock(c1.Round(), c1.Tip(), ticket1, proof1, pk)
	test.Assert(t, b2.Hash() != b11.Hash(), "new round causes block hash to differ")
}

func TestRankingStrengthAndTipSwapping(t *testing.T) {
	c, err := slot.NewChain()
	test.Ok(t, err)

	gen := c.Tip()

	b1 := slot.NewBlock(1, gen, slot.NilTicket, slot.NilProof, slot.NilPK)
	b1.Ticket[0] = 0x02
	id1 := c.Append(b1)
	test.Equals(t, id1, c.Tip())

	_, rank1 := c.Read(id1)
	test.Equals(t, 1, rank1)
	s1, _ := c.Strength(id1)
	test.Equals(t, "904625697166532776746648320380374280103671755200316906558262375061821325312", s1.FloatString(0))

	t.Run("non existing block strength", func(t *testing.T) {
		s, err := c.Strength(slot.NilID)
		test.Equals(t, slot.ErrPrevNotExist, err)
		test.Equals(t, (*big.Rat)(nil), s)
	})

	t.Run("lower strength", func(t *testing.T) {

		//adding another block should reduce the rank of the first block
		b2 := slot.NewBlock(1, gen, slot.NilTicket, slot.NilProof, slot.NilPK)
		b2.Ticket[0] = 0x03
		id2 := c.Append(b2)
		test.Equals(t, id2, c.Tip()) //should swap tip

		_, rank2 := c.Read(id2)
		test.Equals(t, 1, rank2)

		_, rank1 = c.Read(id1)
		test.Equals(t, 2, rank1) //now has a new rank
		s1, _ = c.Strength(id1)  //strength halved now since it moved down in rank
		test.Equals(t, "452312848583266388373324160190187140051835877600158453279131187530910662656", s1.FloatString(0))

		t.Run("swap only on higher strength", func(t *testing.T) {

			b3 := slot.NewBlock(1, gen, slot.NilTicket, slot.NilProof, slot.NilPK)
			b3.Ticket[0] = 0x01
			id3 := c.Append(b3)

			test.Equals(t, id2, c.Tip()) //should not swap tip

			_, rank3 := c.Read(id3)
			test.Equals(t, 3, rank3)
		})
	})
}

var (
	ticketS1 = [slot.TicketSize]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS2 = [slot.TicketSize]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS3 = [slot.TicketSize]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

func TestMidwayTipSwap(t *testing.T) {
	c, err := slot.NewChain()
	test.Ok(t, err)

	genid := c.Tip()
	_, genr := c.Read(genid)
	test.Equals(t, 1, genr)

	b1 := slot.NewBlock(2, c.Tip(), slot.NilTicket, slot.NilProof, slot.NilPK) //zero ticket doesn't increase sum weight
	c.Append(b1)
	test.Equals(t, genid, c.Tip()) //not bigger then genesis weight, still the tip

	b2 := slot.NewBlock(2, c.Tip(), ticketS1[:], slot.NilProof, slot.NilPK)
	id2 := c.Append(b2)
	test.Equals(t, id2, c.Tip()) //s1 ticket is bigger then genesis
	_, r2 := c.Read(id2)
	test.Equals(t, 1, r2)

	b3 := slot.NewBlock(3, c.Tip(), slot.NilTicket, slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	c.Append(b3)
	test.Equals(t, id2, c.Tip()) //adding zero weight ticket doesnt change tip

	b4 := slot.NewBlock(3, c.Tip(), ticketS2[:], slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	id4 := c.Append(b4)
	test.Equals(t, id4, c.Tip()) //s1 ticket is bigger then genesis
	_, r4 := c.Read(id4)
	test.Equals(t, 1, r4)

	tipS1, _ := c.Strength(id4)
	expS := new(big.Rat).SetFrac(new(big.Int).SetBytes(ticketS3[:]), big.NewInt(1))
	test.Equals(t, 0, tipS1.Cmp(expS)) //stregnth should be "3"

	//now we add a new tip with strength "3" at a lower round. The tips selection
	//should switch over because it will reduce the rank of b4 (current tip)
	b5 := slot.NewBlock(2, genid, ticketS3[:], slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	id5 := c.Append(b5)
	test.Equals(t, id5, c.Tip())
}

func TestChainAppending(t *testing.T) {
	c, err := slot.NewChain()
	test.Ok(t, err)

	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	//create a chain of 100 blocks, each in its own round
	var ids []slot.ID
	n := 100 //@TODO increasing n shouldn't increase testing speed linearly
	var strength *big.Rat
	for i := 0; i < n; i++ {
		ticket, proof, err := c.Draw(pk, sk, c.Tip())
		test.Ok(t, err)

		strength, _ = c.Strength(c.Tip())
		if i == 0 {
			test.Equals(t, "0.0", strength.FloatString(1))
		}

		b := slot.NewBlock(c.Round(), c.Tip(), ticket, proof, pk)
		ids = append(ids, b.Hash())

		ok, err := c.Verify(b)
		test.Ok(t, err)
		test.Equals(t, true, ok)

		c.Append(b)
		test.Equals(t, b.Hash(), c.Tip()) //should become new tip

		c.Progress(c.Round() + 1)
	}

	//assert chain
	test.Equals(t, n, len(ids))
	test.Equals(t, "5814292125257717654856305910480344465283549495696731797414878196001414625954011", strength.FloatString(0))

	//walk backwards from the new tip
	test.Ok(t, c.Walk(c.Tip(), func(id slot.ID, b *slot.Block, rank int) error {
		n--
		test.Equals(t, uint64(n+1), b.Round)
		test.Equals(t, 1, rank)

		if n > 0 { //-1 is the genesis block
			test.Equals(t, ids[n], id)
		}

		return nil
	}))

	t.Run("walk from unknown block", func(t *testing.T) {
		test.Equals(t, slot.ErrPrevNotExist, c.Walk(slot.NilID, nil))
	})

	t.Run("walk from unknown block", func(t *testing.T) {
		e := fmt.Errorf("foo")
		test.Equals(t, e, c.Walk(c.Tip(), func(id slot.ID, b *slot.Block, rank int) error {
			return e
		}))
	})

}

func TestVerify(t *testing.T) {
	//@TODO test all the verification edge cases
}
