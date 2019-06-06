package bchain

import (
	"fmt"
	"math"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestBlockID(t *testing.T) {
	bid1 := NewBID(1)
	bid1[8] = 0xff
	test.Equals(t, 40, len(bid1.Bytes()))
	test.Equals(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0xff, 0x0}, bid1.Bytes()[:10])
	test.Equals(t, uint64(1), bid1.Round())
	test.Equals(t, "1-ff000000", bid1.String())

	test.Equals(t, "18446744073709551615-00000000", NewBID(math.MaxUint64).String())
}

func TestBlockHashing(t *testing.T) {
	for i := 0; i < 10; i++ {
		b := &Block{}
		test.Equals(t, "52a3e0804d", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.ID[0] = 0x01
		test.Equals(t, "16101a5e2a", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.ID[39] = 0x01
		test.Equals(t, "50cab87bbf", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.Prev[0] = 0x01
		test.Equals(t, "11d5827691", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.Proposer[0] = 0x01
		test.Equals(t, "2a8d422dd0", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.Proof[0] = 0x01
		test.Equals(t, "495d5d64fe", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.Ticket.Proof[0] = 0x01
		test.Equals(t, "ab9dcec53a", fmt.Sprintf("%.5x", b.Hash()))
		b.Header.Ticket.Token[0] = 0x01
		test.Equals(t, "ebd4751ee2", fmt.Sprintf("%.5x", b.Hash()))

		b.Data = []byte{0x01}
		test.Equals(t, "9862d244c9", fmt.Sprintf("%.5x", b.Hash()))
	}
}

func TestBlockSigning(t *testing.T) {
	id1 := NewIdentity([]byte{0x01}, nil)
	b1 := &Block{}
	var prevt [32]byte
	test.Equals(t, byte(0x00), b1.Header.Ticket.Token[0])
	test.Equals(t, byte(0x00), b1.Header.Ticket.Proof[0])

	b2 := &Block{}
	for i := 0; i < 10; i++ {
		b2 = id1.SignBlock(1, b1, prevt)
		test.Equals(t, "1-253e01e7", b2.Header.ID.String())
		test.Equals(t, uint64(1), b2.Round())
	}

	test.Equals(t, b2, b1)
	test.Equals(t, byte(0xf7), b1.Header.Ticket.Token[0])
	test.Equals(t, byte(0xf4), b1.Header.Ticket.Proof[0])
	test.Equals(t, true, b2.CheckSignature())

	//changing the round should invalidate signature
	id1.SignBlock(1, b1, prevt).Header.ID[0] = 0x01
	test.Equals(t, false, b2.CheckSignature())

	//changing signature should invalidate
	id1.SignBlock(1, b1, prevt).Header.ID[39] = 0x01
	test.Equals(t, false, b2.CheckSignature())

	//changing proof should invalidate
	id1.SignBlock(1, b1, prevt).Header.Proof[0] = 0x01
	test.Equals(t, false, b2.CheckSignature())
}

func TestBlockPrevChecking(t *testing.T) {
	id1 := NewIdentity([]byte{0x01}, nil)
	b1 := id1.SignBlock(0, NewBlock(nil), [32]byte{}) //zero block
	test.Equals(t, true, b1.CheckSignature())

	b2 := id1.SignBlock(b1.Round()+1, NewBlock(&b1.Header), b1.Token()) //linked
	test.Equals(t, true, b2.CheckSignature())
	test.OkEquals(t, true)(b2.CheckAgainstPrev(&b1.Header))

	t.Run("ticket invalid", func(t *testing.T) {
		b1.Header.Ticket.Token[0] = 0x01
		ok, err := b2.CheckAgainstPrev(&b1.Header)
		test.Equals(t, ErrBlockTicketInvalid, err)
		test.Equals(t, false, ok)
	})

	t.Run("round invalid", func(t *testing.T) {
		b1 := id1.SignBlock(0, NewBlock(nil), [32]byte{})
		b2 := id1.SignBlock(0, NewBlock(&b1.Header), b1.Token())
		ok, err := b2.CheckAgainstPrev(&b1.Header)
		test.Equals(t, ErrBlockRoundInvalid, err)
		test.Equals(t, false, ok)
	})
}
