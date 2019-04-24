package engine

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/advanderveer/27067dd17/onl"
)

// Engine reads messages from the broadcast and advances through rounds
type Engine struct {
	bc    Broadcast
	logs  *log.Logger
	pulse Pulse
	idn   *onl.Identity
	done  chan struct{}
	chain *onl.Chain
	ooo   *OutOfOrder
	round uint64
	maxw  int

	mempool   map[onl.WID]*onl.Write
	mempoolmu sync.RWMutex
}

// New initiates an engine
func New(logw io.Writer, bc Broadcast, p Pulse, idn *onl.Identity, c *onl.Chain) (e *Engine) {
	e = &Engine{
		idn:   idn,
		bc:    bc,
		pulse: p,
		done:  make(chan struct{}, 2),
		logs:  log.New(logw, "", 0),
		chain: c,
		round: 1,
		maxw:  10, //@TODO make configurable, or measure total block size in MiB

		//for inspiration about mempool handling in bitcoin:
		//https://blog.kaiko.com/an-in-depth-guide-into-how-the-mempool-works-c758b781c608
		mempool: make(map[onl.WID]*onl.Write),
	}

	//setup out of order buffer
	e.ooo = NewOutOfOrder(e)

	//round switching
	go func() {
		clock := onl.NewWallClock()
		genesis := e.chain.Genesis().Hash()

		for {
			err := e.pulse.Next()
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from pulse, shutting down pulse handling at round %d", e.idn, e.round)
				break
			}

			//increment round
			round := atomic.AddUint64(&e.round, 1)

			//handle round
			e.handleRound(clock, genesis, round)
		}

		e.done <- struct{}{}
	}()

	//message handling
	go func() {
		for {
			msg := &Msg{}
			err := e.bc.Read(msg)
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from broadcast, shutting down message handling at round %d", e.idn, e.round)
				break //were done here
			} else if err != nil {
				e.logs.Printf("[ERRO] failed to read message from broadcast, shutting down: %v", err)
				break //shutting down
			}

			//handle out-of-order
			e.ooo.Handle(msg)
		}

		e.done <- struct{}{} //indicate we've closed down message handling
	}()

	return e
}

//Handle a single message, assumes that is called in-order
func (e *Engine) Handle(msg *Msg) {
	if msg.Write != nil {
		e.handleWrite(msg.Write)
	} else if msg.Block != nil {
		e.handleBlock(msg.Block)
	} else {
		e.logs.Printf("[INFO][%s] read messages that is neither a write or a block, ignoring", e.idn)
		return
	}
}

func (e *Engine) handleRound(clock onl.Clock, genesis onl.ID, round uint64) {

	//read tip and current state from chain
	tip := e.chain.Tip()
	state, err := e.chain.State(tip)
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to rebuild state for round %d: %v", e.idn, round, err)
		return
	}

	//mint a block for our current tip
	b := e.idn.Mint(clock, tip, genesis, round)

	//apply random writes from the mempool, if the work include them
	e.mempoolmu.RLock()
	for _, w := range e.mempool {
		err = state.Apply(w, false)
		if err != nil {
			continue
		}

		b.AppendWrite(w)
		if len(b.Writes) >= e.maxw {
			break
		}
	}
	e.mempoolmu.RUnlock()

	//sign the block
	e.idn.Sign(b)

	//append to our own chain first
	err = e.chain.Append(b)
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to append our own block at round %d: %v", e.idn, round, err)
		return
	}

	//broadcast our block
	err = e.bc.Write(&Msg{Block: b})
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to broadcast new block to peers: %v", e.idn, err)
	}
}

func (e *Engine) handleWrite(w *onl.Write) {
	//@TODO validate syntax/signature

	e.mempoolmu.Lock()
	defer e.mempoolmu.Unlock()

	//check if we already have a write in the mempool
	wid := w.Hash()
	_, ok := e.mempool[wid]
	if ok {
		return
	}

	//@TODO check if the write is already in the heaviest tip chain, if so reject

	//add to mempool
	e.mempool[wid] = w

	//relay to peers
	err := e.bc.Write(&Msg{Write: w})
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to relay write to peers: %v", e.idn, err)
	}
}

func (e *Engine) handleBlock(b *onl.Block) {

	//check if the block is for a round we didn't reach yet
	round := atomic.LoadUint64(&e.round)
	if b.Round > round {
		e.logs.Printf("[INFO][%s] received block from round %d while we're only at round %d, skipping", e.idn, b.Round, round)
		return
	}

	//append the block to the chain, any invalid blocks will be rejected here
	err := e.chain.Append(b)
	if err != nil {
		e.logs.Printf("[INFO][%s] failed to append incoming block: %v", e.idn, err)
		return
	}

	//@TODO remove all writes from mempool if the block was finalized
	//@TODO remove all conflicting tx from mempool if the block is finalized

	//handle any messages that were waiting on this block
	e.ooo.Resolve(b.Hash())

	//relay to peers
	err = e.bc.Write(&Msg{Block: b})
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to relay block to peers: %v", e.idn, err)
	}
}

// Shutdown will gracefully shutdown the engine. It will ask subsystems to close
// gracefully first and wait for them before. If the context expires first its
// error is returned.
func (e *Engine) Shutdown(ctx context.Context) (err error) {
	err = e.bc.Close()
	if err != nil {
		return fmt.Errorf("failed to close broadcast: %v", err)
	}

	err = e.pulse.Close()
	if err != nil {
		return fmt.Errorf("failed to close pulse: %v", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done: //1th subystem
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.done: //2th subsystem
			return nil
		}
	}
}
