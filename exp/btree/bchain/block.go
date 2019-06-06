package bchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/advanderveer/27067dd17/vrf"
)

// BID is a the block id, prefixed with the round nr such that it can be easily
// searched and iterated by the round nr
type BID [sha256.Size + 8]byte

//NilBID is a zero value block ID
var NilBID = BID{}

// NewBID createas a block id for a certain round
func NewBID(round uint64) (bid BID) {
	WriteRound(round, bid[:])
	return
}

// WriteRound to a byte slice
func WriteRound(round uint64, prefix []byte) {
	binary.BigEndian.PutUint64(prefix[:], round)
}

// ReadRound from a byte slice
func ReadRound(prefix []byte) uint64 {
	return binary.BigEndian.Uint64(prefix[:8])
}

// Bytes returns the id as a variable-sized byte slice
func (bid BID) Bytes() []byte { return bid[:] }

// Round returns the round part of the block id
func (bid BID) Round() uint64  { return ReadRound(bid[:8]) }
func (bid BID) String() string { return fmt.Sprintf("%d-%.4x", bid.Round(), bid[8:]) }

// Block is stored in the chain structure
type Block struct {
	Header BlockHdr

	//@TODO add writes, as a transparent log?
	//@TODO separate reads from writes?
	Data []byte
}

// Token determines the verifiable random round rank
type Token [vrf.Size]byte

// BlockHdr represents the small fixed size portion of the block that can be
// loaded into memory without much trouble
type BlockHdr struct {

	// A unique and provably random signature of the whole block prefixed with
	// that includes the round of the block
	ID BID

	// The previous block in the chain
	Prev BID

	// Proposer of the block
	Proposer PK

	// The proof that the ID is indeed random and is used to verify the ID
	Proof [vrf.ProofSize]byte

	// verifiably random ticket draw that will be used to pick the highest ranking
	// block for a certain round
	Ticket struct {
		Proof [vrf.ProofSize]byte
		Token Token
	}
}

// NewBlock creates an empty block with the provided prev
func NewBlock(prevh *BlockHdr) (b *Block) {
	b = &Block{}
	if prevh != nil {
		b.Header.Prev = prevh.ID
	}

	return
}

// ID returns the ID
func (b *Block) ID() BID { return b.Header.ID }

// Round returns the round
func (b *Block) Round() uint64 { return b.Header.ID.Round() }

// Token returns the ticket token
func (b *Block) Token() [vrf.Size]byte { return b.Header.Ticket.Token }

// Hash the block
func (b *Block) Hash() (h [sha256.Size]byte) {
	var fields [][]byte
	fields = append(fields, b.Header.ID.Bytes())
	fields = append(fields, b.Header.Prev.Bytes())
	fields = append(fields, b.Header.Proposer[:])
	fields = append(fields, b.Header.Proof[:])
	fields = append(fields, b.Header.Ticket.Proof[:])
	fields = append(fields, b.Header.Ticket.Token[:])

	fields = append(fields, b.Data) //@TOOD ok to load this all into memory at once?
	return sha256.Sum256(bytes.Join(fields, nil))
}

// CheckSignature performs cheap verification of the block signature
func (b *Block) CheckSignature() (ok bool) {
	id := b.Header.ID
	proof := b.Header.Proof
	b.Header.ID = NewBID(id.Round())
	b.Header.Proof = [vrf.ProofSize]byte{}

	h := b.Hash()
	ok = vrf.Verify(b.Header.Proposer[:], h[:], id[8:], proof[:])
	if !ok {
		return false
	}

	b.Header.ID = id
	b.Header.Proof = proof
	return true
}

// CheckAgainstPrev checks the validity of the block considering the prev exists
// and could be loaded
func (b *Block) CheckAgainstPrev(prev *BlockHdr) (ok bool, err error) {
	ok = vrf.Verify(b.Header.Proposer[:], prev.Ticket.Token[:], b.Header.Ticket.Token[:], b.Header.Ticket.Proof[:])
	if !ok {
		return false, ErrBlockTicketInvalid
	}

	//this block's round must be after the prev's round
	if b.Header.ID.Round() <= prev.ID.Round() {
		return false, ErrBlockRoundInvalid
	}

	return
}
