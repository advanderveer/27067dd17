package slot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
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
		return false, ErrInvalidBlockNilPrev
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

// Hash the proposal into a unique ID
func (p *Proposal2) Hash() ID {
	var bid ID
	if p.Block != nil {
		bid = p.Block.Hash()
	}

	return ID(sha256.Sum256(bytes.Join([][]byte{
		p.Token,
		p.PK,
		p.Proof,
		bid[:],
	}, nil)))
}

//Validate the syntax of a proposal
func (p *Proposal2) Validate() (ok bool, err error) {
	if len(p.Token) != TicketSize {
		return false, ErrInvalidProposalTokenLen
	}

	if len(p.PK) != PKSize {
		return false, ErrInvalidProposalPKLen
	}

	if len(p.Proof) != ProofSize {
		return false, ErrInvalidProposalProofLen
	}

	if p.Block == nil {
		return false, ErrInvalidProposalNoBlock
	}

	return p.Block.Validate()
}

//Rank of the proposal
func (p *Proposal2) Rank() (r *big.Int) {
	r = big.NewInt(0)
	r.SetBytes(p.Token)
	return
}

//RanksGtOrEqThen returns true if this proposal ranks higher then the other proposal
func (p *Proposal2) RanksGtOrEqThen(other *Proposal2) (ok bool) {
	if p.Rank().Cmp(other.Rank()) >= 0 {
		return true
	}

	return false
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
		return false, ErrInvalidVoteTokenLen
	}

	if len(v.PK) != PKSize {
		return false, ErrInvalidVotePKLen
	}

	if len(v.Proof) != ProofSize {
		return false, ErrInvalidVoteProofLen
	}

	if v.Proposal == nil {
		return false, ErrInvalidVoteNoProposal
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
	ltop       *Proposal2
	bw         BroadcastWriter2
	lt         *Lottery
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
func NewVoter2(bw BroadcastWriter2, lt *Lottery, round uint64, ltop *Proposal2, bt time.Duration) (v *Voter2) {
	v = &Voter2{
		ltop:  ltop,
		bw:    bw,
		lt:    lt,
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
	}

	if err != nil {
		return
	}

	v.candidates = []*Proposal2{} //reset candidates
	return nil
}

// ProposalResult provide structured feedback about a proposal was handled
type ProposalResult struct {

	//Syntax of the proposal is invalid
	SyntaxInvalid bool

	//Syntax validation showed this error
	SyntaxValidationErr error

	//provided lottery token is invalid
	ProposalTokenInvalid bool

	//Propose block is build on a tip we don't know
	UnexpectedPrev bool

	//It was detected that a proposer already proposed something else
	ProposerAlreadyProposed bool

	//Proposal's token doesn't rank higher then something we already saw
	ProposalRankNotHigher bool

	//Relay has failed
	RelayFailure error

	//Immediate cast because the timer has expired already
	ImmediateCast bool

	//Immediate vote casting because the timer expired, failed
	ImmediateCastFailure error
}

// Handle proposals for the round
func (v *Voter2) Handle(p *Proposal2) (res ProposalResult) {

	//validate syntax
	ok, err := p.Validate()
	if !ok {
		res.SyntaxInvalid = true
		res.SyntaxValidationErr = err
		return
	}

	//verify proposer token
	ok = v.lt.Verify(v.round, p.PK, p.Token, p.Proof)
	if !ok {
		res.ProposalTokenInvalid = true
		return
	}

	//@TODO verify that the token grants the right to propose (including stake)
	//@TODO if we add randomness to a ticket draw we need to order the blocks
	//we receive here on chain strength. else proposers can grind blocks with
	//the different randomness input (old blocks) to find high ranks for their
	//blocks. Even though we allow only one proposal per round this might still
	//be harmfull. Alternatively we might require a small proof of work for each
	//proposal.

	//course lock
	v.mu.Lock()
	defer v.mu.Unlock()

	// @TODO according to the DFinity Paper we also need to check if the prev
	// block is the previous round: "prv B = H(B′) and rd B′ = rd B − 1," and
	// if there is "notarization" of the previous block.

	// if we encounter a proposal that refers to a block in a previous round that
	// is not the same as the winner of our previous round it could mean
	// that that block does not exist or we do not know about it. Before voting on
	// it we need evicdence that it even exists. We can either wait for evidence of
	// this previous block (proposal) to arrive out of order or not vote on it at all
	// and trust that order voters are more informed and pick up the slack. We chose
	// the latter but this behaviour needs more testing to know how this influences
	// liveness much.
	if p.Block.Prev != v.ltop.Block.Hash() {
		res.UnexpectedPrev = true
		return
	}

	//enforce only one proposal per user per round. This is not triggered if
	//we receive the same block from the same proposal. This IS triggered when
	//the proposal comes with a different token but the same block
	npro := v.pf.Add(p.PK, p.Token)
	if npro > 1 { //already proposed at least onces
		res.ProposerAlreadyProposed = true
		return
	}

	//check if the proposal is higher (or as high). If not, do nothing, the network
	//has no futher use for this proposal so we filter it out
	rank := big.NewInt(0).SetBytes(p.Token)
	if rank.Cmp(v.top) > 0 {
		v.candidates = []*Proposal2{p} //reset, with new highest proposal
		v.top = rank
	} else if rank.Cmp(v.top) == 0 {
		v.candidates = append(v.candidates, p) //augment with also highest proposal
	} else {
		res.ProposalRankNotHigher = true
		return
	}

	//relay the proposal to the rest of network, if this fails do not stop
	//the handling logic
	err = v.bw.Write(v.round, &Msg2{Proposal: p})
	if err != nil {
		res.RelayFailure = err
	}

	//if votes were already cast as part of the timeout, send this new
	//vote to peers right away
	if v.casted {
		res.ImmediateCast = true
		res.ImmediateCastFailure = v.cast()
	}

	return
}

// Ballot handles votes for a certain round
type Ballot struct {
	ltop     *Proposal2
	lt       *Lottery
	accepted chan *Proposal2
	bw       BroadcastWriter2
	round    uint64
	votes    map[ID]map[[PKSize]byte]struct{}
	pf       ProposerFilter
	minv     uint64
	mu       sync.RWMutex
	top      *big.Int
}

// NewBallot creates the round's vote handler
func NewBallot(bw BroadcastWriter2, lt *Lottery, round uint64, top *Proposal2, minv uint64) (b *Ballot) {
	b = &Ballot{
		ltop:     top,
		bw:       bw,
		lt:       lt,
		accepted: make(chan *Proposal2),
		round:    round,
		votes:    make(map[ID]map[[TicketSize]byte]struct{}),
		pf:       NewProposerFilter(),
		minv:     minv,
		top:      big.NewInt(0),
	}
	return
}

// Accepted channel will return a proposal that we've accepted
func (b *Ballot) Accepted() (c <-chan *Proposal2) {
	return b.accepted
}

// Handle votes for the round
func (b *Ballot) Handle(v *Vote2) (err error) {

	//validate syntax
	ok, err := v.Validate()
	if !ok {
		panic("failed to validate vote: " + err.Error())
	}

	//course lock
	b.mu.Lock()
	defer b.mu.Unlock()

	//verify voter token
	ok = b.lt.Verify(b.round, v.PK, v.Token, v.Proof)
	if !ok {
		panic("invalid voter token")
	}

	//verify proposer token
	ok = b.lt.Verify(b.round, v.Proposal.PK, v.Proposal.Token, v.Proposal.Proof)
	if !ok {
		panic("invalid proposer token")
	}

	//@TODO verify token gives voter privileges (including stake)

	// if the proposed block's prev in the vote doesn't match our winning block
	// from the last round it could mean that it doesn't exist or we do not know
	// about it (yet). We do not draw  any conclusion about that fact here and
	// assume that voters know what they are doing. In the case we are wrong this
	// allows us to simply catch up to the system by listening to vote messages
	// and start our own rounds from majority votes.
	if v.Proposal.Block.Prev != b.ltop.Block.Hash() {
		//@TODO for now we ignore this, but find out what that means for security
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

	// count proposal votes
	id := v.Proposal.Hash()
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
		b.accepted <- v.Proposal

		// At this point we can close the channel, we filtered votes that
		// are not voting for the highest known proposal and the ones that were
		// not filtered reached a threshold. We do not expect that another proposal
		// will suddenly reach a threshold and can conclude the round.
		close(b.accepted)
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

func (lt *Lottery) seed(round uint64) (seed []byte) {
	seed = make([]byte, 8)
	binary.LittleEndian.PutUint64(seed, round)
	return
}

//Draw a new token for the lottery for the provided round number
func (lt *Lottery) Draw(round uint64) (token, proof, pk []byte) {
	token, proof = vrf.Prove(lt.seed(round), lt.sk)
	return token, proof, lt.pk
}

//Verify the token drawn by a lottery from another member
func (lt *Lottery) Verify(round uint64, pk, token, proof []byte) (ok bool) {
	return vrf.Verify(pk, lt.seed(round), token, proof)
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
	top *Proposal2 //top proposal from the last round

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
func NewRound(lt *Lottery, bc Broadcast2, num uint64, top *Proposal2, bt time.Duration, minv uint64) (r *Round) {
	r = &Round{
		bt:   bt,
		minv: minv,
		top:  top,

		num: num,
		lt:  lt,
		bc:  bc,

		ballot: NewBallot(bc, lt, num, top, minv),
		voter:  NewVoter2(bc, lt, num, top, bt),
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
				res := r.voter.Handle(msg.Proposal)
				_ = res //@TODO log and handle gracefully
				// if err != nil {
				// 	panic("failed to handle proposal: " + err.Error())
				// }

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

// Top returns the winning proposal from the last round
func (r *Round) Top() *Proposal2 {
	return r.top
}

// Run the round by writing any propoposal and block until at least 1 block with
// a majority vote was found by the network. Returns a new round when this
// succeeded or an error when the timeout expired
func (r *Round) Run(ctx context.Context) (newr *Round, err error) {
	//@TODO based on token, check threshold and see if we are granted the right to
	//propose or not

	//@TODO add a vote for our own proposal to the voter right away, if nothing
	//else we will vote for our own proposal which is fine

	//@TODO in the DFINITY paper it states that: "We emphasize that a block contains
	//the notarization z of the previous block in the chain that it references.".
	//later is explained that this is to ensure that notarizations are published on
	//time and not withheld. It seems that in our case this mechanism is not
	//necessary at all because notarization tokens (and proposal tokens) are only
	//valid for a specific round anyway.

	//write our own block proposal for the network
	if err = r.bc.Write(r.num, &Msg2{
		Proposal: &Proposal2{
			Token: r.token,
			PK:    r.pk,
			Proof: r.proof,
			Block: &Block2{
				Prev: r.top.Block.Hash(),
				Data: r.pk,
			},
		},
	}); err != nil {
		panic("failed to write our own proposal to the broadcast: " + err.Error())
	}

	select {
	case <-ctx.Done(): //took too long
		return nil, ctx.Err()
	case p := <-r.ballot.Accepted(): //accepted proposal

		//@TODO stop voting process once we've accepted a proposal for this round

		//@TODO we want to build on the longest chain, by creating the new round and
		//passing the proposal we might not be doing that? How does that influence
		//consensus. In case of a network split there might be two winning blocks.
		//According to the DFINITY paper: "As soon as clients receive a notarized block,
		//they use it to extend their copies of the blockchain thereby ending round
		//r in their respective views.", It also states that the notary is optimistic
		//and non-interactive

		return NewRound(r.lt, r.bc, r.num+1, p, r.bt, r.minv), nil
	}
}
