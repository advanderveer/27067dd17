package chain

import (
	"crypto/rand"
	"fmt"
	"strings"
	"testing"

	"github.com/advanderveer/27067dd17/onl/wall"
	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
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
				t.Errorf("should panic")
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

	utro := c.indexUTRO(c.Tip())

	_, ok := utro.Get(wall.Ref(tr1.ID, 0))
	test.Equals(t, false, ok) //should not be spendable
	_, ok = utro.Get(wall.Ref(tr2.ID, 0))
	test.Equals(t, true, ok) //should be spendable
}

func TestBlockAppending(t *testing.T) {
	idn := wall.NewIdentity([]byte{0x01}, rand.Reader)
	p := wall.DefaultParams()

	t.Run("cheap verification error", func(t *testing.T) {
		c := NewMemChain(p)
		b := idn.SignBlock(&wall.Block{}, [32]byte{})
		err := c.Append(b)
		test.Equals(t, wall.ErrBlockRoundIsZero, errors.Cause(err))
	})

	t.Run("block already exists", func(t *testing.T) {
		c := NewMemChain(p)
		b := idn.SignBlock(&wall.Block{Vote: wall.Vote{Round: 1}}, [32]byte{})
		c.writeBlock(b)

		err := c.Append(b)
		test.Equals(t, ErrBlockExists, errors.Cause(err))
	})

	t.Run("prev doesn't exist", func(t *testing.T) {
		c := NewMemChain(p)
		b := idn.SignBlock(&wall.Block{Vote: wall.Vote{Round: 1}}, [32]byte{})

		err := c.Append(b)
		test.Equals(t, true, strings.Contains(err.Error(), "prev"))
		test.Equals(t, ErrBlockNotExist, errors.Cause(err))
	})

	t.Run("prev verification", func(t *testing.T) {
		c := NewMemChain(p)
		prev, _ := c.Read(c.Tip())
		b := idn.SignBlock(&wall.Block{Vote: wall.Vote{Prev: c.Tip(), Round: 1}}, prev.Ticket.Token)

		err := c.Append(b)
		test.Equals(t, wall.ErrBlockTimstampInPast, errors.Cause(err))
	})

	t.Run("multiple votes", func(t *testing.T) {
		c := NewMemChain(p)
		tr1 := wall.NewTr().Send(100, idn, 10, true).Sign(idn)
		c.blocks[c.Genesis()].Transfers = []*wall.Tr{tr1}

		prev, _ := c.Read(c.Tip())
		b1 := idn.SignBlock(&wall.Block{Vote: wall.Vote{Deposit: wall.Ref(tr1.ID, 0), Prev: c.Tip(), Round: 1, Timestamp: 1}}, prev.Ticket.Token)
		b2 := idn.SignBlock(&wall.Block{Vote: wall.Vote{Deposit: wall.Ref(tr1.ID, 0), Prev: c.Tip(), Round: 1, Timestamp: 2}}, prev.Ticket.Token)

		err := c.Append(b1)
		test.Ok(t, err)

		err = c.Append(b2)
		test.Equals(t, ErrVoterAlreadyVoted, errors.Cause(err))
	})

	t.Run("utro verification", func(t *testing.T) {
		c := NewMemChain(p)
		tr1 := wall.NewTr().Send(100, idn, 10, true).Sign(idn)
		tr2 := wall.NewTr().Consume(tr1, 0).Send(100, idn, 0, false).Sign(idn)
		c.blocks[c.Genesis()].Transfers = []*wall.Tr{tr1, tr2}

		prev, _ := c.Read(c.Tip())
		b1 := idn.SignBlock(&wall.Block{Vote: wall.Vote{Deposit: wall.Ref(tr1.ID, 0), Prev: c.Tip(), Round: 1, Timestamp: 1}}, prev.Ticket.Token)

		err := c.Append(b1)
		test.Equals(t, wall.ErrBlockDepositNotSpendable, errors.Cause(err))
	})
}

func TestMintAndAppend(t *testing.T) {
	idn := wall.NewIdentity([]byte{0x01}, rand.Reader)
	p := wall.DefaultParams()

	//chain with injected deposit
	c := NewMemChain(p)
	tr1 := wall.NewTr().Send(100, idn, 10, true).Sign(idn)
	c.blocks[c.Genesis()].Transfers = []*wall.Tr{tr1}

	//mint a block
	_, signed, err := c.Mint(c.Tip(), 1, 1, idn)
	test.Ok(t, err)

	//which should append succesfully
	err = c.Append(signed())
	test.Ok(t, err)
}

func TestLongestChain(t *testing.T) {
	idn1 := wall.NewIdentity([]byte{0x01}, rand.Reader)
	idn2 := wall.NewIdentity([]byte{0x02}, rand.Reader)
	p := wall.DefaultParams()

	c := NewMemChain(p)
	prev, _ := c.Read(c.Tip())
	b1 := idn1.SignBlock(&wall.Block{Vote: wall.Vote{Prev: c.Tip(), Round: 1, Timestamp: 1}}, prev.Ticket.Token)
	c.writeBlock(b1)

	b2 := idn2.SignBlock(&wall.Block{Vote: wall.Vote{Prev: b1.ID, Round: 3, Timestamp: 1}}, prev.Ticket.Token)
	c.writeBlock(b2)
	b3 := idn1.SignBlock(&wall.Block{Vote: wall.Vote{Prev: b1.ID, Round: 3, Timestamp: 2}}, prev.Ticket.Token)
	c.writeBlock(b3)

	test.Equals(t, b2.ID, c.rounds[3][0])
	test.Equals(t, b3.ID, c.rounds[3][1])

	c.selectLongestChain(0)

	//should have re-sorted blocks in the round
	test.Equals(t, b2.ID, c.rounds[3][1])
	test.Equals(t, b3.ID, c.rounds[3][0])

	// weight should match what was expected
	test.Assert(t, c.weights[b3.ID] > c.weights[b2.ID], "b3 should weigh more")
	test.Equals(t, p.WeightPoints*3, c.weights[b3.ID])                    //rank 1 on all three rounds
	test.Equals(t, (p.WeightPoints*2)+p.WeightPoints/2, c.weights[b2.ID]) //ranks second on round 3

	// block three should now be tip
	test.Equals(t, b3.ID, c.Tip())

	t.Run("panic on round block not existing", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should panic")
			}

			c.blocks[b2.ID] = b2
		}()

		delete(c.blocks, b2.ID)
		c.selectLongestChain(0)
	})

	t.Run("panic on block weight not existing", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should panic")
			}
		}()

		delete(c.weights, c.Genesis())
		c.selectLongestChain(1)
	})
}

func TestLinearBlockMintingAppending(t *testing.T) {
	for _, c := range []struct {
		Name             string
		RoundCoefficient *apd.Decimal
		ExpEmptyRounds   int
		DepositTTL       uint64
	}{
		{Name: "high chance on empty round", DepositTTL: 75, RoundCoefficient: apd.New(999, -3), ExpEmptyRounds: 34},
		{Name: "low chance o empty round", DepositTTL: 75, RoundCoefficient: apd.New(99999, -5), ExpEmptyRounds: 0},
		{Name: "reduced deposit ttl", DepositTTL: 50, RoundCoefficient: apd.New(999, -3), ExpEmptyRounds: 9},
	} {

		t.Run(c.Name, func(t *testing.T) {
			idn := wall.NewIdentity([]byte{0x02}, rand.Reader)
			p := wall.DefaultParams()
			p.RoundCoefficient = c.RoundCoefficient

			chain := NewMemChain(p)
			tr1 := wall.NewTr().Send(100, idn, c.DepositTTL, true).Sign(idn)
			chain.blocks[chain.Genesis()].Transfers = []*wall.Tr{tr1}

			var emptyRounds int
			var successRounds int
			for i := uint64(0); i < 100; i++ {

				//mint a block
				_, signed, err := chain.Mint(chain.Tip(), i+1, i+1, idn)
				if errors.Cause(err) == ErrDepositNotFound {
					break //deposit expired
				}

				b := signed()

				//which should append succesfully
				err = chain.Append(b)
				if errors.Cause(err) == wall.ErrBlocksTicketNotGoodEnough {
					emptyRounds++
					continue
				}

				//should always be new tip if it has just 1 tip per round
				test.Equals(t, b.ID, chain.Tip())

				test.Ok(t, err)
				successRounds++
			}

			//amount of rounds that we reached should always be one less then deposit allows
			test.Equals(t, uint64(emptyRounds+successRounds), tr1.Outputs[0].UnlocksAfter-1)

			//should have some empty rounds also
			test.Equals(t, c.ExpEmptyRounds, emptyRounds)
		})
	}
}
