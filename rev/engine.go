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
	mu        sync.RWMutex
}

//NewRound sets up a new round
func NewRound(ps ...*Proposal) (r *Round) {
	r = &Round{proposals: make(map[PID]*Proposal)}
	for _, p := range ps {
		r.proposals[p.Hash()] = p
	}

	//draw ticket from lottery
	return
}

// Observe a proposal for this round. With enough observations we can Prove
// that we saw enough proposals and our conclusion about the top ranking one
// has merrit.
func (r *Round) Observe(p *Proposal) (witness PIDSet) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//add proposal to proof

	//check if we have enough/can solve puzzle

	//figure out how we can balance the witness functions such that it indicates
	//that the majority of members in the network get the time to read the message

	//if so, return ok
	return nil
}

// Question the witness and verify if its proof is valid for this round from
// our perspective
func (r *Round) Question(witness PIDSet) (ok bool, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	//check  if we know about the provided witness
	for wid := range witness {
		_, ok = r.proposals[wid]
		if !ok {
			break
		}
	}

	if !ok {
		return false, ErrProposalWitnessUnknown
	}

	//proposals must exist
	//must have solved puzzle
	//must be above threshold
	//must be min amount

	return true, nil
}

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
}

// NewEngine creates the engine
func NewEngine(logw io.Writer, bc Broadcast) (e *Engine) {
	e = &Engine{
		rounds: make(map[uint64]*Round),
		latest: 0,
		bc:     bc,
		done:   make(chan struct{}, 1),
		logs:   log.New(logw, "", 0),
		res:    make(chan *HandleResult, 1),
	}

	//genesis block and proposal
	e.gen = &Proposal{Token: make([]byte, 32)}
	e.rounds[0] = NewRound(e.gen)

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
	last := e.rounds[p.Round-1]
	if last == nil {
		res.WitnessRoundTooFarOff = true
		e.logs.Printf("[INFO] received proposal with witness for round %d but its too far off our current round: %d, ignore it", p.Round-1, e.latest)
		return
	}

	//validate the witness
	ok, err = last.Question(p.Witness)
	if !ok {
		res.InvalidWitnessErr = err
		e.logs.Printf("[INFO] proposal witness was invalid: %v", err)
		return
	}

	//@TODO validate that the prev block doesn't have a weight higher then what
	//any of what we've witnessed.

	//check if we receive a proposal for the next round
	if p.Round > e.latest && (p.Round-e.latest == 1) {
		res.OtherEnteredNewRound = true
		e.rounds[p.Round] = NewRound()
		e.latest = p.Round //move our focus
		//@TODO remove old rounds to finalize
	}

	//get the current round, might just be created. This should always exist because
	//we check for witness round first and created a new one if its 1 in the future
	curr := e.rounds[p.Round]
	if curr == nil {
		e.logs.Printf("[ERRO] received proposal for round %d, but we have no records for it, skip it", p.Round)
		return
	}

	//@TODO limit how much proposals we accept per round? The top N for example and
	//only relay if it was relevant at all? Or put otherwise, members can build on
	//a tip that has the top witness

	//observe the proposal for the current round
	witness := curr.Observe(p)
	if witness != nil {

		//@TODO we can make a proposal for the next round (if our ticket allows us to)
		//@TODO open it if it didn't exist yet
		//@TODO check if we didn't already make a proposal

		//@TODO we encode a block with 'prev' being the heaviest tip at the chain height
		//that corresponds to this round-1. Longest tip consensus.
	}

	//@TODO append the block that is encoded in the proposal to our chain. Or only
	//if it heavier then other blocks at that height? The weight of the block
	//is determined by the proposal token.

	//@TODO only relay if it is in our top N of the round so the network acts as
	//as a filter to not overwhelm it with proposals

	//relay the proposal
	err = e.bc.Write(&Msg{Proposal: p})
	if err != nil {
		e.logs.Printf("[ERRO] failed to relay proposal: %v", err)
	}

	res.Relayed = true
	return
}
