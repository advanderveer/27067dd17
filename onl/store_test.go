package onl_test

import (
	"errors"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

var _ onl.Store = &onl.BadgerStore{}

func TestBadgerReadWriteStore(t *testing.T) {
	s, clean := onl.TempBadgerStore()
	defer clean()

	idn1 := onl.NewIdentity([]byte{0x01})
	b1 := idn1.Mint(testClock(1), bid1, bid2, 1)

	tx := s.CreateTx(true)
	defer tx.Discard()
	err := tx.Write(b1, nil)
	test.Ok(t, err)
	test.Ok(t, tx.Commit())

	tx = s.CreateTx(false)
	defer tx.Discard()

	b2, _, err := tx.Read(b1.Hash())
	test.Ok(t, err)
	test.Equals(t, b1, b2)
	test.Ok(t, tx.Commit())

	t.Run("test block not exist", func(t *testing.T) {
		tx = s.CreateTx(false)
		defer tx.Discard()
		_, _, err = tx.Read(bid1)
		test.Equals(t, onl.ErrBlockNotExist, err)
	})

	t.Run("test round reading", func(t *testing.T) {
		tx = s.CreateTx(false)
		defer tx.Discard()

		test.Ok(t, tx.Round(1, func(b *onl.Block, stk *onl.Stakes) error {
			test.Equals(t, b2, b)
			return nil
		}))

		err1 := errors.New("foo")
		test.Equals(t, err1, tx.Round(1, func(b *onl.Block, stk *onl.Stakes) error {
			return err1
		}))
	})

}
