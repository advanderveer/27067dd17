package slot

import (
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

	chain  *Chain  //blockchain state
	notary *Notary //if we are a notary this holds our state
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

	curr := &Msg{}
	for {
		err = bc.Read(curr)
		if err == io.EOF {
			return ErrBroadcastClosed
		} else if err != nil {
			return MsgError{T: curr.Type(), N: atomic.LoadUint64(&e.rxMsg) + 1, E: err, M: "read message from broadcast"}
		}

		n := atomic.AddUint64(&e.rxMsg, 1)
		out, err := e.Handle(curr)
		if err != nil {
			return MsgError{T: curr.Type(), N: n, E: err, M: "handle rx message"}
		}

		for _, o := range out {
			err = bc.Write(o)
			if err != nil {
				return MsgError{T: curr.Type(), N: n, E: err, M: "write tx message"}
			}

			atomic.AddUint64(&e.txMsg, 1)
		}
	}
}

// Handle a single message, messages may arrive in any order.
func (e *Engine) Handle(in *Msg) (out []*Msg, err error) {
	switch in.Type() {
	case MsgTypeNotarized: //notarization message
		return e.handleNotarized(in.Notarized)
	case MsgTypeProposal: //proposal message
		return e.handleProposal(in.Proposal)
	default:
		return nil, ErrUnknownMessage
	}
}

func (e *Engine) handleNotarized(b *Block) (out []*Msg, err error) {
	// (2.0) do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.1) if previous block is not known, store out of order. When ancestory
	// is completed retry handling of the block.

	// (2.2) verify notarization signature and threshold, if invalid discard message. No
	// relay or state changes from this message.

	// (2.3) verify proposal signature, if invalid, discard message. No relay or state
	// change. it will only pollute the network.

	// (2.4) at this point it is ok to relay the notarization message, append to out
	// broadcast will take care of message deduplication (CRC check?) to prevent
	// infinite relay

	// (2.5) add to notarization counter for the block. If counter already reached the
	// threshold value we don't need we can skip the rest and return.

	// (2.6) if we are a notary and the block is of the same round as we're notarizing
	// for we will stop notarizing

	// (2.7) if the counter passed the nt threshold we can now append the block to our chain.
	// if the tip changed we can now draw another ticket.

	// (2.8) if the ticket grants us the right to propose, propose and broadcast the block
	// with the proof.

	// (2.9) if the ticket grants us the right to notarize, create a notary for the new
	// round and handling proposals. And start the BlockTime timer, whenever the
	// timer expires. Write a notarized message to the broadcast.

	return
}

func (e *Engine) handleProposal(b *Block) (out []*Msg, err error) {
	// (2.0) do basic syntax inspection, if any fields are missing or wrong size
	// discard right away.

	// (2.1) if the block arrived without its prev reference existing, store out of
	// order and defer handling until ancestory is complete

	// (2.2) if the block's round is equal to the round of our current tip+1 we will
	// relay the proposal. If not, discard the proposal.

	// (2.3) if we are not a notary, we can stop here.

	// (2.4) if we are a notary, check if the proposed block is of the round we are
	// notarizing. If its not, discard (@TODO check if that is effectively the
	// the same check as 2.2)

	// (2.5) Add the proposed block to the notaries current proposal. It will be written
	// to the broadcast on the next timer experiation

	return
}
