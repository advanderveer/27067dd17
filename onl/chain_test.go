package onl_test

import (
	"errors"
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
		kv.Set([]byte{0x01}, []byte{0x02})
	})

	c1, gen1 := onl.NewChain(s1, w1, w2)
	g1 := c1.Genesis()
	test.Equals(t, gen1, g1.Hash())

	t.Run("read genesis state", func(t *testing.T) {
		st2, err := c1.State(gen1)
		test.Ok(t, err)

		st2.Read(func(kv *onl.KV) {
			test.Equals(t, []byte{0x02}, kv.Get([]byte{0x01}))
		})
	})

	t.Run("re-apply to second chain", func(t *testing.T) {
		c2, gen2 := onl.NewChain(s1)
		g2 := c2.Genesis()
		test.Equals(t, gen2, g1.Hash())

		test.Equals(t, uint64(0), g2.Round)
		test.Equals(t, []byte("vi veri veniversum vivus vici"), g2.Token)
		test.Equals(t, g2, g1)
		test.Equals(t, 1, len(g2.Writes))
	})
}

func TestChainAppendingAndWalking(t *testing.T) {
	s1, clean := onl.TempBadgerStore()
	defer clean()

	c1, g1 := onl.NewChain(s1)
	idn1 := onl.NewIdentity([]byte{0x01})

	b1 := idn1.Mint(testClock(1), g1, g1, 1)
	idn1.Sign(b1)

	test.Ok(t, c1.Append(b1))
	b2 := idn1.Mint(testClock(1), b1.Hash(), g1, 1)
	idn1.Sign(b2)

	test.Ok(t, c1.Append(b2))
	t.Run("should walk backwards in correct order", func(t *testing.T) {
		var seen []onl.ID
		test.Ok(t, c1.Walk(b2.Hash(), func(id onl.ID, b *onl.Block, stk *onl.Stakes) error {
			seen = append(seen, id)
			return nil
		}))

		test.Equals(t, []onl.ID{b2.Hash(), b1.Hash(), g1}, seen)
	})

	t.Run("should fail with block not exist", func(t *testing.T) {
		test.Equals(t, onl.ErrBlockNotExist, c1.Walk(bid2, func(id onl.ID, b *onl.Block, stk *onl.Stakes) error {
			return nil
		}))
	})

	t.Run("should pass on error in walk f", func(t *testing.T) {
		errt := errors.New("foo")
		test.Equals(t, errt, c1.Walk(g1, func(id onl.ID, b *onl.Block, stk *onl.Stakes) error {
			return errt
		}))
	})
}

func TestChainGenesisWitchStakeDeposit(t *testing.T) {
	s1, clean := onl.TempBadgerStore()
	defer clean()

	st1, _ := onl.NewState(nil)
	w1 := st1.Update(func(kv *onl.KV) {
		//write coinbase amount

		//try to read it right away

		//deposit currency as stake

		//read the deposity

	})

	c1, _ := onl.NewChain(s1, w1)

	_ = c1
}
