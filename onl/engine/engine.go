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
	bc      Broadcast
	logs    *log.Logger
	pulse   Pulse
	idn     *onl.Identity
	done    chan struct{}
	chain   *onl.Chain
	ooo     *OutOfOrder
	round   uint64
	maxw    int
	genesis onl.ID

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

	//genesis is kept for resolving purposes
	e.genesis = e.chain.Genesis().Hash()

	//setup out of order buffer, genesis is always marked as resolved
	e.ooo = NewOutOfOrder(e)
	e.ooo.Resolve(e.genesis)

	//@TODO make it possible for a member to catch up to the correct round of the network
	//@TODO start a process that allows out of sync members to catch up with missing blocks

	//round switching
	go func() {
		clock := onl.NewWallClock()

		for {
			err := e.pulse.Next()
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from pulse, shutting down pulse handling at round %d", e.idn, e.round)
				break
			}

			//increment round
			round := atomic.AddUint64(&e.round, 1)

			//handle round
			e.handleRound(clock, e.genesis, round)

			//@TODO enable random syncing of old blocks
		}

		e.done <- struct{}{}
	}()

	//message handling
	go func() {
		for {
			msg := &Msg{}
			err := e.bc.Read(msg)
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from broadcast, shutting down message handling", e.idn)
				break //were done here
			} else if err != nil {
				e.logs.Printf("[ERRO] failed to read message from broadcast, shutting down: %v", err)
				break //shutting down
			}

			//handle out-of-order
			//@TODO (#3) out-of-order needs to be thread safe
			e.ooo.Handle(msg)
		}

		e.done <- struct{}{} //indicate we've closed down message handling
	}()

	return e
}

// View will read the chain's state
func (e *Engine) View(f func(kv *onl.KV)) (err error) {
	//@TODO take some handle to watch for
	e.chain.View(f)
	return
}

// Update will submit a change the key-value state by, it returns when the change
// was submitted and ended up in the longest chain.
func (e *Engine) Update(ctx context.Context, f func(kv *onl.KV)) (err error) {
	w := e.chain.Update(f)
	if w == nil {
		return nil //no changes, "succeeds" immediately
	}

	//handle our own write
	e.handleWrite(w)

	//@TODO wait for write to be accepted or context to expire
	//@TODO return some hash/handle that a call to read/watch can wait on
	//@TODO allow configurable certainty on network consensus per handle

	return
}

//Round returns the current round the engine is on
func (e *Engine) Round() uint64 {
	return atomic.LoadUint64(&e.round)
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

	//@TODO (#1) sometimes the newly appointed tip doesn't exist yet? probably because
	//the tip is not part of the storage transaction (wich takes longer). Fix that
	//instead of waiting here with an ugly hack
	for {
		_, _, err := e.chain.Read(tip)
		if err == nil {
			break
		}
	}

	fmt.Printf("%s round %d, tip round: %d\n", e.idn, round, tip.Round())
	state, err := e.chain.State(tip)
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to rebuild state for round %d: %v", e.idn, round, err)
		return
	}

	//check if we have stake in the heaviest tip state
	var stake uint64
	state.View(func(kv *onl.KV) {
		stake, _ = kv.ReadStake(e.idn.PK())
	})

	if stake < 1 {
		return //no stake, no proposing for us
	}

	//mint a block for our current tip
	b := e.idn.Mint(clock, tip, genesis, round)

	//try to apply random writes from the mempool, if they work include until max is reached
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

	//@TODO are empty blocks allowed? what is the incentive of adding writes?

	//sign the block
	e.idn.Sign(b)

	//handle the block as if we received it
	e.handleBlock(b)
}

func (e *Engine) handleWrite(w *onl.Write) {
	//@TODO validate syntax/signature

	e.mempoolmu.Lock()
	defer e.mempoolmu.Unlock()

	//check if we already have the write in our mempool
	wid := w.Hash()
	_, ok := e.mempool[wid]
	if ok {
		return
	}

	//@TODO check if the write is already in our heaviest tip chain, if so reject

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
	round := e.Round()
	if b.Round > round {
		e.logs.Printf("[INFO][%s] received block from round %d while we're at %d, skipping", e.idn, b.Round, round)
		return
	}

	//append the block to the chain, any invalid blocks will be rejected here
	err := e.chain.Append(b)
	if err != nil {
		if err == onl.ErrBlockExist {
			return //nothing too see really
		}

		e.logs.Printf("[INFO][%s] failed to append incoming block: %v", e.idn, err)
		return
	}

	//@TODO remove all writes from mempool if the block was finalized or if block
	//became one round old and part of our heaviest chain
	//@TODO remove all conflicting writes from mempool if the block is finalized

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
