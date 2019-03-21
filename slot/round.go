package slot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/advanderveer/27067dd17/vrf"
)

// Block2 is the final structure that the protocol reached consensus over
type Block2 struct {
	Prev ID
	Data []byte
}

//GenesisBlock returns the protocol constant first block
func GenesisBlock() *Block2 {
	return &Block2{
		Prev: NilID,
		Data: []byte("vi veri veniversum vivus vici"),
	}
}

//Hash the block
func (b *Block2) Hash() ID {
	return ID(sha256.Sum256(bytes.Join([][]byte{
		b.Prev[:],
		b.Data,
	}, nil)))
}

//Validate the syntax
func (b *Block2) Validate() (ok bool, err error) {
	if b.Prev == NilID { //no prev block in proposal
		return false, fmt.Errorf("empty prev")
	}

	return true, nil
}

// Proposal2 describes a block proposal from the network
type Proposal2 struct {
	Token []byte
	PK    []byte
	Proof []byte

	Block *Block2
}

//Validate the syntax of a proposal
func (p *Proposal2) Validate() (ok bool, err error) {
	if len(p.Token) != TicketSize {
		return false, fmt.Errorf("wrong length for token")
	}

	if len(p.PK) != PKSize {
		return false, fmt.Errorf("wrong length for token proof pk")
	}

	if len(p.Proof) != ProofSize {
		return false, fmt.Errorf("wrong length for token proof")
	}

	if p.Block == nil {
		return false, fmt.Errorf("no block in proposal")
	}

	return p.Block.Validate()
}

// Vote2 describes a vote for a block from the network
type Vote2 struct {
	Token []byte
	PK    []byte
	Proof []byte

	Proposal *Proposal2
}

//Validate the syntax
func (v *Vote2) Validate() (ok bool, err error) {
	if len(v.Token) != TicketSize {
		return false, fmt.Errorf("wrong length for token")
	}

	if len(v.PK) != PKSize {
		return false, fmt.Errorf("wrong length for token proof pk")
	}

	if len(v.Proof) != ProofSize {
		return false, fmt.Errorf("wrong length for token proof")
	}

	if v.Proposal == nil {
		return false, fmt.Errorf("no proposal in vote")
	}

	return v.Proposal.Validate()
}

// Msg2 is our new message structure
type Msg2 struct {
	Proposal *Proposal2
	Vote     *Vote2
}

//ProposerFilter makes it easier to check if the proposal was proposed by a
//proposer that already sent another proposal for a block
type ProposerFilter map[[PKSize]byte]map[[TicketSize]byte]struct{}

//NewProposerFilter creates an empty proposer filter
func NewProposerFilter() ProposerFilter {
	return make(ProposerFilter)
}

// Add will add the proposal to the filter, it returns how often a proposer has
// then proposed during the round
func (pf ProposerFilter) Add(pkb []byte, token []byte) (n int) {
	var pk [PKSize]byte
	copy(pk[:], pkb)

	pids, ok := pf[pk]
	if !ok {
		pids = make(map[[TicketSize]byte]struct{})
	}

	var t [TicketSize]byte
	copy(t[:], token)

	pids[t] = struct{}{}
	pf[pk] = pids
	return len(pids)
}

// Voter2 manages votes for a certain round
type Voter2 struct {
	tip        ID
	bw         BroadcastWriter2
	round      uint64
	mu         sync.Mutex
	pf         ProposerFilter
	top        *big.Int
	candidates []*Proposal2
	timer      *time.Timer
	casted     bool
}

// NewVoter2 sets up a voter for the round. It will collect proposals it has seen
// during a round and broadcasts the highest ranking ones as votes after a configurable
// timeout. New high-ranking proposals that arrive after this and are broadcasted
// as votes right away.
func NewVoter2(bw BroadcastWriter2, round uint64, tip ID, bt time.Duration) (v *Voter2) {
	v = &Voter2{
		tip:   tip,
		bw:    bw,
		round: round,
		pf:    NewProposerFilter(),
		top:   big.NewInt(0),
		timer: time.NewTimer(bt),
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

	//validate syntax
	ok, err := p.Validate()
	if !ok {
		panic("failed to validate proposal: " + err.Error())
	}

	//@TODO verify VRF token with pk and proof
	//@TODO verify that the token provides the right to propose

	//@TODO if we add randomness to a ticket draw we need to order the blocks
	//we receive here on chain strength. else proposers can grind blocks with
	//the different randomness input (old blocks) to find high ranks for their
	//blocks. Even though we allow only one proposal per round this might still
	//be harmfull. Alternatively we might require a small proof of work for each
	//proposal.

	//course lock
	v.mu.Lock()
	defer v.mu.Unlock()

	// if the proposed block is building on a tip that we don't consider to be
	// winning in the last round we do not vote on the proposal.
	if p.Block.Prev != v.tip {
		panic("vote's proposed block is build on a tip that we don't consider to be winning")
	}

	//enforce only one proposal per user per round. This is not triggered if
	//we receive the same block from the same proposal. This is triggered when
	//the proposal comes with a different trigger
	npro := v.pf.Add(p.PK, p.Token)
	if npro > 1 { //already proposed at least onces
		panic("proposer already proposed a block for this round")
	}

	//check if the proposal is higher (or as high). If not do nothing, the network
	//has no futher use for this proposal so we filter it out
	rank := big.NewInt(0).SetBytes(p.Token)
	if rank.Cmp(v.top) > 0 {
		v.candidates = []*Proposal2{p} //reset, with new highest proposal
		v.top = rank
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
	tip    ID
	blocks chan *Block2
	bw     BroadcastWriter2
	round  uint64
	votes  map[ID]map[[PKSize]byte]struct{}
	pf     ProposerFilter
	minv   uint64
	mu     sync.RWMutex
	top    *big.Int
}

// NewBallot creates the round's vote handler
func NewBallot(bw BroadcastWriter2, round uint64, tip ID, minv uint64) (b *Ballot) {
	b = &Ballot{
		tip:    tip,
		bw:     bw,
		blocks: make(chan *Block2),
		round:  round,
		votes:  make(map[ID]map[[TicketSize]byte]struct{}),
		pf:     NewProposerFilter(),
		minv:   minv,
		top:    big.NewInt(0),
	}
	return
}

// Blocks channel that sets send any blocks that hav gathered enough votes
func (b *Ballot) Blocks() (c <-chan *Block2) {
	return b.blocks
}

// Handle votes for the round
func (b *Ballot) Handle(v *Vote2) (err error) {

	//validate syntax
	ok, err := v.Validate()
	if !ok {
		panic("failed to validate vote: " + err.Error())
	}

	//@TODO verify voter token
	//@TODO verify token gives voter privileges

	b.mu.Lock()
	defer b.mu.Unlock()

	// if the vote's proposed block is building on a tip that we don't consider to
	// be the winning in the last round we drip the vote.
	if v.Proposal.Block.Prev != b.tip {
		panic("vote's proposed block is build on a tip that we don't consider to be winning")
	}

	// if the proposal in this vote doesn't have the highest rank we know of skip
	// relay to other peers.
	rank := big.NewInt(0).SetBytes(v.Proposal.Token)
	if rank.Cmp(b.top) >= 0 {
		b.top = rank
	} else {
		panic("vote's proposal had a lower rank then an other proposa we already saw")
	}

	// check that we're not counting votes for proposals that were proposed while
	// a proposer send out other proposals during the round
	npro := b.pf.Add(v.Proposal.PK, v.Proposal.Token)
	if npro > 1 { //already proposed at least onces
		panic("proposer already proposed a block for this round")
	}

	// at this point we consider the vote usefull to the network: relay it
	err = b.bw.Write(b.round, &Msg2{Vote: v})
	if err != nil {
		panic("failed to relay vote: " + err.Error())
	}

	// count votes
	id := v.Proposal.Block.Hash()
	votes, ok := b.votes[id]
	if !ok {
		votes = make(map[[TicketSize]byte]struct{})
	}

	// map uniquely to a voter's identity so each voter can only vote once on a proposal
	var voter [PKSize]byte
	copy(voter[:], v.PK)
	votes[voter] = struct{}{}
	b.votes[id] = votes

	// when exacty 1 vote over the threshold we consider the majority being in
	// favor and accept the block
	if uint64(len(votes)) == b.minv+1 {
		b.blocks <- v.Proposal.Block

		// At this point we can close the channel, we filtered votes that
		// are not voting for the highest known proposal and the ones that were
		// not filtered reached a threshold. We do not expect that another proposal
		// will suddenly reach a threshold and can conclude the round.
		close(b.blocks)
	}

	return
}

//Lottery uses a long term public and private key to draw random tokens locally
//that others can verify: A VRF
type Lottery struct {
	pk []byte                   //public key
	sk *[vrf.SecretKeySize]byte //secret key
}

//NewLottery will generate keys and setup a local lottery using the provided
//source of crypto randomness
func NewLottery(rndr io.Reader) (lt *Lottery) {
	lt = &Lottery{}

	var err error
	lt.pk, lt.sk, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys: " + err.Error())
	}

	return
}

//Draw a new token for the lottery for the provided round number
func (lt *Lottery) Draw(round uint64) (token, proof, pk []byte) {
	roundd := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundd, round)

	token, proof = vrf.Prove(roundd, lt.sk)
	return token, proof, lt.pk
}

// Round is a time bound segmentation of messages that uses messages read from the
// broadcast to find a block that is mostly likey to be part of the consensus chain
// very soon.
type Round struct {
	bt   time.Duration //timeout until we cast vote
	minv uint64        //minimum nr of votes until we accept a block

	num uint64     //round number
	lt  *Lottery   //vrf lottery
	bc  Broadcast2 //message broadcast
	tip ID         //previous round winning block

	ballot *Ballot
	voter  *Voter2
	mu     sync.RWMutex

	token []byte
	proof []byte
	pk    []byte
}

// NewRound creates a round and start reading messages for the given round number
// from the broadcast layer. The tip is what we locally consider to be the highest
// ranking block we know of. We will base our proposal on this, if it turns out
// a later block comes around for this round we will not change as the proposal
// is already send out and everyone can only propose on block per round anyway.
// We will sill consider votes for blocks that have a higher ranking prev block
// so we can still complete this round
func NewRound(lt *Lottery, bc Broadcast2, num uint64, tip ID, bt time.Duration, minv uint64) (r *Round) {
	r = &Round{
		bt:   bt,
		minv: minv,
		tip:  tip,

		num: num,
		lt:  lt,
		bc:  bc,

		ballot: NewBallot(bc, num, tip, minv),
		voter:  NewVoter2(bc, num, tip, bt),
	}

	//draw ticket just by round nr. A possible downside of
	//doing it just by number is that people can calculate their values ahead of
	//time and decide when they want to actually show up and propose something.
	//this might influence livelyness in a negative way
	r.token, r.proof, r.pk = lt.Draw(num)

	//@TODO based on token, check threshold to see if we can be a voter. If not
	//do not handle those messages at all.

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

	return r
}

// Num returns the round number
func (r *Round) Num() uint64 {
	return r.num
}

// Tip returns the winning block id from the previous round that this round is
// building on
func (r *Round) Tip() ID {
	return r.tip
}

// Run the round by writing any propoposal and block until at least 1 block with
// a majority vote was found by the network. Returns a new round when this
// succeeded or an error when the timeout expired
func (r *Round) Run(ctx context.Context) (newr *Round, err error) {
	//@TODO based on token, check threshold and see if we are granted the right to
	//propose or not

	//write our own block proposal for the network
	if err = r.bc.Write(r.num, &Msg2{
		Proposal: &Proposal2{
			Token: r.token,
			PK:    r.pk,
			Proof: r.proof,
			Block: &Block2{
				Prev: r.tip,
				Data: r.pk,
			},
		},
	}); err != nil {
		panic("failed to write our own proposal to the broadcast: " + err.Error())
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case b := <-r.ballot.Blocks():
		return NewRound(r.lt, r.bc, r.num+1, b.Hash(), r.bt, r.minv), nil
	}
}
