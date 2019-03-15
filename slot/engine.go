package slot

import (
	"fmt"
	"io"
	"log"
	"sync"
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
	chain *Chain      //holds state for majority voted blocks
	voter *Voter      //hold our voting state (if we have the right)
	ooo   *OutOfOrder //out-of-order message handling and storage

	logs *log.Logger
	mu   sync.RWMutex
}

// NewEngine sets up the engine
func NewEngine(logw io.Writer, vrfpk []byte, vrfsk *[vrf.SecretKeySize]byte, bc Broadcast, bt time.Duration, minVotes uint64) (e *Engine) {
	e = &Engine{
		vrfSK: vrfsk,
		vrfPK: vrfpk,
		rxMsg: 0,
		txMsg: 0,
		chain: NewChain(),

		blockTime: bt,
		minVotes:  minVotes,
		logs:      log.New(logw, fmt.Sprintf("%s: ", PKString(vrfpk)), 0),
	}

	//keep transmission metrics by wrapping
	e.bc = &metrics{
		bc:  bc,
		txf: func() { atomic.AddUint64(&e.txMsg, 1) },
		rxf: func() { atomic.AddUint64(&e.rxMsg, 1) },
	}

	e.ooo = NewOutOfOrder(e.bc, e.chain, e.Handle)
	return
}

// Chain return the block chain used by the engine
func (e *Engine) Chain() *Chain {
	return e.chain
}

// Stats returns statistics about the engine
func (e *Engine) Stats() (rx, tx uint64, votes map[ID]*Block) {
	rx = atomic.LoadUint64(&e.rxMsg)
	tx = atomic.LoadUint64(&e.txMsg)
	e.mu.RLock()
	defer e.mu.RUnlock()

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
	for {
		curr := &Msg{}

		err = e.bc.Read(curr)
		if err == io.EOF {
			return ErrBroadcastClosed
		} else if err != nil {
			return MsgError{T: curr.Type(), N: atomic.LoadUint64(&e.rxMsg) + 1, E: err, M: "read message from broadcast"}
		}

		err := e.ooo.Handle(curr)
		if err != nil {
			return MsgError{T: curr.Type(), N: atomic.LoadUint64(&e.rxMsg), E: err, M: "handle rx message"}
		}
	}
}

// Handle a single message, messages may arrive in any order.
func (e *Engine) Handle(in *Msg, bw BroadcastWriter) (err error) {
	switch in.Type() {
	case MsgTypeVote: //vote message
		return e.HandleVote(in.Vote)
	case MsgTypeProposal: //proposal message
		return e.HandleProposal(in.Proposal)
	default:
		return ErrUnknownMessage
	}
}

// HandleVote is called when a block with vote comes along. This might be
// called out-of-order so potentially hours after the message actually arriving
// on the machine.
func (e *Engine) HandleVote(v *Vote) (err error) {
	e.logs.Printf("[TRAC] %s: start handling", v)
	e.mu.Lock()
	defer e.mu.Unlock()

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
		e.logs.Printf("[INFO] failed to verify %s: %v", v, err)
		return nil
	}

	// (2.4) at this point it is ok to relay the vote message, broadcast
	// will take care of message deduplication.
	e.logs.Printf("[TRAC] verified %s: relaying", v)
	err = e.bc.Write(&Msg{Vote: v})
	if err != nil {
		e.logs.Printf("[ERRO] failed to relay %s", v)
	}

	// (2.5) add vote to a counter of unique votes for that block. If
	// counter already reached the threshold value we we can skip the rest and return.
	// @TODO how do we prevent the voting count to grow unbounded:
	// - an unhonest voter might sign and vote on any block proposal it sees.
	// - an unhonest voter might keep signing blocks in a round that are created
	//   ad infinitum by a proposer that is also under its control for that round
	//   @TODO how to prevent this?: Every proposer can propose only one block per round
	nvotes := e.chain.Tally(v)
	e.logs.Printf("[TRAC] tallied %s, number of votes: %d", v, nvotes)

	// (2.6) if the counter passed the vote threshold we can now append the block to our chain.
	// and resolve any out of order handle calls that are waiting
	// @TODO can the minvotes be based on chain history?
	if uint64(nvotes) <= e.minVotes {
		e.logs.Printf("[TRAC] %s doesn't cause enough votes (%d<%d): no progress", v, nvotes, e.minVotes+1)
		return nil //not enough votes (yet), the block in this vote is worth nothing (yet)
	}

	e.logs.Printf("[TRAC] %s caused enough votes (%d>%d), progress!", v, nvotes, e.minVotes)

	//resolve out-of-order blocks, messages can now be handled
	err = e.ooo.Resolve(v.Block)
	if err != nil {
		e.logs.Printf("[ERRO] failed to resolve out-of-order messages: %v", err)
	}

	// (2.7) if we are a voter and the block is of the same round as we're voting for
	// for we will stop casting voting
	if e.voter != nil && v.Block.Round >= e.voter.round {

		//send any open votes onto the network and shutdown
		votes := e.voter.Cast(e.bc)
		e.logs.Printf("[TRAC] %s while voter was active, casted remaining %d votes before teardown", v, len(votes))

		//setting this to nil
		e.voter = nil
	}

	//append the block in the vote and see if it changes the tip
	e.logs.Printf("[TRAC] %s caused enough votes, appending it's block to chain!", v)
	_, newtip := e.chain.Append(v.Block)
	if !newtip {
		e.logs.Printf("[TRAC] %s has caused no new tip: do not progress to next round", v)
		return nil //tip didn't change nothing left to do
	}

	e.logs.Printf("[TRAC] %s has caused a new tip: progress to next round", v)
	return e.WorkNewTip() //enter the next round
}

// WorkNewTip is called when a vote caused the member to experience a
// new tip.
func (e *Engine) WorkNewTip() (err error) {

	// (2.8) if the tip changed we can now draw another ticket.
	tipb := e.chain.Read(e.chain.Tip())
	if tipb == nil {
		panic("new tip block doesn't exist")
	}

	//@TODO for which round do we want to draw? always tip round +1? Can the system
	//get stuck in a certain round on a certain tip?
	newround := tipb.Round + 1
	e.logs.Printf("[TRAC] draw ticket with new tip '%s' as round %d", BlockName(e.chain.Tip()), newround)

	ticket, err := e.chain.Draw(e.vrfPK, e.vrfSK, e.chain.Tip(), newround)
	if err != nil {
		return fmt.Errorf("failed to draw new ticket: %v", err)
	}

	// (2.9) if the ticket grants us the right to propose, propose and broadcast the block
	// with the proof.
	if ticket.Propose {
		block := NewBlock(newround, e.chain.Tip(), ticket.Data, ticket.Proof, e.vrfPK)

		e.logs.Printf("[TRAC] drew proposer ticket! proposing block '%s'", BlockName(block.Hash()))
		err = e.bc.Write(&Msg{Proposal: block})
		if err != nil {
			e.logs.Printf("[ERRO] failed to broadcast %s", block)
		}
	}

	// (2.10) if the ticket grants us the right to vote, create a voter for the new
	// round and start handling proposals. And start the BlockTime timer, whenever the
	// timer expires. Write a vote message to the broadcast.
	if ticket.Vote {
		e.voter = NewVoter(e.logs.Writer(), newround, e.chain, ticket, e.vrfPK)
		e.logs.Printf("[TRAC] drew voter ticket! setup voter for round %d", e.voter.round)

		//schedule the voter to cast its votes after a configurable amount of time
		//this determines the pace of the network @TODO can we determine this value
		//by looking at the chain history
		if e.blockTime > 0 {
			e.logs.Printf("[TRAC] blocktime is higher then zero, schedule vote casting in %s", e.blockTime)
			time.AfterFunc(e.blockTime, func() {
				e.mu.RLock()
				defer e.mu.RUnlock()

				if e.voter != nil {
					votes := e.voter.Cast(e.bc)
					e.logs.Printf("[TRAC] blocktime has passed, and we are still voter, casted %d votes", len(votes))
				} else {
					e.logs.Printf("[TRSC] blocktime has passed but we are no longer a voter")
				}
			})
		}
	} else {
		e.logs.Printf("[TRAC] drew no voter ticket, tearing down previous voter of round %d", e.voter.round)
		e.voter = nil
	}

	return
}

// HandleProposal is called when a block proposal comes along. This might be
// potentially hours after the actual message arriving out-of-order.
func (e *Engine) HandleProposal(b *Block) (err error) {
	e.logs.Printf("[TRAC] %s: start handling", b)
	e.mu.Lock()
	defer e.mu.Unlock()

	// (2.0) @TODO do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.2) if the block's round is equal to the round of our current tip+1 we will
	// relay the proposal. If not, discard the proposal.
	tipb := e.chain.Read(e.chain.Tip())
	if tipb == nil {
		e.logs.Printf("[ERRO] failed to read tip block, cannot progress securely: %v", err)
		return nil
	}

	if b.Round != tipb.Round+1 {
		e.logs.Printf("[INFO] %s is of round %d, we are in round %d: discarding", b, b.Round, tipb.Round+1)
		return nil //discard
	}

	e.logs.Printf("[TRAC] %s was verified and of the correct round: relaying", b)
	err = e.bc.Write(&Msg{Proposal: b})
	if err != nil {
		e.logs.Printf("[ERRO] %s failed to relay: %v", b, err)
	}

	// (2.3) if we are not a voter, we can simply stop here and return.
	if e.voter == nil {
		e.logs.Printf("[TRAC] %s is not handled further, we are not a voter", b)
		return nil
	}

	// (2.4) if we are a voter, check if the proposed block is of the round we are
	// voting for. If its not we stop
	ok, _ := e.voter.Verify(b)
	if !ok {
		e.logs.Printf("[INFO] %s isn't considered for voting, failed to verify: %v", b, err)
		return nil
	}

	// (2.5) Add the proposed block to the voters current votes. It will be written
	// to the broadcast on the next timer expiration
	newhigh, _ := e.voter.Propose(b)
	if newhigh {
		e.logs.Printf("[TRAC] %s is new highest ranking block for next vote casting", b)
		return
	}

	e.logs.Printf("[TRAC] %s was considered but is not the new higest ranking proposal", b)
	return
}

//metrics wraps a broadcast and calls rxf for every read and txf for every write
type metrics struct {
	bc  Broadcast
	txf func()
	rxf func()
}

func (m *metrics) Write(msg *Msg) (err error) {
	err = m.bc.Write(msg)
	if err == nil {
		m.txf()
	}

	return
}

func (m *metrics) Read(msg *Msg) (err error) {
	err = m.bc.Read(msg)
	if err == nil {
		m.rxf()
	}

	return
}
