package wall

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/onl/thr"
	"github.com/advanderveer/27067dd17/vrf"
)

// BID is the ID of a block
type BID [vrf.Size]byte

//NilBID is a zero value block ID
var NilBID = BID{}

// Vote encodes the voting aspect of a block proposal
type Vote struct {
	Round     uint64
	Prev      BID
	Voter     PK
	Deposit   OID
	Timestamp uint64
	Signature [vrf.Size]byte
	Proof     [vrf.ProofSize]byte
}

// Hash the vote structure
func (v *Vote) Hash() (h [32]byte) {
	var fields [][]byte
	round := make([]byte, 8)
	binary.BigEndian.PutUint64(round, v.Round)
	fields = append(fields, round)
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, v.Timestamp)
	fields = append(fields, timestamp)
	fields = append(fields, v.Prev[:])
	fields = append(fields, v.Voter[:])
	fields = append(fields, v.Deposit[:])
	fields = append(fields, v.Signature[:])
	fields = append(fields, v.Proof[:])
	return sha256.Sum256(bytes.Join(fields, nil))
}

// Block encompasses a vote for consensus itself and data to reach consensus over
type Block struct {
	ID    BID
	Proof [vrf.ProofSize]byte

	// the voting part of the block
	Vote Vote

	// witness encodes other votes the proposer saw in the previous round
	Witness []*Vote

	// verifiably random ticket draw
	Ticket struct {
		Proof [vrf.ProofSize]byte
		Token [vrf.Size]byte
	}

	// the currency transfers we're reaching  consensus over
	Transfers []*Tr
}

// Hash the block's content
func (b *Block) Hash() (h [32]byte) {
	var fields [][]byte

	//id& proof
	fields = append(fields, b.ID[:])
	fields = append(fields, b.Proof[:])

	//vote fields
	voteh := b.Vote.Hash()
	fields = append(fields, voteh[:])

	//witnesses
	for _, wv := range b.Witness {
		voteh := wv.Hash()
		fields = append(fields, voteh[:])
	}

	//ticket fields
	fields = append(fields, b.Ticket.Proof[:])
	fields = append(fields, b.Ticket.Token[:])

	//the transfers
	for _, tr := range b.Transfers {
		trh := tr.Hash()
		fields = append(fields, trh[:])
	}

	return sha256.Sum256(bytes.Join(fields, nil))
}

// Verify the vote's integrity
func (v *Vote) Verify() (ok bool) {
	sig := v.Signature
	proof := v.Proof
	v.Signature = [vrf.Size]byte{}
	v.Proof = [vrf.ProofSize]byte{}

	h := v.Hash()
	ok = vrf.Verify(v.Voter[:], h[:], sig[:], proof[:])
	if !ok {
		return false
	}

	v.Signature = sig
	v.Proof = proof
	return true
}

func (b *Block) verifyBlockID() (ok bool) {
	id := b.ID
	proof := b.Proof
	b.ID = [vrf.Size]byte{}
	b.Proof = [vrf.ProofSize]byte{}

	h := b.Hash()
	ok = vrf.Verify(b.Vote.Voter[:], h[:], id[:], proof[:])
	if !ok {
		return false
	}

	b.ID = id
	b.Proof = proof
	return true
}

func (b *Block) verifyTicket(prevt [vrf.Size]byte) (ok bool) {
	ok = vrf.Verify(b.Vote.Voter[:], prevt[:], b.Ticket.Token[:], b.Ticket.Proof[:])
	return
}

// VerifyCheap is the verification of fields that are not dependant on the current
// system state can can be done quickly and cheaply. It takes inspiration from the Bitcoin protocol rules:
// https://en.bitcoin.it/wiki/Protocol_rules#.22block.22_messages
func (b *Block) VerifyCheap() (ok bool, err error) {
	if !b.verifyBlockID() {
		return false, ErrBlockIDInvalid
	}

	if !b.Vote.Verify() {
		return false, ErrBlockVoteSignatureInvalid
	}

	//round must be at least one
	if b.Vote.Round < 1 {
		return false, ErrBlockRoundIsZero
	}

	return true, nil
}

// VerifyAgainstPrev are semi-cheap verification steps that only require the previous
// block to be loaded from the state. It takes inspiration from the Bitcoin protocol rules:
// https://en.bitcoin.it/wiki/Protocol_rules#.22block.22_messages
func (b *Block) VerifyAgainstPrev(prevv *Vote, prevt [vrf.Size]byte) (ok bool, err error) {
	if !b.verifyTicket(prevt) {
		return false, ErrBlockTicketInvalid
	}

	// round nr must make sense
	if b.Vote.Round <= prevv.Round {
		return false, ErrBlockRoundInPast
	}

	// make sure the timestamp makes sense
	if b.Vote.Timestamp <= prevv.Timestamp {
		return false, ErrBlockTimstampInPast
	}

	// verify witnesses next the prev block
	for _, w := range b.Witness {
		if !w.Verify() {
			return false, ErrWitnessSignatureInvalid
		}

		//the witness round must always equal the prev's block round
		if w.Round != prevv.Round {
			return false, ErrWitnessInvalidRound
		}

		//the witness timestamp must be before this blocks creation timestamp
		if w.Timestamp >= b.Vote.Timestamp {
			return false, ErrWitnessTimestampNotInPast
		}

		if w.Signature == prevv.Signature {
			return false, ErrWitnessWasPrevBlock
		}
	}

	return true, nil
}

// VerifyAgainstUTRO are the verification steps that require a fully loaded
// index of unspend transfer outputs.  It takes inspiration from the Bitcoin protocol rules:
// https://en.bitcoin.it/wiki/Protocol_rules#.22block.22_messages
func (b *Block) VerifyAgainstUTRO(utro *UTRO, p *Params) (ok bool, err error) {

	//deposit must be a spendable output
	deposit, ok := utro.Get(b.Vote.Deposit)
	if !ok {
		return false, ErrBlockDepositNotSpendable
	}

	// check if the deposit is usable for the block's round
	if ok, err := deposit.UsableDepositFor(b.Vote.Round, p.MaxDepositTTL); !ok {
		return false, err
	}

	//the deposit must be owned by the voter
	if deposit.Receiver != b.Vote.Voter {
		return false, ErrBlockVoterDoesntOwnDeposit
	}

	//read the total amount of deposited stake
	total := utro.Deposited(b.Vote.Round, p.MaxDepositTTL)
	if total < 1 {
		return false, ErrNoSpendableDepositAvailable
	}

	//check if the drawn ticket passes the threshold
	_, _, ok = thr.Thr(p.DecimalContext, p.RoundCoefficient, deposit.Amount, total, b.Ticket.Token[:])
	if !ok {
		return false, ErrBlocksTicketNotGoodEnough
	}

	//verify transfers
	for _, tr := range b.Transfers {
		ok, err := tr.Verify(false, b.Vote.Round, utro, p.MaxDepositTTL)
		if !ok {
			return false, err
		}
	}

	return true, nil
}
