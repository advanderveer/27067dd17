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

// Notary keeps state for block notarization of a certain round
type Notary struct {
	round    uint64
	reader   BlockReader
	proposal map[ID]*Block
	ticket   Ticket
	pk       []byte
	mu       sync.Mutex
}

// NewNotary creates a block notarization
func NewNotary(round uint64, r BlockReader, t Ticket, pk []byte) (n *Notary) {
	n = &Notary{
		round:    round,
		reader:   r,
		proposal: make(map[ID]*Block),
		ticket:   t,
		pk:       pk,
	}

	return
}

// Verify the block according to notarization rules
func (n *Notary) Verify(b *Block) (ok bool, err error) {
	if b.Round != n.round {
		return false, ErrWrongRound
	}

	//@TODO verify that the picked prev is not too far in the past, this
	//could result in an artifical low threshold that the user may force itself
	//into for the current round? i.e: create a block for this round based on the
	//genesis block. All open blocks in between cause will cause the threshold to
	//be very low

	prevb := n.reader.Read(b.Prev)
	if prevb == nil {
		return false, ErrPrevNotExist
	}

	if !vrf.Verify(b.PK[:], Seed(prevb, b.Round), b.Ticket[:], b.Proof[:]) {
		return false, ErrProposeProof
	}

	return true, nil
}

// Propose will ask the notary to consider this block for the next notarization.
// returns whether this block is (one of) the highest in this round and the new
// number of highest blocks
func (n *Notary) Propose(b *Block) (ok bool, nh int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	id := b.Hash()
	if len(n.proposal) < 1 {
		n.proposal[id] = b
		return true, 1
	}

	curr := new(big.Rat)
	for _, b := range n.proposal {
		curr.Set(b.Strength(1))
		break
	}

	new := b.Strength(1)
	if new.Cmp(curr) > 0 { //nuke proposal, new block is bigger
		n.proposal = make(map[ID]*Block)
		n.proposal[id] = b
		return true, 1
	}

	if new.Cmp(curr) == 0 { //add to proposal, block is as big
		_, ok := n.proposal[id]
		n.proposal[id] = b
		return !ok, len(n.proposal)
	}

	return false, len(n.proposal)
}

// Notarize will return the highest scoring blocks it has seen for this round
// with a proof of notarization.
func (n *Notary) Notarize() (nots []*Block) {
	n.mu.Lock()
	for _, b := range n.proposal {
		nots = append(nots, b)
	}

	n.proposal = make(map[ID]*Block)
	n.mu.Unlock()

	for _, b := range nots {
		num := copy(b.NtTicket[:], n.ticket.Data)
		if num != TicketSize || len(n.ticket.Data) != TicketSize {
			panic("invalid ticket for notarization")
		}

		num = copy(b.NtProof[:], n.ticket.Proof)
		if num != ProofSize || len(n.ticket.Proof) != ProofSize {
			panic("invalid proof for notarization")
		}

		num = copy(b.NtPK[:], n.pk)
		if num != PKSize || len(n.pk) != PKSize {
			panic("invalid pk for notarization")
		}
	}

	return
}
