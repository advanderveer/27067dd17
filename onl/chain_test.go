package onl_test

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestChainCreationAndGenesis(t *testing.T) {
	s1, clean := onl.TempBadgerStore()
	defer clean()

	st1, err := onl.NewState(nil)
	test.Ok(t, err)
	w1 := st1.Update(func(*onl.KV) { /* empty should be ignored */ })
	w2 := st1.Update(func(kv *onl.KV) {
		kv.Tx.Set([]byte{0x01}, []byte{0x02})
	})

	c1, gen1, err := onl.NewChain(s1, w1, w2)
	test.Ok(t, err)

	g1 := c1.Genesis()
	test.Equals(t, gen1, g1.Hash())
	test.Equals(t, gen1, c1.Tip())

	t.Run("read genesis state", func(t *testing.T) {
		st2, err := c1.State(gen1)
		test.Ok(t, err)

		st2.Read(func(kv *onl.KV) {
			test.Equals(t, []byte{0x02}, kv.Tx.Get([]byte{0x01}))
		})
	})

	t.Run("re-apply to second chain", func(t *testing.T) {
		c2, gen2, err := onl.NewChain(s1)
		test.Ok(t, err)

		g2 := c2.Genesis()
		test.Equals(t, gen2, g1.Hash())

		test.Equals(t, uint64(0), g2.Round)
		test.Equals(t, []byte("vi veri veniversum vivus vici"), g2.Token)
		test.Equals(t, g2, g1)
		test.Equals(t, 1, len(g2.Writes))
	})
}

func TestChainAppendingAndWalking(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	s1, clean := onl.TempBadgerStore()
	defer clean()

	st1, err := onl.NewState(nil)
	test.Ok(t, err)

	c1, g1, err := onl.NewChain(s1, st1.Update(func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)             //mint 1 currency
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK()) //then deposit it
	}))
	test.Ok(t, err)

	b1 := idn1.Mint(testClock(1), g1, g1, 1)
	idn1.Sign(b1)

	test.Ok(t, c1.Append(b1))
	test.Equals(t, b1.Hash(), c1.Tip())

	b2 := idn1.Mint(testClock(2), b1.Hash(), g1, 2)
	idn1.Sign(b2)

	test.Ok(t, c1.Append(b2))
	test.Equals(t, b2.Hash(), c1.Tip())

	t.Run("should walk backwards in correct order", func(t *testing.T) {
		var seen []onl.ID
		test.Ok(t, c1.Walk(b2.Hash(), func(id onl.ID, b *onl.Block, stk *onl.Stakes, rank *big.Int) error {
			seen = append(seen, id)
			return nil
		}))

		test.Equals(t, []onl.ID{b2.Hash(), b1.Hash(), g1}, seen)
	})

	t.Run("should fail with block not exist", func(t *testing.T) {
		test.Equals(t, onl.ErrBlockNotExist, c1.Walk(bid2, func(id onl.ID, b *onl.Block, stk *onl.Stakes, rank *big.Int) error {
			return nil
		}))
	})

	t.Run("should pass on error in walk f", func(t *testing.T) {
		errt := errors.New("foo")
		test.Equals(t, errt, c1.Walk(g1, func(id onl.ID, b *onl.Block, stk *onl.Stakes, rank *big.Int) error {
			return errt
		}))
	})

	t.Run("reading non-existing should fail", func(t *testing.T) {
		_, _, err := c1.Read(bid4)
		test.Equals(t, onl.ErrBlockNotExist, err)
	})

}

func TestRoundWeigh(t *testing.T) {
	store, clean := onl.TempBadgerStore()
	defer clean()

	state, err := onl.NewState(nil)
	test.Ok(t, err)

	idn1 := onl.NewIdentity([]byte{0x01})
	write1 := state.Update(func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK())
	})

	idn2 := onl.NewIdentity([]byte{0x03})
	write2 := state.Update(func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn2.PK(), 1)
		kv.DepositStake(idn2.PK(), 1, idn2.TokenPK())
	})

	chain, gen, err := onl.NewChain(store, write1, write2)
	test.Ok(t, err)

	b0, w0, err := chain.Read(gen)
	test.Ok(t, err)
	test.Equals(t, uint64(1000), w0)
	test.Equals(t, b0, chain.Genesis())

	clock := onl.NewWallClock()
	b1 := idn1.Mint(clock, gen, gen, 1)
	idn1.Sign(b1)
	test.Ok(t, chain.Append(b1))

	b11, w1, err := chain.Read(b1.Hash())
	test.Ok(t, err)
	test.Equals(t, uint64(2000), w1)
	test.Equals(t, b1, b11)

	b2 := idn2.Mint(clock, gen, gen, 1)
	idn2.Sign(b2)
	test.Ok(t, chain.Append(b2))
	test.Equals(t, b2.Hash(), chain.Tip())

	//calling weigh shouldn't change anything
	test.Ok(t, chain.Weigh(0))

	//block 2 should have over taken the blocks 1 ranking
	b12, w2, err := chain.Read(b1.Hash())
	test.Ok(t, err)
	test.Equals(t, uint64(1500), w2)
	test.Equals(t, b1, b12)

	b22, w3, err := chain.Read(b2.Hash())
	test.Ok(t, err)
	test.Equals(t, uint64(2000), w3)
	test.Equals(t, b2, b22)

	test.Equals(t, b2.Hash(), chain.Tip())
}

func tallRound(height, width uint64, t *testing.T) {
	store, clean := onl.TempBadgerStore()
	defer clean()

	state, err := onl.NewState(nil)
	test.Ok(t, err)

	var idns []*onl.Identity
	write := state.Update(func(kv *onl.KV) {
		for i := uint64(0); i < height; i++ {
			idb := make([]byte, 8)
			binary.BigEndian.PutUint64(idb, i)

			idn := onl.NewIdentity(idb)
			kv.CoinbaseTransfer(idn.PK(), 1)            //mint 1 currency
			kv.DepositStake(idn.PK(), 1, idn.TokenPK()) //then deposit it

			idns = append(idns, idn)
		}
	})

	chain, gen, err := onl.NewChain(store, write)
	test.Ok(t, err)

	clock := onl.NewWallClock()
	for j := uint64(1); j <= width; j++ {
		tip := chain.Tip()

		for _, idn := range idns {
			b := idn.Mint(clock, tip, gen, j)
			idn.Sign(b)

			//@TODO add operations?

			test.Ok(t, chain.Append(b))
		}
	}
}

func TestWeigh1TallRound(t *testing.T) {
	tallRound(10, 10, t) //@TODO assert performance, or nr of operations
}
