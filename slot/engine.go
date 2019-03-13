package slot

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/advanderveer/27067dd17/vrf"
)

// Engine manages the core message handling logic and always corresponds to
// one identity in the system.
type Engine struct {
	vrfSK *[vrf.SecretKeySize]byte //private key for verifiable random function
	vrfPK []byte                   //public key for verifiable random function
	rxMsg uint64                   //received message count
	txMsg uint64                   //transmit message count

	chain  *Chain //holds state for voted blocks
	notary *Voter //hold our voting state (if we have the right)
}

// NewEngine sets up the engine
func NewEngine(vrfpk []byte, vrfsk *[vrf.SecretKeySize]byte) (e *Engine) {
	e = &Engine{
		vrfSK: vrfsk,
		vrfPK: vrfpk,
		rxMsg: 0,
		txMsg: 0,
		chain: NewChain(),
	}

	return
}

// Stats returns statistics about the engine
func (e *Engine) Stats() (rx, tx uint64) {
	rx = atomic.LoadUint64(&e.rxMsg)
	tx = atomic.LoadUint64(&e.txMsg)
	return
}

// Run will keep reading messages from the broadcast layer and write new
// messages to it.
func (e *Engine) Run(bc Broadcast) (err error) {
	ooo := NewOutOfOrder(bc, e.chain, e.Handle)

	curr := &Msg{}
	for {
		err = bc.Read(curr)
		if err == io.EOF {
			return ErrBroadcastClosed
		} else if err != nil {
			return MsgError{T: curr.Type(), N: atomic.LoadUint64(&e.rxMsg) + 1, E: err, M: "read message from broadcast"}
		}

		n := atomic.AddUint64(&e.rxMsg, 1)
		err := ooo.Handle(curr)
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
// on the machine
func (e *Engine) HandleVote(b *Block, bw BroadcastWriter) (err error) {
	// (2.0) do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.2) verify vote signature and threshold, if invalid discard message. No
	// relay or state changes from this message.
	// (2.3) verify proposal signature, if invalid, discard message. No relay or state
	// change. it will only pollute the network.
	// - Verify if it a vote at all
	// - Verify that the proposer pk didn't already propose a block @TODO what if
	//   others use another pk then there own?
	ok, err := e.chain.Verify(b)
	if !ok {
		_ = err    //@TODO log verification errors
		return nil //not a valid block
	}

	// (2.4) at this point it is ok to relay the vote message, broadcast
	// will take care of message deduplication.
	err = bw.Write(&Msg{Vote: b})
	if err != nil {
		return fmt.Errorf("failed to relay notarized block: %v", err)
	}

	// (2.5) add vote to a counter of unique votes for that block. If
	// counter already reached the threshold value we we can skip the rest and return.
	// @TODO how do we prevent the voting count to grow unbounded:
	// - an unhonest voter might sign and vote on any block proposal it sees.
	// - an unhonest voter might keep signing blocks in a round that are created
	//   ad infinitum by a proposer that is also under its control for that round
	//   @TODO how to prevent this?: Every proposer can propose only one block per round

	// (2.6) if the counter passed the vote threshold we can now append the block to our chain.
	// and resolve any out of order handle calls that are waiting

	// (2.7) if we are a voter and the block is of the same round as we're voting for
	// for we will stop casting voting

	// (2.8) if the tip changed we can now draw another ticket.

	// (2.9) if the ticket grants us the right to propose, propose and broadcast the block
	// with the proof.

	// (2.10) if the ticket grants us the right to vote, create a voter for the new
	// round and start handling proposals. And start the BlockTime timer, whenever the
	// timer expires. Write a vote message to the broadcast.

	return
}

// HandleProposal is called when a block proposal comes along. This might be
// potentially hours after the actual message arriving out-of-order.
func (e *Engine) HandleProposal(b *Block, bw BroadcastWriter) (err error) {
	// (2.0) do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.2) if the block's round is equal to the round of our current tip+1 we will
	// relay the proposal. If not, discard the proposal.

	// (2.3) if we are not a voter, we can simply stop here and return.

	// (2.4) if we are a voter, check if the proposed block is of the round we are
	// voting for. If its not, discard (@TODO check if that is effectively the
	// the same check as 2.2)

	// (2.5) Add the proposed block to the voters current votes. It will be written
	// to the broadcast on the next timer expiration

	return
}
