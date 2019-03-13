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
}

// NewEngine sets up the engine
func NewEngine(vrfpk []byte, vrfsk *[vrf.SecretKeySize]byte) (e *Engine) {
	e = &Engine{
		vrfSK: vrfsk,
		vrfPK: vrfpk,
		rxMsg: 0,
		txMsg: 0,
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
			return MsgError{T: MsgTypeUnkown, N: 0, E: err, M: "read message from broadcast"}
		}

		n := atomic.AddUint64(&e.rxMsg, 1)
		typ := curr.Type()

		out, err := e.Handle(typ, curr)
		if err != nil {
			return MsgError{T: typ, N: n, E: err, M: "handle rx message"}
		}

		for _, o := range out {
			err = bc.Write(o)
			if err != nil {
				return MsgError{T: typ, N: n, E: err, M: "write tx message"}
			}

			atomic.AddUint64(&e.txMsg, 1)
		}
	}
}

// Handle a single message, messages may arrive in any order.
func (e *Engine) Handle(typ MsgType, in *Msg) (out []*Msg, err error) {
	switch typ {
	case MsgTypeNotarized: //notarization message
		return
	case MsgTypePropose: //proposal message
		return
	default:
		return nil, ErrUnknownMessage
	}
}
