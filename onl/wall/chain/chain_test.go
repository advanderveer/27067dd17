package chain

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/onl/wall"
	"github.com/advanderveer/go-test"
	"github.com/pkg/errors"
)

func TestChainInitAndReading(t *testing.T) {
	params1 := wall.DefaultParams()

	for i := 0; i < 10; i++ {
		c1 := NewMemChain(params1)

		tipb, err := c1.Read(c1.Tip())
		test.Ok(t, err)
		test.Equals(t, "8cbc2c5213", fmt.Sprintf("%.5x", tipb.ID))

		genb, err := c1.Read(c1.Genesis())
		test.Ok(t, err)
		test.Equals(t, genb, tipb)
	}

	//changing the genesis ticket should radically change the genesis hash
	for i := 0; i < 10; i++ {
		params1.GenesisTicket[0] = 0x01
		c2 := NewMemChain(params1)
		genb, err := c2.Read(c2.Genesis())
		test.Ok(t, err)

		test.Equals(t, "d2cfb808c7", fmt.Sprintf("%.5x", genb.ID))
	}

	t.Run("read not exist", func(t *testing.T) {
		_, err := NewMemChain(params1).Read(wall.NilBID)
		test.Equals(t, ErrBlockNotExist, err)
	})
}

func TestChainWalkGenesis(t *testing.T) {
	p := wall.DefaultParams()
	c := NewMemChain(p)

	//we monkey patch the insertion of a block so we have some blocks to walk
	b0 := &wall.Block{}
	b0.Vote.Prev = c.Genesis()
	c.writeBlock(b0)

	//walk normally
	var visisted int
	test.Ok(t, c.Walk(b0.ID, true, func(b *wall.Block) (err error) {
		visisted++
		return
	}))

	test.Equals(t, 2, visisted)

	t.Run("start walk at non-existing block", func(t *testing.T) {
		bid := wall.NilBID
		bid[0] = 0x01
		err := c.Walk(bid, false, nil)
		test.Equals(t, ErrBlockNotExist, errors.Cause(err))
	})

	t.Run("walk should pass through the error", func(t *testing.T) {
		err1 := errors.New("foo")
		test.Equals(t, err1, c.Walk(c.Tip(), false, func(b *wall.Block) (err error) {
			return err1
		}))
	})
}

func TestDepositFinding(t *testing.T) {
	idn0 := wall.NewIdentity([]byte{0x02}, rand.Reader)
	idn := wall.NewIdentity([]byte{0x01}, rand.Reader)
	p := wall.DefaultParams()
	c := NewMemChain(p)

	//we monkey patch a block with
	b1 := &wall.Block{}
	b1.Vote.Prev = c.Genesis()
	b1.Vote.Round = 1
	b1.ID[0] = 0x01
	c.writeBlock(b1)

	//shouldn't find antything now
	tr0, i0 := c.findDeposit(b1.ID, 2, idn)
	test.Equals(t, (*wall.Tr)(nil), tr0)
	test.Equals(t, uint64(0), i0)

	//create valid deposit output
	outp1 := wall.TrOut{
		Receiver:     idn.PublicKey(),
		IsDeposit:    true,
		Amount:       12,
		UnlocksAfter: 250,
	}

	//invalid deposit output, not receiver
	outp2 := wall.TrOut{
		Receiver: idn0.PublicKey(),
		Amount:   13,
	}

	//invalid depost not usable
	outp3 := wall.TrOut{
		Receiver: idn.PublicKey(),
		Amount:   15,
	}

	b200 := &wall.Block{}
	b200.Vote.Prev = b1.ID
	b200.Vote.Round = 200
	b200.ID[0] = 0x02
	b200.Transfers = []*wall.Tr{
		&wall.Tr{Sender: idn0.PublicKey(), Outputs: []wall.TrOut{outp1}},
		&wall.Tr{Sender: idn.PublicKey(), Outputs: []wall.TrOut{outp2}},
		&wall.Tr{Sender: idn.PublicKey(), Outputs: []wall.TrOut{outp3}},

		//should find the following
		&wall.Tr{Sender: idn.PublicKey(), Outputs: []wall.TrOut{
			outp2,
			outp1,
		}},

		//should ignore this one as it takes the first it finds
		&wall.Tr{Sender: idn.PublicKey(), Outputs: []wall.TrOut{outp1}},
	}

	c.writeBlock(b200)

	tr1, i1 := c.findDeposit(b200.ID, 201, idn)
	test.Equals(t, b200.Transfers[3], tr1)
	test.Equals(t, uint64(1), i1)
}

func TestBlockMinting(t *testing.T) {
	params := wall.DefaultParams()
	idn := wall.NewIdentity([]byte{0x01}, rand.Reader)
	c := NewMemChain(params)

	t.Run("non existing block as mint", func(t *testing.T) {
		id := wall.NilBID
		id[0] = 0x01
		_, _, err := c.Mint(id, 1, 1, idn)
		test.Equals(t, ErrBlockNotExist, errors.Cause(err))
	})

	t.Run("no deposit", func(t *testing.T) {
		_, _, err := c.Mint(wall.NilBID, 1, 1, idn)
		test.Equals(t, ErrDepositNotFound, err)
	})

	//monkey patch a deposit into the genesis block
	deposit := &wall.Tr{
		Sender: idn.PublicKey(),
		Outputs: []wall.TrOut{{}, {
			Receiver:     idn.PublicKey(),
			IsDeposit:    true,
			Amount:       12,
			UnlocksAfter: 100,
		}},
	}
	deposit.ID[0] = 0x01
	c.blocks[c.Genesis()].Transfers = append(c.blocks[c.Genesis()].Transfers, deposit)

	// try to read a witness that was in a round but not in the blocks map
	t.Run("panic on witness corruption", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should panic on failure to read random bytes")
			}

			c.rounds[0] = c.rounds[0][:0]
		}()

		//monkey patch a block listed in a round that was not in block map
		bogusid := wall.BID{}
		bogusid[0] = 0x5
		c.rounds[0] = append(c.rounds[0], bogusid)

		c.Mint(wall.NilBID, 1, 1, idn)
	})

	// monkey patch a witness block
	wb := &wall.Block{Vote: wall.Vote{}}
	wb.Vote.Signature[0] = 0x03
	c.writeBlock(wb)

	//should now be able to mint consistently
	for i := 0; i < 10; i++ {
		b, signf, err := c.Mint(wall.NilBID, 1, 1, idn)
		test.Ok(t, err)

		//sign the block with identity it was minted for:
		signf()

		//should refer to the genesis stake deposit
		test.Equals(t, deposit.ID, b.Vote.Deposit.Tr())
		test.Equals(t, uint64(1), b.Vote.Deposit.Idx())
		test.Equals(t, c.Genesis(), b.Vote.Prev)
		test.Equals(t, 1, len(b.Witness))
		test.Equals(t, wb.Vote.Signature, b.Witness[0].Signature)
		test.Equals(t, "ef677ae374", fmt.Sprintf("%.5x", b.ID))
	}
}

func TestUTROIndexing(t *testing.T) {
	idn := wall.NewIdentity([]byte{0x01}, rand.Reader)
	p := wall.DefaultParams()
	c := NewMemChain(p)

	tr1 := wall.NewTr().Send(100, idn, 0, false).Sign(idn)
	tr2 := wall.NewTr().Consume(tr1, 0).Send(100, idn, 0, false).Sign(idn)

	c.blocks[c.Genesis()].Transfers = []*wall.Tr{tr1, tr2}

	utro, err := c.indexUTRO(c.Tip())
	test.Ok(t, err)

	_, ok := utro.Get(wall.Ref(tr1.ID, 0))
	test.Equals(t, false, ok) //should not be spendable
	_, ok = utro.Get(wall.Ref(tr2.ID, 0))
	test.Equals(t, true, ok) //should be spendable

	t.Run("non existing", func(t *testing.T) {
		_, err := c.indexUTRO(wall.BID{})
		test.Equals(t, ErrBlockNotExist, errors.Cause(err))
	})

}
