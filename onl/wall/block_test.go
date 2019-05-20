package wall

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
	"github.com/pkg/errors"
)

func TestVoteHashing(t *testing.T) {
	for i := 0; i < 10; i++ {
		v := &Vote{}
		test.Equals(t, "b3ab698298", fmt.Sprintf("%.5x", v.Hash()))
		v.Deposit[0] = 0x01
		test.Equals(t, "5be8ad242d", fmt.Sprintf("%.5x", v.Hash()))
		v.Prev[0] = 0x01
		test.Equals(t, "c4e7cb4f96", fmt.Sprintf("%.5x", v.Hash()))
		v.Proof[0] = 0x01
		test.Equals(t, "2066befc78", fmt.Sprintf("%.5x", v.Hash()))
		v.Round = 1
		test.Equals(t, "e417ed8db2", fmt.Sprintf("%.5x", v.Hash()))
		v.Signature[0] = 0x01
		test.Equals(t, "b88f12fad2", fmt.Sprintf("%.5x", v.Hash()))
		v.Timestamp = 1
		test.Equals(t, "79d08dbcb5", fmt.Sprintf("%.5x", v.Hash()))
		v.Voter[0] = 0x01
		test.Equals(t, "8ca8be7ab0", fmt.Sprintf("%.5x", v.Hash()))
	}
}

func TestBlockHashing(t *testing.T) {
	for i := 0; i < 10; i++ {
		b := &Block{}
		test.Equals(t, "3f71cc8f1a", fmt.Sprintf("%.5x", b.Hash()))
		b.ID[0] = 0x01
		test.Equals(t, "8d3ebf8d72", fmt.Sprintf("%.5x", b.Hash()))
		b.Proof[0] = 0x01
		test.Equals(t, "ff978aa54b", fmt.Sprintf("%.5x", b.Hash()))
		b.Vote.Round = 1
		test.Equals(t, "152dc46799", fmt.Sprintf("%.5x", b.Hash()))
		b.Witness = append(b.Witness, &Vote{})
		test.Equals(t, "682b9a98bb", fmt.Sprintf("%.5x", b.Hash()))
		b.Ticket.Proof[0] = 0x01
		test.Equals(t, "bdfe387a20", fmt.Sprintf("%.5x", b.Hash()))
		b.Ticket.Token[0] = 0x01
		test.Equals(t, "9e72d674a4", fmt.Sprintf("%.5x", b.Hash()))
		b.Transfers = append(b.Transfers, &Tr{})
		test.Equals(t, "da438a165f", fmt.Sprintf("%.5x", b.Hash()))
	}
}

func TestBlockSigning(t *testing.T) {
	b0 := &Block{} //empty block
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", b0.Vote.Voter.Bytes()))
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", b0.ID[:]))
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", b0.Proof[:]))

	id1 := NewIdentity([]byte{0x01}, rand.Reader)

	for i := 0; i < 10; i++ {
		b1 := id1.SignBlock(&Block{}, [vrf.Size]byte{})
		test.Equals(t, "4762ad6415", fmt.Sprintf("%.5x", b1.Vote.Voter.Bytes()))
		test.Equals(t, "3eaffa8e4f", fmt.Sprintf("%.5x", b1.Vote.Signature))
		test.Equals(t, "c011b985e8", fmt.Sprintf("%.5x", b1.Vote.Proof))
		test.Equals(t, "8aa0c60737", fmt.Sprintf("%.5x", b1.ID[:]))
		test.Equals(t, "9faf6eaaa0", fmt.Sprintf("%.5x", b1.Proof[:]))
		test.Equals(t, "f777111788", fmt.Sprintf("%.5x", b1.Ticket.Token[:]))
		test.Equals(t, "f4a0dca563", fmt.Sprintf("%.5x", b1.Ticket.Proof[:]))
	}
}

func TestVoteVerification(t *testing.T) {
	b := NewIdentity([]byte{0x01}, rand.Reader).SignBlock(&Block{}, [vrf.Size]byte{})
	test.Equals(t, "3eaffa8e4f", fmt.Sprintf("%.5x", b.Vote.Signature[:]))
	test.Equals(t, "c011b985e8", fmt.Sprintf("%.5x", b.Vote.Proof[:]))

	test.Equals(t, true, b.Vote.Verify())

	//test reset of block fields during verification
	test.Equals(t, "3eaffa8e4f", fmt.Sprintf("%.5x", b.Vote.Signature[:]))
	test.Equals(t, "c011b985e8", fmt.Sprintf("%.5x", b.Vote.Proof[:]))

	b.Vote.Round = 1 //imitate someone manipulating
	test.Equals(t, false, b.Vote.Verify())
}

func TestBlockIDVerification(t *testing.T) {
	b := NewIdentity([]byte{0x01}, rand.Reader).SignBlock(&Block{}, [vrf.Size]byte{})
	test.Equals(t, "8aa0c60737", fmt.Sprintf("%.5x", b.ID[:]))
	test.Equals(t, "9faf6eaaa0", fmt.Sprintf("%.5x", b.Proof[:]))

	test.Equals(t, true, b.verifyBlockID())

	//test reset of block fields during verification
	test.Equals(t, "8aa0c60737", fmt.Sprintf("%.5x", b.ID[:]))
	test.Equals(t, "9faf6eaaa0", fmt.Sprintf("%.5x", b.Proof[:]))

	b.ID[0] = 0x01 //imitate someone manipulating
	test.Equals(t, false, b.verifyBlockID())
}

func TestBlockVerification(t *testing.T) {
	params := DefaultParams()

	idn := NewIdentity([]byte{0x01}, rand.Reader)
	prevv := &Vote{}          //imaginary previous round block's vote
	prevt := [vrf.Size]byte{} //imaginary previous round block's ticket
	b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)

	bw := idn.SignBlock(&Block{Vote: Vote{Timestamp: 100, Round: 100}}, prevt)

	utro := NewUTRO()
	utro.Put(OID{}, TrOut{Amount: 100, UnlocksAfter: 10, Receiver: idn.PublicKey(), IsDeposit: true})

	ok, err := b.Verify(prevv, prevt, utro, params)
	test.Ok(t, err)
	test.Equals(t, true, ok)

	t.Run("threshold not reached", func(t *testing.T) {
		params2 := DefaultParams()
		params2.RoundCoefficient = apd.New(1, -3) //lower the chance of passing threshold

		ok, err = b.Verify(prevv, prevt, utro, params2)
		test.Equals(t, ErrBlocksTicketNotGoodEnough, err)
		test.Equals(t, false, ok)
	})

	// invalid vote
	b = idn.SignBlock(&Block{}, prevt)
	b.Vote.Proof[0] = 0x01
	ok, err = b.Verify(prevv, prevt, utro, params)
	test.Equals(t, ErrBlockVoteSignatureInvalid, err)
	test.Equals(t, false, ok)

	// invalid id
	b = idn.SignBlock(&Block{}, prevt)
	b.Proof[0] = 0x01
	ok, err = b.Verify(prevv, prevt, utro, params)
	test.Equals(t, ErrBlockIDInvalid, err)
	test.Equals(t, false, ok)

	// invalid ticket
	b = idn.SignBlock(&Block{}, prevt)
	b.Ticket.Token[0] = 0x01
	ok, err = b.Verify(prevv, prevt, utro, params)
	test.Equals(t, ErrBlockTicketInvalid, err)
	test.Equals(t, false, ok)

	// invalid witness
	b = idn.SignBlock(
		&Block{Witness: []*Vote{{}}}, prevt)
	ok, err = b.Verify(prevv, prevt, utro, params)
	test.Equals(t, ErrWitnessSignatureInvalid, err)
	test.Equals(t, false, ok)

	t.Run("witness in wrong round", func(t *testing.T) {
		b1 := idn.SignBlock(
			&Block{Vote: Vote{Timestamp: 1, Round: 2}, Witness: []*Vote{&bw.Vote}}, prevt)

		ok, err = b1.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrWitnessInvalidRound, err)
		test.Equals(t, false, ok)
	})

	t.Run("timestamp in past", func(t *testing.T) {
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 0, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockTimstampInPast, err)
		test.Equals(t, false, ok)
	})

	t.Run("round in the past", func(t *testing.T) {
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 0}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockRoundInPast, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit not Owned", func(t *testing.T) {
		utro.Put(OID{}, TrOut{IsDeposit: true, UnlocksAfter: 2})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockVoterDoesntOwnDeposit, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit must be locked", func(t *testing.T) {
		utro.Put(OID{}, TrOut{Receiver: idn.PublicKey(), UnlocksAfter: 0, IsDeposit: true})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockDepositNotLocked, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit must be locked", func(t *testing.T) {
		utro.Put(OID{}, TrOut{Receiver: idn.PublicKey(), UnlocksAfter: 1000, IsDeposit: true})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockDepositLockedTooLong, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit must be marked", func(t *testing.T) {
		utro.Put(OID{}, TrOut{Receiver: idn.PublicKey(), UnlocksAfter: 1000})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockDepositNotMarkedAsDeposit, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit not spendable", func(t *testing.T) {
		utro.Del(OID{})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrBlockDepositNotSpendable, err)
		test.Equals(t, false, ok)
	})

	t.Run("zero spendable deposit", func(t *testing.T) {
		utro.Put(OID{}, TrOut{Amount: 0, UnlocksAfter: 10, Receiver: idn.PublicKey(), IsDeposit: true})
		b := idn.SignBlock(&Block{Vote: Vote{Timestamp: 1, Round: 1}}, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrNoSpendableDepositAvailable, err)
		test.Equals(t, false, ok)
	})

	t.Run("invalid transaction", func(t *testing.T) {
		utro.Put(OID{}, TrOut{Amount: 100, UnlocksAfter: 10, Receiver: idn.PublicKey(), IsDeposit: true})
		b := &Block{Vote: Vote{Timestamp: 1, Round: 1}}
		b.Transfers = append(b.Transfers, &Tr{})

		b = idn.SignBlock(b, prevt)
		ok, err := b.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrTransferEmpty, errors.Cause(err))
		test.Equals(t, false, ok)
	})

	t.Run("witness timestamp", func(t *testing.T) {
		b1 := idn.SignBlock(
			&Block{Vote: Vote{Timestamp: 1, Round: 101}, Witness: []*Vote{&bw.Vote}}, prevt)

		prevv.Round = 100

		ok, err = b1.Verify(prevv, prevt, utro, params)
		test.Equals(t, ErrWitnessTimestampNotInPast, err)
		test.Equals(t, false, ok)
	})
}
