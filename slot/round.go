package slot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"math/big"
	"sync"
	"time"
)

// Block2 is the final structure that the protocol reached consensus over
type Block2 struct {
	Prev ID
	Data []byte
}

//Hash the block
func (b *Block2) Hash() ID {
	return ID(sha256.Sum256(bytes.Join([][]byte{
		b.Prev[:],
		b.Data,
	}, nil)))
}

//GenesisBlock returns the protocol constant first block
func GenesisBlock() *Block2 {
	return &Block2{
		Prev: NilID,
		Data: []byte("vi veri veniversum vivus vici"),
	}
}

// Proposal2 describes a block proposal from the network
type Proposal2 struct {
	Token []byte
	PK    []byte
	Proof []byte

	Block *Block2
}

// Vote2 describes a vote for a block from the network
type Vote2 struct {
	Token []byte
	PK    []byte
	Proof []byte

	Proposal *Proposal2
}

// Msg2 is our new message structure
type Msg2 struct {
	Proposal *Proposal2
	Vote     *Vote2
}

// Voter2 manages votes for a certain round
type Voter2 struct {
	bw         BroadcastWriter2
	round      uint64
	mu         sync.Mutex
	proposers  map[[PKSize]byte]struct{}
	top        *big.Int
	candidates []*Proposal2
	timer      *time.Timer
	casted     bool
}

// NewVoter2 sets up a voter for the round. It will collect proposals it has seen
// during a round and broadcasts the highest ranking ones as votes after a configurable
// timeout. New high-ranking proposals that arrive after this and are broadcasted
// as votes right away.
func NewVoter2(bw BroadcastWriter2, round uint64, bt time.Duration) (v *Voter2) {
	v = &Voter2{
		bw:        bw,
		round:     round,
		proposers: make(map[[PKSize]byte]struct{}),
		top:       big.NewInt(0),
		timer:     time.NewTimer(bt),
	}

	go func() {
		<-v.timer.C
		v.mu.Lock()
		defer v.mu.Unlock()

		err := v.cast()
		if err != nil {
			panic("failed to cast after timer expired: " + err.Error())
		}

		v.casted = true //mark as votes casted
	}()

	return
}

func (v *Voter2) cast() (err error) {
	for _, p := range v.candidates {
		pv := &Vote2{Proposal: p}
		err = v.bw.Write(v.round, &Msg2{Vote: pv})
		if err != nil {
			//@TODO do not cancel writing votes for other candidates if this fails
			panic("failed to write proposal vote: " + err.Error())
		}
	}

	v.candidates = []*Proposal2{} //reset candidates
	return nil
}

// Handle proposals for the round
func (v *Voter2) Handle(p *Proposal2) (err error) {
	if len(p.Token) != TicketSize {
		panic("wrong length for token in proposal")
	}

	if len(p.PK) != PKSize { //wrong length rank proof pk in proposal
		panic("wrong length token proof pk in proposal")
	}

	if len(p.Proof) != ProofSize { //wrong length rank proof in proposal
		panic("wrong length token proof pk in proposal")
	}

	if p.Block == nil { //no block in proposal
		panic("no block in proposal")
	}

	if p.Block.Prev == NilID { //no prev block in proposal
		panic("no prev block in propoal")
	}

	//@TODO verify VRF token with pk and proof
	//@TODO verify that the token provides the right to propose

	//@TODO if we add randomness to a ticket draw we need to order the blocks
	//we receive here on chain strength. Else proposers can grind blocks with
	//the different randomness input (old blocks) to find high ranks for their
	//blocks. Even though we allow only one proposal per round this might still
	//be harmfull. Alternatively we might require a small proof of work for each
	//proposal.

	//course lock
	v.mu.Lock()
	defer v.mu.Unlock()

	//enforce only one proposal per user per round
	//@TODO how unique are vrf public keys, 32 bytes is pretty good right?
	var pk [PKSize]byte
	copy(pk[:], p.PK)
	_, ok := v.proposers[pk]
	if ok { //already proposed
		panic("proposer already proposed a block for this round")
	}

	//check if the proposal is higher (or as high). If not do nothing, the network
	//has no futher use for this proposal so we filter it out
	rank := big.NewInt(0).SetBytes(p.Token)
	if rank.Cmp(v.top) > 0 {
		v.candidates = []*Proposal2{p} //reset, with new highest proposal
	} else if rank.Cmp(v.top) == 0 {
		v.candidates = append(v.candidates, p) //augment with also highest proposal
	} else {
		panic("proposed block has no higher or equal value then existing blocks")
	}

	//relay the proposal to the rest of network
	err = v.bw.Write(v.round, &Msg2{Proposal: p})
	if err != nil {
		panic("failed to relay proposal: " + err.Error())
	}

	//if votes were already casted, send out vote(s) right away
	if v.casted {
		err = v.cast()
		if err != nil {
			panic("failed to cast votes right away: " + err.Error())
		}
	}

	return
}

// Ballot handles votes for a certain round
type Ballot struct {
	blocks chan *Block2
	bw     BroadcastWriter2
	round  uint64
}

// NewBallot creates the round's vote handler
func NewBallot(bw BroadcastWriter2, round uint64) (b *Ballot) {
	b = &Ballot{bw: bw, blocks: make(chan *Block2), round: round}
	return
}

// Blocks channel that sets send any blocks that hav gathered enough votes
func (b *Ballot) Blocks() (c <-chan *Block2) {
	return b.blocks
}

// Handle votes for the round
func (b *Ballot) Handle(v *Vote2) (err error) {
	//@TODO verify syntax of vote, proposal and block

	//@TODO verify voter token
	//@TODo verify token gives voter privileges

	//@TODO if the votes block prev is not the highest ranking block in our view of the
	//world we ignore it: the voter was running behind and is on the wrong track.
	//if the prev is equal or higher then whant we know, do count the vote.

	//@TODO is the block in this vote highest ranking proposal we know of? If not
	//skip relaying or tallying the vote. Useless to the network

	//@TODO verify also that proposer has made no other proposals in this round,
	//the voter cannot immitate proposals so this check is ok and makes it harder
	//for proposers and voters to coerse

	//@TODO what if proposals have the same rank? We relay the votes still.
	//@TODO relay vote to peers

	//@TODO count votes until threshold
	//@TODO if threshold is reached, send block over result channel. We got ourselves
	//a new block

	return
}

// Round is a time bound segmentation of messages
type Round struct {
	bt   time.Duration
	num  uint64
	bc   Broadcast2
	tip  *Block2     //previous round's top block
	done chan *Round //we write to when we consider this round complete

	ballot *Ballot
	voter  *Voter2
}

// NewRound creates a round and start reading messages for the given round number
// from the broadcast layer. The tip is what we locally consider to be the highest
// ranking block we know of. We will base our proposal on this, if it turns out
// a later block comes around for this round we will not change as the proposal
// is already send out and everyone can only propose on block per round anyway.
// We will sill consider votes for blocks that have a higher ranking prev block
// so we can still complete this round
func NewRound(bc Broadcast2, num uint64, tip *Block2, bt time.Duration) (r *Round) {
	r = &Round{
		num:  num,
		bc:   bc,
		tip:  tip,
		done: make(chan *Round),

		ballot: NewBallot(bc, num),
		voter:  NewVoter2(bc, num, bt),
	}

	//@TODO draw ticket just by round, pass to voter to influence if it should
	//sends out votes (or handle vote messages at all). A possible downside of
	//doing it just by number is that people can calculat their values ahead of
	//time and decide when they want to actually show up and propose something.
	//this might influence livelyness in a negative way

	//message ingress
	go func() {
		for {
			msg := &Msg2{}
			err := r.bc.Read(r.num, msg)
			if err == io.EOF {
				return //done, closing down
			} else if err != nil {
				panic("failed to read message: " + err.Error())
			}

			//decide what to do with the message
			if msg.Proposal != nil {
				err = r.voter.Handle(msg.Proposal)
				if err != nil {
					panic("failed to handle proposal: " + err.Error())
				}

			} else if msg.Vote != nil {
				err = r.ballot.Handle(msg.Vote)
				if err != nil {
					panic("failed to handle vote: " + err.Error())
				}

			} else {
				panic("saw unknown message, neither vote nor proposal")
			}
		}
	}()

	//valid blocks handling
	go func() {
		for {
			select {
			case b := <-r.ballot.Blocks():
				//new block with enough votes, we will immediately consider this round
				//closed and build a new round on top of it. If this turns out to be wrong we
				//might be useless as a proposer (because we propose low ranking blocks) but
				//we can still do voting as long as we at some point receive that prev block
				//everyone is going on about

				//@TODO store it somewhere, resolve OutOfOrder etc

				//with at least one new block we can call this round complete, we might
				//receive more votes for higher prio blocks but we won't be doing anything
				//with that for our position in the next round
				r.done <- NewRound(r.bc, r.num+1, b, r.bt)
				close(r.done)
			}
		}
	}()

	return r
}

// Run the round by writing any propoposal and block until at least 1 block with
// a majority vote was found by the network. Returns a new round when this
// succeeded or an error when the timeout expired
func (r *Round) Run(ctx context.Context) (newr *Round, err error) {
	//@TODO if our ticket gives us proposal right, make a proposal right away. and
	//write it to network

	//@TODO figure out how we determine the 'prev' of this block? we always propose
	//on what we believe was the highest. if this turns out to be wrong our proposal
	//will not be accepted and no votes will favor it. We will receive votes for blocks
	//that have a higher prev rank then ours so we can still complete the round. Our
	//voter role is also fine as we can still determine if the block that is voted
	//for is

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case newr = <-r.done:
		return
	}
}
