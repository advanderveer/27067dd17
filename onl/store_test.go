package onl_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

var _ onl.Store = &onl.BadgerStore{}

func TestBadgerReadWriteStore(t *testing.T) {
	s, clean := onl.TempBadgerStore()
	defer clean()

	idn1 := onl.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(1, bid1, bid2, 1)
	b2 := idn1.Mint(2, b1.Hash(), b1.Hash(), 2)

	tx := s.CreateTx(true)
	defer tx.Discard()

	t.Run("tip should be zero values", func(t *testing.T) {
		tip1, tipw1, err := tx.ReadTip()
		test.Ok(t, err)
		test.Equals(t, onl.NilID, tip1)
		test.Equals(t, uint64(0), tipw1)

		test.Ok(t, tx.WriteTip(bid1, 100))
		tip2, tipw2, err := tx.ReadTip()
		test.Ok(t, err)

		test.Equals(t, bid1, tip2)
		test.Equals(t, uint64(100), tipw2)
	})

	test.Equals(t, uint64(0), tx.MaxRound())

	test.Ok(t, tx.Write(b2, nil, big.NewInt(2)))
	test.Ok(t, tx.Write(b1, nil, big.NewInt(1)))
	test.Ok(t, tx.Commit())

	tx = s.CreateTx(false)
	defer tx.Discard()

	test.Equals(t, uint64(2), tx.MaxRound())
	b2, _, rank1, err := tx.Read(b1.Hash())
	test.Ok(t, err)
	test.Equals(t, b1, b2)
	test.Equals(t, "1", rank1.String())
	test.Ok(t, tx.Commit())

	t.Run("test block not exist", func(t *testing.T) {
		tx = s.CreateTx(false)
		defer tx.Discard()
		_, _, _, err = tx.Read(bid1)
		test.Equals(t, onl.ErrBlockNotExist, err)
	})

	t.Run("test round reading", func(t *testing.T) {
		tx = s.CreateTx(false)
		defer tx.Discard()

		test.Ok(t, tx.Round(1, func(id onl.ID, b *onl.Block, stk *onl.Stakes, rank *big.Int) error {
			test.Equals(t, b2.Hash(), id)
			test.Equals(t, b2, b)
			return nil
		}))

		err1 := errors.New("foo")
		test.Equals(t, err1, tx.Round(1, func(id onl.ID, b *onl.Block, stk *onl.Stakes, rank *big.Int) error {
			return err1
		}))
	})

}
