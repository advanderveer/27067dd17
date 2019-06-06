package ttl

import (
	"io"
	"math/big"
	"time"
)

type Msg struct {
	Round uint64
	Prio  [32]byte
}

// Round captures block proposals and closes after a deadline when no more
// messages are received
type Round struct {
	bc   Broadcast
	num  uint64
	prev *Round
	top  *big.Int
	msgs chan *Msg
	done chan *Round
}

func NewRound(bc Broadcast, num uint64) (r *Round) {
	r = &Round{
		bc:   bc,
		num:  num,
		top:  big.NewInt(0),
		msgs: make(chan *Msg),
		done: make(chan *Round),
	}

	//read messages from broadcast
	go func() {
		for {
			msg := &Msg{}

			err := r.bc.Read(msg)
			if err == io.EOF {
				panic("broadcast closed")
			} else if err != nil {
				panic("failed to read broadcast: " + err.Error())
			}

			r.msgs <- msg
		}
	}()

	return
}

// @TODO proof that what you're proposing for has a majority vote? Can we enforce
// a single proposal per round? what if the tip switches?

//
// // Seed returns the seed of this round
// func (r *Round) Seed() [32]byte {
// 	//@TODO add read mutex
// 	return r.seed
// }

// Done returns a channel that completes when this round ends and results in
// a new round
func (r *Round) Done() <-chan *Round {
	//@TODO return a committer listener that gathers votes for committing the round
	return r.done
}

// Start the round
func (r *Round) Start(to time.Duration) {
	//@TODO draw a ticket based on the current round nr only
	//@TODO set the draw immediately as the highest and write to broadcast

	timer := time.NewTimer(to)
	defer timer.Stop()

	for {
		select {
		case <-timer.C: // expires (timeout)
			r.done <- NewRound(r.bc, r.num+1)
			return

		case msg := <-r.msgs:

			//@TODO if the round nr is not equal this this round: discard. Other member
			//is too fast or too slow: all non-faulty members will ignore it.

			// if our prev block is higher in rank than the other members prev block
			// we ignore the message (the other member is out-of-date). If the provided
			// prev block is higher in rank then ours we are out-of-date.

			//@TODO verify vrf proof

			//check if the prio in this msg is better then current top
			//@TODO check if total strength of the chain is better
			prio := big.NewInt(0)
			prio.SetBytes(msg.Prio[:])
			if prio.Cmp(r.top) <= 0 {
				continue //no new prio, do not relay anything
				//@TODO relay our highest instead, or schedule to do so?
			}

			//if the timer expired while we were comparing, drain it first
			if !timer.Stop() {
				<-timer.C
			}

			//reset the timer
			timer.Reset(to)

			//set our new highest value
			r.top = prio

			//@TODO relay the newest highest
		}
	}
}
