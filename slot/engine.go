package slot

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/advanderveer/27067dd17/vrf"
)

// Engine manages the core message handling logic and always corresponds to
// one identity in the system.
type Engine struct {
	vrfSK *[vrf.SecretKeySize]byte //private key for verifiable random function
	vrfPK []byte                   //public key for verifiable random function
	rxMsg uint64                   //received message count
	txMsg uint64                   //transmit message count

	minVotes  uint64        //minimum nr votes requires before a block is appended
	blockTime time.Duration //the time after which voters will cast their votes to the network

	bc    Broadcast   //message broadcast  rx/tx
	chain *Chain      //holds state for voted blocks
	voter *Voter      //hold our voting state (if we have the right)
	ooo   *OutOfOrder //out-of-order message handling
}

// NewEngine sets up the engine
func NewEngine(vrfpk []byte, vrfsk *[vrf.SecretKeySize]byte, bc Broadcast, bt time.Duration, minVotes uint64) (e *Engine) {
	e = &Engine{
		vrfSK:     vrfsk,
		vrfPK:     vrfpk,
		rxMsg:     0,
		txMsg:     0,
		chain:     NewChain(),
		bc:        bc,
		blockTime: bt,
		minVotes:  minVotes,
	}

	e.ooo = NewOutOfOrder(e.bc, e.chain, e.Handle)
	return
}

// Read a block from the chain, if it doesn't exist it returns nil
func (e *Engine) Read(id ID) (b *Block) {
	return e.chain.Read(id)
}

// Stats returns statistics about the engine
func (e *Engine) Stats() (rx, tx uint64, votes map[ID]*Block) {
	rx = atomic.LoadUint64(&e.rxMsg)
	tx = atomic.LoadUint64(&e.txMsg)

	//@TODO protect with lock
	if e.voter == nil {
		return rx, tx, nil
	}

	votes = make(map[ID]*Block)
	for id, b := range e.voter.votes {
		votes[id] = b
	}

	return
}

// Run will keep reading messages from the broadcast layer and write new
// messages to it.
func (e *Engine) Run() (err error) {
	curr := &Msg{}
	for {
		err = e.bc.Read(curr)
		if err == io.EOF {
			return ErrBroadcastClosed
		} else if err != nil {
			return MsgError{T: curr.Type(), N: atomic.LoadUint64(&e.rxMsg) + 1, E: err, M: "read message from broadcast"}
		}

		n := atomic.AddUint64(&e.rxMsg, 1)
		err := e.ooo.Handle(curr)
		if err != nil {
			return MsgError{T: curr.Type(), N: n, E: err, M: "handle rx message"}
		}
	}
}

// Handle a single message, messages may arrive in any order.
func (e *Engine) Handle(in *Msg, bw BroadcastWriter) (err error) {
	switch in.Type() {
	case MsgTypeVote: //vote message
		return e.HandleVote(in.Vote, bw)
	case MsgTypeProposal: //proposal message
		return e.HandleProposal(in.Proposal, bw)
	default:
		return ErrUnknownMessage
	}
}

// HandleVote is called when a block with vote comes along. This might be
// called out-of-order so potentially hours after the message actually arriving
// on the machine. @TODO this implementation is currently full of race conditions
// and should not be called concurrently with any other method
func (e *Engine) HandleVote(v *Vote, bw BroadcastWriter) (err error) {
	// (2.0) @TODO do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.2) verify vote signature and threshold, if invalid discard message. No
	// relay or state changes from this message.
	// (2.3) verify proposal signature, if invalid, discard message. No relay or state
	// change. it will only pollute the network.
	// - Verify if it a vote at all
	// - Verify that the proposer pk didn't already propose a block @TODO what if
	//   others use another pk then there own?
	ok, err := e.chain.Verify(v)
	if !ok {
		_ = err    //@TODO log verification errors
		return nil //not a valid block
	}

	// (2.4) at this point it is ok to relay the vote message, broadcast
	// will take care of message deduplication.
	err = bw.Write(&Msg{Vote: v})
	if err != nil {
		//@TODO log this, but don't fail
		// return fmt.Errorf("failed to relay notarized block: %v", err)
	}

	// (2.5) add vote to a counter of unique votes for that block. If
	// counter already reached the threshold value we we can skip the rest and return.
	// @TODO how do we prevent the voting count to grow unbounded:
	// - an unhonest voter might sign and vote on any block proposal it sees.
	// - an unhonest voter might keep signing blocks in a round that are created
	//   ad infinitum by a proposer that is also under its control for that round
	//   @TODO how to prevent this?: Every proposer can propose only one block per round
	nvotes := e.chain.Tally(v)

	// (2.6) if the counter passed the vote threshold we can now append the block to our chain.
	// and resolve any out of order handle calls that are waiting
	// @TODO can the minvotes be based on chain history?
	if uint64(nvotes) <= e.minVotes {
		return nil //not enough votes (yet), the block in this vote is worth nothing (yet)
	}

	//resolve out-of-order blocks, messages can now be handled
	err = e.ooo.Resolve(v.Block)
	if err != nil {
		//@TODO log resolve errors
	}

	// (2.7) if we are a voter and the block is of the same round as we're voting for
	// for we will stop casting voting
	// @TODO check if it is not our own message or vote (how?) can others imitate
	if e.voter != nil {

		//send any open votes onto the network and shutdown
		//@TODO race condition
		//@TODO ignore our own vote
		_ = e.voter.Cast(bw)
		e.voter = nil
	}

	//append the block in the vote and see if it changes the tip
	_, newtip := e.chain.Append(v.Block)
	if !newtip {
		return nil //tip didn't change nothing left to do
	}

	return e.HandleVoteIntoNewTip(bw)
}

// HandleVoteIntoNewTip is called when a vote caused the member to experience a
// new tip.
func (e *Engine) HandleVoteIntoNewTip(bw BroadcastWriter) (err error) {

	// (2.8) if the tip changed we can now draw another ticket.
	tipb := e.chain.Read(e.chain.Tip())
	if tipb == nil {
		panic("new tip block doesn't exist")
	}

	//@TODO for which round do we want to draw? always tip round +1? Can the system
	//get stuck in a certain round on a certain tip?
	newround := tipb.Round + 1

	ticket, err := e.chain.Draw(e.vrfPK, e.vrfSK, e.chain.Tip(), newround)
	if err != nil {
		return fmt.Errorf("failed to draw new ticket: %v", err)
	}

	// (2.9) if the ticket grants us the right to propose, propose and broadcast the block
	// with the proof.
	if ticket.Propose {
		block := NewBlock(newround, e.chain.Tip(), ticket.Data, ticket.Proof, e.vrfPK)
		err = bw.Write(&Msg{Proposal: block})
		if err != nil {
			//@TODO log the failure of writing our proposal to the broadcast
		}

		atomic.AddUint64(&e.txMsg, 1)
	}

	// (2.10) if the ticket grants us the right to vote, create a voter for the new
	// round and start handling proposals. And start the BlockTime timer, whenever the
	// timer expires. Write a vote message to the broadcast.
	if ticket.Vote {
		e.voter = NewVoter(newround, e.chain, ticket, e.vrfPK)

		//schedule the voter to cast its votes after a configurable amount of time
		//this determines the pace of the network @TODO can we determine this value
		//by looking at the chain history
		if e.blockTime > 0 {
			time.AfterFunc(e.blockTime, func() {
				//@TODO protect by mutex, awefully race condition if we're setting the
				//the voter to nil. This function is called in its own go-routine
				if e.voter != nil {
					e.voter.Cast(bw)
				}
			})
		}

		//@TODO pass the broadcast writer to allow the voter to emit votes
	} else {
		e.voter = nil
	}

	return
}

// HandleProposal is called when a block proposal comes along. This might be
// potentially hours after the actual message arriving out-of-order. @TODO
// this implementation is currently full of race conditions and should not be
// called concurrently with any other method
func (e *Engine) HandleProposal(b *Block, bw BroadcastWriter) (err error) {
	// (2.0) @TODO do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.2) if the block's round is equal to the round of our current tip+1 we will
	// relay the proposal. If not, discard the proposal.
	tipb := e.chain.Read(e.chain.Tip())
	if tipb == nil {
		panic("failed to read tip block")
	}

	if b.Round != tipb.Round+1 {
		return nil //discard
	}

	err = bw.Write(&Msg{Proposal: b})
	if err != nil {
		//@TODO log failure to relay
	}

	// (2.3) if we are not a voter, we can simply stop here and return.
	if e.voter == nil {
		return nil
	}

	// (2.4) if we are a voter, check if the proposed block is of the round we are
	// voting for. If its not we stop
	ok, _ := e.voter.Verify(b)
	if !ok {
		return nil
	}

	// (2.5) Add the proposed block to the voters current votes. It will be written
	// to the broadcast on the next timer expiration
	_, _ = e.voter.Propose(b)

	return
}
