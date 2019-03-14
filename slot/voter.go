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
	bw     BroadcastWriter
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
	//be very low.

	//@TODO accept only block proposals from a proposer if it hasn't proposed any
	//other blocks this round. This prevents grinding attacks in which members will
	//try any other tip from the last round that might give them better odds to
	//have their block be voted on.

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

	// @TODO instead of ordering by indiviual stregth we might want to order by
	// by chain strength. This would prevent grinding attacks in which members
	// would find a old block that gives them an exceptional high ticket draw
	// for the round to ensure they can become a proposer (and reap the reward)
	// this check prevents this because choosing a low block gives you a very
	// low chain strength.

	curr := new(big.Rat)
	for _, b := range v.votes {
		curr.Set(b.Strength(1))
		break
	}

	new := b.Strength(1)
	if new.Cmp(curr) > 0 { //nuke proposal, new block is bigger
		v.votes = make(map[ID]*Block)
		v.votes[id] = b

		//if we can write to the broadcast network, do so right away
		if v.bw != nil {
			err := v.bw.Write(&Msg{Vote: v.sign(b)})
			if err != nil {
				//@TODO log this but continue writing other votes
			}
		}

		return true, 1
	}

	if new.Cmp(curr) == 0 { //add to proposal, block is as big
		_, ok := v.votes[id]
		v.votes[id] = b

		//new block that also has the highest score, if possible broadcast right
		//away
		if !ok && v.bw != nil {
			err := v.bw.Write(&Msg{Vote: v.sign(b)})
			if err != nil {
				//@TODO log this but continue writing other votes
			}
		}

		return !ok, len(v.votes)
	}

	return false, len(v.votes)
}

func (v *Voter) sign(b *Block) (bv *Vote) {
	bv = &Vote{Block: b}

	num := copy(bv.VoteTicket[:], v.ticket.Data)
	if num != TicketSize || len(v.ticket.Data) != TicketSize {
		panic("invalid ticket for voting")
	}

	num = copy(bv.VoteProof[:], v.ticket.Proof)
	if num != ProofSize || len(v.ticket.Proof) != ProofSize {
		panic("invalid proof for voting")
	}

	num = copy(bv.VotePK[:], v.pk)
	if num != PKSize || len(v.pk) != PKSize {
		panic("invalid pk for voting")
	}

	return
}

// Vote will return the highest scoring blocks the voter it has seen for this
// round with a proof that it was allowed to cast a vote. It will also write
// all the votes to the broadcast writer, and from now on all new highest blocks
// that are proposed will automatically be broadcasted
func (v *Voter) Vote(bw BroadcastWriter) (votes []*Vote) {
	v.mu.Lock()
	for _, b := range v.votes {
		votes = append(votes, v.sign(b))
	}

	v.bw = bw
	v.mu.Unlock()

	for _, bv := range votes {
		err := v.bw.Write(&Msg{Vote: bv})
		if err != nil {
			//@TODO log this but continue writing other votes
		}
	}

	return
}
