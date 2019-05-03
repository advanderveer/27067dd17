package engine

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine/sync"
)

//Clock provides an interface for synchronized rounds and reasonably accurate timestamps
type Clock interface {
	Round() (round uint64)
	Next() (round, ts uint64, err error)
	Close() (err error)
}

// Engine reads messages from the broadcast and advances through rounds
type Engine struct {
	bc    Broadcast
	clock Clock
	chain *onl.Chain
	ooo   *OutOfOrder
	pool  *MemPool

	logs    *log.Logger
	idn     *onl.Identity
	done    chan struct{}
	maxw    int
	genesis onl.ID
}

// New initiates an engine
func New(logw io.Writer, bc Broadcast, clock Clock, idn *onl.Identity, c *onl.Chain) (e *Engine) {
	e = &Engine{
		idn:   idn,
		bc:    bc,
		clock: clock,
		done:  make(chan struct{}, 2),
		logs:  log.New(logw, "", 0),
		chain: c,
		maxw:  10, //@TODO make configurable, or measure total block size in MiB

		pool: NewMemPool(),
	}

	//genesis is kept for resolving purposes
	e.genesis = e.chain.Genesis().Hash()

	//setup out of order buffer, genesis is always marked as resolved
	e.ooo = NewOutOfOrder(e)
	e.ooo.Resolve(e.genesis)

	//round progress
	go func() {
		for {
			round, ts, err := e.clock.Next()
			if err == io.EOF {
				e.logs.Printf("[INFO][%s] read EOF from pulse, shutting down pulse handling at round %d", e.idn, round)
				break
			}

			//handle round
			e.handleRound(round, ts)
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
			e.ooo.Handle(msg)
		}

		e.done <- struct{}{} //indicate we've closed down message handling
	}()

	return e
}

//Tip returns the current chain tip we're working with
func (e *Engine) Tip() onl.ID {
	return e.chain.Tip()
}

// View will read the chain's state
func (e *Engine) View(f func(kv *onl.KV)) (err error) {
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

	//generate a nonce for this write
	err = w.GenerateNonce()
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %v", err)
	}

	//sign the write
	w.PK = e.idn.PK()
	e.idn.SignWrite(w)

	//handle our own write
	e.handleWrite(w)
	return
}

//Round returns the current round the engine is on
func (e *Engine) Round() uint64 {
	return e.clock.Round()
}

//Handle a single message, assumes that is called in-order
func (e *Engine) Handle(msg *Msg) {
	if msg.Write != nil {
		e.handleWrite(msg.Write)
	} else if msg.Block != nil {
		e.handleBlock(msg.Block)
	} else if msg.Sync != nil {
		e.handleSyncRequest(msg.Sync)
	} else {
		e.logs.Printf("[INFO][%s] read messages that is neither a write or a block, ignoring", e.idn)
		return
	}
}

func (e *Engine) handleSyncRequest(s *sync.Sync) {
	for _, id := range s.IDs {
		b, _, _, err := e.chain.Read(id)
		if err != nil {
			e.logs.Printf("[INFO][%s] peer requested block %s that we failed to read: %v", e.idn, id, err)
			continue
		}

		err = s.Push(b)
		if err != nil {
			e.logs.Printf("[ERRO][%s] failed to push block %s as respons to a sync: %v", e.idn, id, err)
		}
	}
}

func (e *Engine) handleRound(round, ts uint64) {

	//start handling blocks that we're to early
	e.ooo.ResolveRound(round)

	//read tip and current state from chain
	tip, state, err := e.chain.State(onl.NilID)
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
		e.logs.Printf("[INFO][%s][%d] we have no stake put up, proposing no block this round", e.idn, round)
		return //no stake, no proposing for us in this round
	}

	//mint a block for our current tip
	e.logs.Printf("[INFO][%s][%d] we have %d stake to propose blocks, minting on tip %s", e.idn, round, stake, tip)

	//@TODO find a stable block by walking the chain, not hardcode genesis
	b := e.idn.Mint(ts, tip, e.genesis, round)

	//pick writes that are suited for the new block
	e.pool.Pick(state, func(w *onl.Write) bool {
		b.AppendWrite(w)
		if len(b.Writes) >= e.maxw {
			return true
		}

		return false
	})

	//sign the block
	e.idn.Sign(b)

	//handle the block ourselves
	e.handleBlock(b)
}

func (e *Engine) handleWrite(w *onl.Write) {

	//@TODO check if the write (identified with the nonce) is already in the
	//finalized chain. If so, reject.

	//attempt to add to the mempool
	err := e.pool.Add(w)
	if err != nil {
		e.logs.Printf("[INFO][%s] failed to add write to mempool: %v", e.idn, err)
		return
	}

	//relay to peers
	//@TODO we lock the write here because in some conditions it is simultaneously
	//being written to by the ssi.DB commit. We rather solve the root of the issue
	//instead of introductin another read lock
	w.RLock()
	err = e.bc.Write(&Msg{Write: w})
	w.RUnlock()
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to relay write to peers: %v", e.idn, err)
	}
}

func (e *Engine) handleBlock(b *onl.Block) {

	//append the block to the chain, any invalid blocks will be rejected here
	//@TODO make append retry n configurable and have some exponential backoff
	for i := 0; i < 5; i++ {
		err := e.chain.Append(b)
		if err != nil {
			if err == onl.ErrBlockExist {
				return //nothing too do really
			} else if err == onl.ErrAppendConflict {
				continue //retry
			}

			e.logs.Printf("[INFO][%s] failed to append incoming block: %v", e.idn, err)
			return //unexpected failure
		}

		break //append went through
	}

	//handle any messages that were waiting on this block
	id := b.Hash()
	e.ooo.Resolve(id)
	e.logs.Printf("[INFO][%s] appended block %s to our chain", e.idn, id)

	//relay to peers
	err := e.bc.Write(&Msg{Block: b})
	if err != nil {
		e.logs.Printf("[ERRO][%s] failed to relay block to peers: %v", e.idn, err)
	}
}

//Draw will vizualize the engine's chain using the dot graph language
func (e *Engine) Draw(w io.Writer) (err error) {
	fmt.Fprintln(w, `digraph {`)

	tip := e.chain.Tip()

	if err = e.chain.ForEach(0, func(id onl.ID, b *onl.Block, stk *onl.Stakes) error {
		f := stk.Finalization()

		fmt.Fprintf(w, "\t"+`"%.6x" [shape=box,style="filled,solid",label="%.6x:%d:%d:%.1f"`, id[8:], id[8:], b.Round, len(b.Writes), f)

		if id == tip {
			fmt.Fprintf(w, `,penwidth="3"`)
		} else {
			fmt.Fprintf(w, `,penwidth="1"`)
		}

		switch {
		case f >= 1.0: //unanimous
			fmt.Fprintf(w, `,fillcolor="#BBBBBB"`)
		case f >= 0.66667: //super majority
			fmt.Fprintf(w, `,fillcolor="#CCCCCC"`)
		case f >= 0.5: //majority
			fmt.Fprintf(w, `,fillcolor="#DDDDDD"`)
		case f > 0.0: //minority
			fmt.Fprintf(w, `,fillcolor="#EEEEEE"`)
		case f <= 0.0: //none
			fmt.Fprintf(w, `,fillcolor="#FFFFFF"`)
		}

		fmt.Fprintf(w, "]\n")
		fmt.Fprintf(w, "\t"+`"%.6x" -> "%.6x";`+"\n", id[8:], b.Prev[8:])

		return nil
	}); err != nil {
		return fmt.Errorf("failed to iterate over all blocks: %v", err)
	}

	fmt.Fprintln(w, `}`)
	return
}

// Shutdown will gracefully shutdown the engine. It will ask subsystems to close
// gracefully first and wait for them before. If the context expires first its
// error is returned.
func (e *Engine) Shutdown(ctx context.Context) (err error) {
	err = e.bc.Close()
	if err != nil {
		return fmt.Errorf("failed to close broadcast: %v", err)
	}

	err = e.clock.Close()
	if err != nil {
		return fmt.Errorf("failed to close pulse: %v", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done: //1th subsystem
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.done: //2th subsystem
			return nil
		}
	}
}
