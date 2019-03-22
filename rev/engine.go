package rev

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
)

//Round represents a time-bound segmentation of proposals. Members move through
//rounds such that old ones eventually get no messages and can be finalized.
type Round struct {
	proposals map[PID]*Proposal
}

//NewRound sets up a new round
func NewRound() (r *Round) {
	r = &Round{}

	//draw ticket from lottery
	return
}

// Witness a proposal for this round. With enough witnesses proposals we can
// propose a proposal ourselves for the next round.
func (r *Round) Witness(p *Proposal) {

	//add proposal to proof

	//check if we have enough/can solve puzzle

	//figure out how we can balance the witness functions such that it indicates
	//that the majority of members in the network get the time to read the message

	//if so, return ok

}

// Engine reads messages one-by-one from the broadcast and moves through the rounds
// by witnessing and emitting proposals.
type Engine struct {
	rounds map[uint64]Round
	latest uint64
	bc     Broadcast
	ooo    Handler
	mu     sync.RWMutex
	done   chan struct{}
	logs   *log.Logger
}

// NewEngine creates the engine
func NewEngine(logw io.Writer, bc Broadcast) (e *Engine) {
	e = &Engine{
		rounds: make(map[uint64]Round),
		latest: 0,
		bc:     bc,
		done:   make(chan struct{}, 1),
		logs:   log.New(logw, "engine: ", 0),
	}

	e.ooo = NewOutOfOrder(e)

	go func() {
		for {
			msg := &Msg{}
			err := e.bc.Read(msg)
			if err == io.EOF {
				break //were done here
			} else if err != nil {
				panic("failed to read message from broadcast: " + err.Error())
			}

			//handle different kind of messages
			if msg.Proposal == nil {
				e.logs.Printf("[ERRO] Received message without a proposal, ignore it")
				continue //nothing to do @TODO: report/log
			}

			//handle the proposal, possibly out-of-order
			e.ooo.Handle(msg.Proposal)
		}

		close(e.done) //indicate we've closed down message handling
	}()

	return
}

// Shutdown will gracefully shutdown the engine. It will close the broadcast
// endpoint first and wait until the remaining messages are handled.
func (e *Engine) Shutdown(ctx context.Context) (err error) {
	err = e.bc.Close()
	if err != nil {
		return fmt.Errorf("failed to close broadcast: %v", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return nil
	}
}

// Handle a single proposal read from the broadcast
func (e *Engine) Handle(p *Proposal) {
	e.mu.Lock()
	defer e.mu.Unlock()

	//@TODO what motivates a member to wait for the highest proposals? it can not
	//encode a block with a prev that has an higher weight then the heigest ranking
	//previous proposal that was witnessed. i.e the change is small that the block
	//will be included in the tip eventually. i.e. if you have to wait anyway, why
	//not send the highest ranking proposal, also gives you more options to solve the
	//puzzle. The weight of a block in the chain is determined by token of the proposal
	//it came with

	//if round is too far of the last round we've received a proposal of
	//discard it.

	//validate proposal
	// - syntax
	// - vrf token (proposal threshold)
	// - witness proof (we must have seen the proposals the other member has seen)
	//if it is invalid, discard. (unless we reference the genesis)

	//limit how much proposals we accept per round?

	//append the block that is encoded in the proposal to our chain. Or only
	//if it heavier then other blocks at that height? The weight of the block
	//is determined by the proposal token.

	//resolve out of order messages that were waiting for this proposal to come
	//in.

	//relay the proposal

	//send to round's bucket, if we've seen enough
	//proposals for a round we can propose for the
	//the round after it (if that is still relevant and
	//if we even drew a ticket that giving us this privilege).

	//we encode a block with 'prev' being the heaviest tip at the chain height
	//that corresponds to this round-1. Longest tip consensus

	//if its for a forward round we've not seen yet. move our round
	//"cursor" forward. this will effectively finalize old rounds

	return
}
