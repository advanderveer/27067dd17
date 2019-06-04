package wall

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
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

func TestCheapBlockVerification(t *testing.T) {
	idn := NewIdentity([]byte{0x01}, rand.Reader)
	prevt := [vrf.Size]byte{} //imaginary previous round block's ticket

	t.Run("block id/signature", func(t *testing.T) {
		b := idn.SignBlock(&Block{}, prevt)
		b.Vote.Round = 1

		ok, err := b.VerifyCheap()
		test.Equals(t, ErrBlockIDInvalid, err)
		test.Equals(t, false, ok)
	})

	t.Run("block vote signature", func(t *testing.T) {
		b := &Block{}
		idn.signVote(b)
		b.Vote.Round = 1
		idn.SignBlock(b, prevt)

		ok, err := b.VerifyCheap()
		test.Equals(t, ErrBlockVoteSignatureInvalid, err)
		test.Equals(t, false, ok)
	})

	t.Run("block round non-zero", func(t *testing.T) {
		b := idn.SignBlock(&Block{}, prevt)
		ok, err := b.VerifyCheap()

		test.Equals(t, ErrBlockRoundIsZero, err)
		test.Equals(t, false, ok)
	})
}

func TestPrevBlockVerification(t *testing.T) {
	idn := NewIdentity([]byte{0x01}, rand.Reader)
	prevt := [vrf.Size]byte{} //imaginary previous round block's ticket

	t.Run("invalid ticket", func(t *testing.T) {
		prevv := &Vote{}
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}
		idn.drawTicket(b, prevt)
		b.Ticket.Token[0] = 0x01
		ok, err := b.VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrBlockTicketInvalid, err)
		test.Equals(t, false, ok)
	})

	t.Run("round must be larger then prev round", func(t *testing.T) {
		prevv := &Vote{Round: 2}
		b := idn.SignBlock(&Block{Vote: Vote{Round: 1, Timestamp: 1}}, prevt)

		ok, err := b.VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrBlockRoundInPast, err)
		test.Equals(t, false, ok)
	})

	t.Run("timestamp must be larger then prev timestamp", func(t *testing.T) {
		prevv := &Vote{Round: 0, Timestamp: 2}
		b := idn.SignBlock(&Block{Vote: Vote{Round: 1, Timestamp: 1}}, prevt)

		ok, err := b.VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrBlockTimstampInPast, err)
		test.Equals(t, false, ok)
	})

	t.Run("witness signature is invalid", func(t *testing.T) {
		prevv := &Vote{Round: 0, Timestamp: 0}
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}
		b.Witness = append(b.Witness, &Vote{})

		ok, err := idn.SignBlock(b, prevt).VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrWitnessSignatureInvalid, err)
		test.Equals(t, false, ok)
	})

	t.Run("witness round doesn't match prev round", func(t *testing.T) {
		prevv := &Vote{Round: 0, Timestamp: 0}
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}
		wb1 := idn.SignBlock(&Block{Vote: Vote{Round: 2}}, prevt)

		b.Witness = append(b.Witness, &wb1.Vote)

		ok, err := idn.SignBlock(b, prevt).VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrWitnessInvalidRound, err)
		test.Equals(t, false, ok)
	})

	t.Run("witness timestamp", func(t *testing.T) {
		prevv := &Vote{Round: 0, Timestamp: 0}
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}
		wb1 := idn.SignBlock(&Block{Vote: Vote{Round: 0, Timestamp: 2}}, prevt)

		b.Witness = append(b.Witness, &wb1.Vote)

		ok, err := idn.SignBlock(b, prevt).VerifyAgainstPrev(prevv, prevt)
		test.Equals(t, ErrWitnessTimestampNotInPast, err)
		test.Equals(t, false, ok)
	})

	t.Run("witness is prev block", func(t *testing.T) {
		prevb := idn.SignBlock(&Block{Vote: Vote{}}, prevt)
		b := &Block{Vote: Vote{Prev: prevb.ID, Round: 1, Timestamp: 1}}
		b.Witness = append(b.Witness, &prevb.Vote)

		ok, err := idn.SignBlock(b, prevb.Ticket.Token).VerifyAgainstPrev(&prevb.Vote, prevb.Ticket.Token)
		test.Equals(t, ErrWitnessWasPrevBlock, err)
		test.Equals(t, false, ok)
	})
}

func TestUTROVerification(t *testing.T) {
	idn := NewIdentity([]byte{0x01}, rand.Reader)

	t.Run("deposit is unspendable", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrBlockDepositNotSpendable, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit is not marked as deposit", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		utro.Put(OID{}, TrOut{})
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrBlockDepositNotMarkedAsDeposit, err)
		test.Equals(t, false, ok)
	})

	t.Run("deposit is not owned by voter", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		utro.Put(OID{}, TrOut{IsDeposit: true, UnlocksAfter: 10})
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrBlockVoterDoesntOwnDeposit, err)
		test.Equals(t, false, ok)
	})

	t.Run("no spendable deposit", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		utro.Put(OID{}, TrOut{IsDeposit: true, UnlocksAfter: 2, Receiver: idn.PublicKey()})
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrNoSpendableDepositAvailable, err)
		test.Equals(t, false, ok)
	})

	t.Run("no spendable deposit", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		params.RoundCoefficient = apd.New(1, -3)
		utro.Put(OID{}, TrOut{Amount: 100, IsDeposit: true, UnlocksAfter: 2, Receiver: idn.PublicKey()})
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrBlocksTicketNotGoodEnough, err)
		test.Equals(t, false, ok)
	})

	t.Run("no spendable deposit", func(t *testing.T) {
		utro := NewUTRO()
		params := DefaultParams()
		utro.Put(OID{}, TrOut{Amount: 100, IsDeposit: true, UnlocksAfter: 2, Receiver: idn.PublicKey()})
		b := &Block{Vote: Vote{Round: 1, Timestamp: 1}}
		b.Transfers = append(b.Transfers, &Tr{})

		ok, err := idn.SignBlock(b, [vrf.Size]byte{}).VerifyAgainstUTRO(utro, params)
		test.Equals(t, ErrTransferEmpty, err)
		test.Equals(t, false, ok)
	})
}

func TestValidBlock(t *testing.T) {
	idn := NewIdentity([]byte{0x01}, rand.Reader)
	utro := NewUTRO()
	utro.Put(OID{}, TrOut{Amount: 100, IsDeposit: true, UnlocksAfter: 2, Receiver: idn.PublicKey()})
	params := DefaultParams()
	prevb := idn.SignBlock(&Block{Vote: Vote{}}, [vrf.Size]byte{})
	b := idn.SignBlock(&Block{Vote: Vote{Prev: prevb.ID, Round: 1, Timestamp: 1}}, prevb.Ticket.Token)

	test.OkEquals(t, true)(b.VerifyCheap())
	test.OkEquals(t, true)(b.VerifyAgainstPrev(&prevb.Vote, prevb.Ticket.Token))
	test.OkEquals(t, true)(b.VerifyAgainstUTRO(utro, params))
}
