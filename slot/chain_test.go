package slot_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

//tickets of various of strength 1,2,3 etc
var (
	ticketS1 = [slot.TicketSize]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS2 = [slot.TicketSize]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS3 = [slot.TicketSize]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

//@TODO test concurrent operations

func TestChainCreationB(t *testing.T) {
	c1 := slot.NewChain()

	// test.Equals(t, uint64(1), c1.Round())

	tip1 := c1.Tip() //tip hash of genesis
	test.Equals(t, "6d9c54dee5660c46886f32d80e57e9dd0ffa57ee0cd2a762b036d9c8e0c3a33a", hex.EncodeToString(tip1[:]))

	b1 := c1.Read(tip1)
	rank1 := c1.Rank(tip1)
	test.Equals(t, 1, rank1)
	test.Equals(t, slot.NilID, b1.Prev)
	test.Equals(t, slot.NilTicket, b1.Ticket[:])
	test.Equals(t, slot.NilProof, b1.Proof[:])
	test.Equals(t, slot.NilPK, b1.PK[:])

	b2 := c1.Read(slot.NilID) //should not exist
	test.Equals(t, (*slot.Block)(nil), b2)
}

func TestChainTicketDrawing(t *testing.T) {
	c1 := slot.NewChain()

	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	t1, err := c1.Draw(pk, sk, c1.Tip(), 1)
	test.Ok(t, err)

	_, err = c1.Draw(pk, sk, slot.NilID, 1)
	test.Equals(t, slot.ErrPrevNotExist, err)

	b1 := slot.NewBlock(1, c1.Tip(), t1.Data, t1.Proof, pk)
	b11 := slot.NewBlock(1, c1.Tip(), t1.Data, t1.Proof, pk)
	test.Equals(t, b1.Hash(), b11.Hash()) //must be deterministic

	// @TODO verify proof?
	// ok, err := c1.Verify(b1)
	// test.Ok(t, err)
	// test.Equals(t, true, ok)

	// c1.Progress(2)

	b2 := slot.NewBlock(2, c1.Tip(), t1.Data, t1.Proof, pk)
	test.Assert(t, b2.Hash() != b11.Hash(), "new round causes block hash to differ")
}

func TestRankingStrengthAndTipSwapping(t *testing.T) {
	c := slot.NewChain()

	gen := c.Tip()

	b1 := slot.NewBlock(1, gen, slot.NilTicket, slot.NilProof, slot.NilPK)
	b1.Ticket[0] = 0x02
	id1, newt1 := c.Append(b1)
	test.Equals(t, id1, c.Tip())
	test.Equals(t, true, newt1)

	rank1 := c.Rank(id1)
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
		id2, _ := c.Append(b2)
		test.Equals(t, id2, c.Tip()) //should swap tip

		rank2 := c.Rank(id2)
		test.Equals(t, 1, rank2)

		rank1 = c.Rank(id1)
		test.Equals(t, 2, rank1) //now has a new rank
		s1, _ = c.Strength(id1)  //strength halved now since it moved down in rank
		test.Equals(t, "452312848583266388373324160190187140051835877600158453279131187530910662656", s1.FloatString(0))

		t.Run("swap only on higher strength", func(t *testing.T) {

			b3 := slot.NewBlock(1, gen, slot.NilTicket, slot.NilProof, slot.NilPK)
			b3.Ticket[0] = 0x01
			id3, _ := c.Append(b3)

			test.Equals(t, id2, c.Tip()) //should not swap tip

			rank3 := c.Rank(id3)
			test.Equals(t, 3, rank3)
		})
	})
}

func TestMidwayTipSwap(t *testing.T) {
	c := slot.NewChain()

	genid := c.Tip()
	genr := c.Rank(genid)
	test.Equals(t, 1, genr)

	b1 := slot.NewBlock(2, c.Tip(), slot.NilTicket, slot.NilProof, slot.NilPK) //zero ticket doesn't increase sum weight
	c.Append(b1)
	test.Equals(t, genid, c.Tip()) //not bigger then genesis weight, still the tip

	b2 := slot.NewBlock(2, c.Tip(), ticketS1[:], slot.NilProof, slot.NilPK)
	id2, _ := c.Append(b2)
	test.Equals(t, id2, c.Tip()) //s1 ticket is bigger then genesis
	r2 := c.Rank(id2)
	test.Equals(t, 1, r2)

	b3 := slot.NewBlock(3, c.Tip(), slot.NilTicket, slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	c.Append(b3)
	test.Equals(t, id2, c.Tip()) //adding zero weight ticket doesnt change tip

	b4 := slot.NewBlock(3, c.Tip(), ticketS2[:], slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	id4, _ := c.Append(b4)
	test.Equals(t, id4, c.Tip()) //s1 ticket is bigger then genesis
	r4 := c.Rank(id4)
	test.Equals(t, 1, r4)

	tipS1, _ := c.Strength(id4)
	expS := new(big.Rat).SetFrac(new(big.Int).SetBytes(ticketS3[:]), big.NewInt(1))
	test.Equals(t, 0, tipS1.Cmp(expS)) //stregnth should be "3"

	//now we add a new tip with strength "3" at a lower round. The tips selection
	//should switch over because it will reduce the rank of b4 (current tip)
	b5 := slot.NewBlock(2, genid, ticketS3[:], slot.NilProof, slot.NilPK) //1 weight ticket does in crease weight
	id5, _ := c.Append(b5)
	test.Equals(t, id5, c.Tip())
}

func TestChainAppending(t *testing.T) {
	c := slot.NewChain()

	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	//create a chain of 100 blocks, each in its own round
	var ids []slot.ID
	n := uint64(100) //@TODO increasing n shouldn't increase testing speed linearly
	var strength *big.Rat
	for i := uint64(0); i < n; i++ {
		ticket, err := c.Draw(pk, sk, c.Tip(), i+1)
		test.Ok(t, err)

		strength, _ = c.Strength(c.Tip())
		if i == 0 {
			test.Equals(t, "0.0", strength.FloatString(1))
		}

		b := slot.NewBlock(i+1, c.Tip(), ticket.Data, ticket.Proof, pk)
		ids = append(ids, b.Hash())

		// @TODO verify?
		// ok, err := c.Verify(b)
		// test.Ok(t, err)
		// test.Equals(t, true, ok)

		id, _ := c.Append(b)
		test.Equals(t, b.Hash(), c.Tip()) //should become new tip
		test.Equals(t, id, c.Tip())       //should become new tip

		// c.Progress(c.Round() + 1)
	}

	//assert chain
	test.Equals(t, n, uint64(len(ids)))
	test.Equals(t, "5814292125257717654856305910480344465283549495696731797414878196001414625954011", strength.FloatString(0))

	//walk each block
	var j uint64
	test.Ok(t, c.Each(func(id slot.ID, b *slot.Block) error {
		j++
		test.Assert(t, id != slot.NilID, "should all have id")
		test.Assert(t, b != nil, "should see blocks")
		return nil
	}))

	test.Equals(t, uint64(n+1), j) //100 plus genesis

	//walk backwards from the new tip
	test.Ok(t, c.Walk(c.Tip(), func(id slot.ID, b *slot.Block, rank int) error {
		n--
		test.Equals(t, uint64(n+1), b.Round)
		test.Equals(t, 1, rank)

		if n < math.MaxUint64 {
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

func TestTally(t *testing.T) {
	c := slot.NewChain()
	b1 := &slot.Vote{Block: slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}
	n1 := c.Tally(b1)
	test.Equals(t, 1, n1)

	n2 := c.Tally(b1) //adding the same shouldnt increase the vote count
	test.Equals(t, 1, n2)

	b2 := &slot.Vote{Block: slot.NewBlock(1, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)}
	b2.VoteTicket[0] = 0x01 //different ticket should add another vote

	n3 := c.Tally(b2) //adding a vote for the same block with another ticket
	test.Equals(t, 2, n3)

	n4 := c.Tally(b2) //adding the same ticket should not increase votes
	test.Equals(t, 2, n4)
}

func TestThreshold(t *testing.T) {
	//scenario: just the genesis block and genesis threshold
	//scenario: linear declining threshold: start at genesis
	//add blocks with with the same ticket. Threshold should
	//decrease at a certain rate
	//scenario: N empty rounds should reduce the threshold
	//significantly.

	//can we keep the threshold stable without sometimes failing to
	//generate a block? Is this only a problem in small networks?

	//can the nr of votes indicate how many proposals it saw before
	//picking the top one? This is probably a nice indicator of the
	//current difficulty? adversary voters can influence this?
}

func TestVerify(t *testing.T) {
	//@TODO test all the verification edge cases
}
