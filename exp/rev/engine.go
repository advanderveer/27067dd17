package rev

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
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
	idn    *Identity
	gen    *Proposal
	chain  *Chain
}

// NewEngine creates the engine
func NewEngine(logw io.Writer, idn *Identity, bc Broadcast, c *Chain) (e *Engine) {
	e = &Engine{
		rounds: make(map[uint64]*Round),
		latest: 0,
		bc:     bc,
		done:   make(chan struct{}, 1),
		logs:   log.New(logw, "", 0),
		idn:    idn,
		res:    make(chan *HandleResult, 1),
		chain:  c,
	}

	//genesis block and proposal
	genb := &Block{}
	e.chain.Append(genb, big.NewInt(0))
	e.gen = &Proposal{Token: make([]byte, 32), Block: genb}
	e.rounds[0] = NewRound(e.chain, e.gen)

	//out-of-order buffer, genesis is always already handled
	e.ooo = NewOutOfOrder(e, e.gen)

	go func() {
		for {
			msg := &Msg{}
			err := e.bc.Read(msg)
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from broadcast, shutting down message handling at round %d", e.idn, e.latest)
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
			//@TODO (optimization) run handle concurrently if ooo is thread safe
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
	//@TODO this looks ugly, is it a race condition?
	if len(e.res) > 0 {
		<-e.res
	}
	res := &HandleResult{}
	defer func() { e.res <- res }()

	//validate basic syntax and round proof
	ok, err := p.Validate()
	if !ok {
		res.ValidationErr = err
		e.logs.Printf("[INFO][%s] invalid proposal received, ignore it: %v", e.idn, err)
		return
	}

	//check if the proposal is not working a round that is too far off
	prior := e.rounds[p.Round-1]
	if prior == nil {
		res.WitnessRoundTooFarOff = true
		e.logs.Printf("[INFO][%s] received proposal with witness for round %d but its too far off our current round: %d, ignore it", e.idn, p.Round-1, e.latest)
		return
	}

	//validate the witness given the tip the other member is proposing the build on
	ok, err = prior.Question(p.Block.Prev, p.Witness)
	if !ok {
		res.InvalidWitnessErr = err
		e.logs.Printf("[INFO][%s] proposal %s witness was invalid: %v", e.idn, p, err)
		return
	}

	//check if we receive a proposal for the next round. If so move our cursor
	if p.Round > e.latest && (p.Round-e.latest == 1) {
		e.logs.Printf("[INFO][%s] received valid proposal %s for round %d while our latest is %d: enterering new round", e.idn, p, p.Round, e.latest)
		res.OtherEnteredNewRound = true
		e.rounds[p.Round] = NewRound(e.chain)
		e.latest = p.Round //move our latest cursor

		//@TODO (optimization) remove old rounds to finalize
	}

	//get the round this proposal is for, might just be created. This should now
	//always exist because we check for witness round first and created a new one
	//if its 1 in the future
	curr := e.rounds[p.Round]
	if curr == nil {
		e.logs.Printf("[ERRO][%s] received proposal for round %d, but we have no records for it, skip it", e.idn, p.Round)
		return
	}

	//@TODO (optimization) check if a user didn't already make a proposal, do not
	//relay a second proposal from the same user.

	//@TODO (optimization) limit how much proposals we accept per round? The top N
	//for example and only relay if it was relevant at all?

	//observe the proposal for the current round
	_, newp, witness, prev := curr.Observe(p)
	if witness != nil {

		//create a new proposal
		newp := e.idn.CreateProposal(p.Round + 1)
		newp.Block = &Block{Prev: prev, Data: []byte(fmt.Sprintf("%d: %.6x", newp.Round, e.idn.pk))}
		newp.Witness = witness

		//@TODO (optimization) handle our own proposal right away, loopback to ourselves?
		//@TODO (optimization) only relay if the newp.Token grants us this

		e.logs.Printf("[INFO][%s] created proposal %s for round %d with %d witnesses", e.idn, newp, newp.Round, len(newp.Witness))

		//broadcast our new proposal to the network
		err = e.bc.Write(&Msg{Proposal: newp})
		if err != nil {
			e.logs.Printf("[ERRO][%s] failed to broadcast our new proposal: %v", e.idn, err)
		}
	}

	//if the proposal didn't rank higher then our existing proposal don't boter
	//relaying it
	if !newp {
		return
	}

	//append the block that is encoded in the proposal to our chain with with
	//the weight that corresponds to the proposal token rank.
	e.chain.Append(p.Block, p.Rank())

	//relay the proposal
	err = e.bc.Write(&Msg{Proposal: p})
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to relay proposal: %v", e.idn, err)
	}

	res.Relayed = true
	return
}
