package slot

import (
	"math/big"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

// BlockReader is an interface for reading blocks
type BlockReader interface {
	Read(id ID) (b *Block)
}

// Voter keeps state for block notarization of a certain round
type Voter struct {
	round  uint64
	reader BlockReader
	votes  map[ID]*Block
	ticket Ticket
	pk     []byte
	mu     sync.Mutex
}

// NewVoter creates a block voter
func NewVoter(round uint64, r BlockReader, t Ticket, pk []byte) (v *Voter) {
	v = &Voter{
		round:  round,
		reader: r,
		votes:  make(map[ID]*Block),
		ticket: t,
		pk:     pk,
	}

	return
}

// Verify the block according to notarization rules
func (v *Voter) Verify(b *Block) (ok bool, err error) {
	if b.Round != v.round {
		return false, ErrWrongRound
	}

	//@TODO verify that the picked prev is not too far in the past, this
	//could result in an artifical low threshold that the user may force itself
	//into for the current round? i.e: create a block for this round based on the
	//genesis block. All open blocks in between cause will cause the threshold to
	//be very low

	prevb := v.reader.Read(b.Prev)
	if prevb == nil {
		return false, ErrPrevNotExist
	}

	if !vrf.Verify(b.PK[:], Seed(prevb, b.Round), b.Ticket[:], b.Proof[:]) {
		return false, ErrProposeProof
	}

	return true, nil
}

// Propose will ask the notary to consider this block to vote on. returns whether
// this block is (one of) the highest scoring in this round and the new number
// of highest blocks
func (v *Voter) Propose(b *Block) (ok bool, nh int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	id := b.Hash()
	if len(v.votes) < 1 {
		v.votes[id] = b
		return true, 1
	}

	curr := new(big.Rat)
	for _, b := range v.votes {
		curr.Set(b.Strength(1))
		break
	}

	new := b.Strength(1)
	if new.Cmp(curr) > 0 { //nuke proposal, new block is bigger
		v.votes = make(map[ID]*Block)
		v.votes[id] = b
		return true, 1
	}

	if new.Cmp(curr) == 0 { //add to proposal, block is as big
		_, ok := v.votes[id]
		v.votes[id] = b
		return !ok, len(v.votes)
	}

	return false, len(v.votes)
}

// Vote will return the highest scoring blocks the voter it has seen for this
// round with a proof that it was allowed to cast a vote.
func (v *Voter) Vote() (votes []*Block) {
	v.mu.Lock()
	for _, b := range v.votes {
		votes = append(votes, b)
	}

	v.votes = make(map[ID]*Block)
	v.mu.Unlock()

	for _, b := range votes {
		num := copy(b.VoteTicket[:], v.ticket.Data)
		if num != TicketSize || len(v.ticket.Data) != TicketSize {
			panic("invalid ticket for voting")
		}

		num = copy(b.VoteProof[:], v.ticket.Proof)
		if num != ProofSize || len(v.ticket.Proof) != ProofSize {
			panic("invalid proof for voting")
		}

		num = copy(b.VotePK[:], v.pk)
		if num != PKSize || len(v.pk) != PKSize {
			panic("invalid pk for voting")
		}
	}

	return
}
