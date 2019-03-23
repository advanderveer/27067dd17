package rev

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
)

// Engine reads messages one-by-one from the broadcast and moves through the rounds
// by witnessing and emitting proposals.
type Engine struct {
	res    chan *HandleResult
	rounds map[uint64]*Round
	latest uint64
	bc     Broadcast
	ooo    Handler
	mu     sync.RWMutex
	done   chan struct{}
	logs   *log.Logger
	gen    *Proposal
	chain  *Chain
}

// NewEngine creates the engine
func NewEngine(logw io.Writer, bc Broadcast, t int) (e *Engine) {
	e = &Engine{
		rounds: make(map[uint64]*Round),
		latest: 0,
		bc:     bc,
		done:   make(chan struct{}, 1),
		logs:   log.New(logw, "", 0),
		res:    make(chan *HandleResult, 1),
		chain:  NewChain(t),
	}

	//genesis block and proposal
	e.gen = &Proposal{Token: make([]byte, 32), Block: &Block{}}
	e.rounds[0] = NewRound(e.chain, e.gen)

	//out-of-order buffer, genesis is always already handled
	e.ooo = NewOutOfOrder(e, e.gen)

	go func() {
		for {
			msg := &Msg{}
			err := e.bc.Read(msg)
			if err == io.EOF {
				break //were done here
			} else if err != nil {
				e.logs.Printf("[ERRO] failed to read message from broadcast, shutting down: %v", err)
				break //shutting down
			}

			//handle different kind of messages
			if msg.Proposal == nil {
				e.logs.Printf("[INFO] received message without a proposal, ignoring it")
				continue //nothing to do
			}

			//handle the proposal, possibly out-of-order
			e.ooo.Handle(msg.Proposal)
		}

		close(e.done) //indicate we've closed down message handling
	}()

	return
}

// Genesis returns the only proposal in round 0
func (e *Engine) Genesis() *Proposal {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.gen
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

// Result returns a copy of the handle result of the nest or last handled proposal
func (e *Engine) Result() <-chan *HandleResult {
	return e.res
}

//HandleResult provides
type HandleResult struct {
	ValidationErr         error
	WitnessRoundTooFarOff bool
	InvalidWitnessErr     error
	OtherEnteredNewRound  bool
	Relayed               bool
}

// Handle a single proposal read from the broadcast
func (e *Engine) Handle(p *Proposal) {
	e.mu.Lock()
	defer e.mu.Unlock()

	//empty our result buffer if no-one read it
	if len(e.res) > 0 {
		<-e.res
	}
	res := &HandleResult{}
	defer func() { e.res <- res }()

	//validate basic syntax and round proof
	ok, err := p.Validate()
	if !ok {
		res.ValidationErr = err
		e.logs.Printf("[INFO] invalid proposal received, ignore it: %v", err)
		return
	}

	//check if the proposal is not working a round that is too far off
	prior := e.rounds[p.Round-1]
	if prior == nil {
		res.WitnessRoundTooFarOff = true
		e.logs.Printf("[INFO] received proposal with witness for round %d but its too far off our current round: %d, ignore it", p.Round-1, e.latest)
		return
	}

	//validate the witness given the tip the other member is proposing the build on
	ok, err = prior.Question(p.Block.Prev, p.Witness)
	if !ok {
		res.InvalidWitnessErr = err
		e.logs.Printf("[INFO] proposal witness was invalid: %v", err)
		return
	}

	//check if we receive a proposal for the next round
	if p.Round > e.latest && (p.Round-e.latest == 1) {
		res.OtherEnteredNewRound = true
		e.rounds[p.Round] = NewRound(e.chain)
		e.latest = p.Round //move our focus
		//@TODO (optimization) remove old rounds to finalize
	}

	//get the current round, might just be created. This should always exist because
	//we check for witness round first and created a new one if its 1 in the future
	curr := e.rounds[p.Round]
	if curr == nil {
		e.logs.Printf("[ERRO] received proposal for round %d, but we have no records for it, skip it", p.Round)
		return
	}

	//@TODO (optimization) check if a user didn't already make a proposal, do not
	//relay a second proposal from the same user

	//@TODO (optimization) limit how much proposals we accept per round? The top N
	//for example and only relay if it was relevant at all?

	//observe the proposal for the current round
	witness, prev := curr.Observe(p)
	if witness != nil {

		_ = prev
		//@TODO we can make a proposal for the next round (if our ticket allows us to)
		//@TODO open a new round if we haven't seen anyone else with a proposal for that round
		/// - the weight of our block will be that of the proposals priority but blocks
		//    will always try to build on blocks with the highest priority so there is
		//    no guarantee that it will be part of the longest chain
		//  - the block that is references by 'prev' needs to be part of the witnesses
		//    that we provide. Logically we want to build on the one that is the highest

		//@TODO we encode a block with 'prev' being the heaviest tip at the chain height
		//that corresponds to this round-1. Longest tip consensus.
	}

	//@TODO append the block that is encoded in the proposal to our chain with with
	//the weight that corresponds to the proposal token rank.

	//relay the proposal
	err = e.bc.Write(&Msg{Proposal: p})
	if err != nil {
		e.logs.Printf("[ERRO] failed to relay proposal: %v", err)
	}

	res.Relayed = true
	return
}
