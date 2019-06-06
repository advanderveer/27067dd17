package tt

import (
	"context"
	"fmt"
	"io"
	"log"
)

//Engine reads, handles and writes messages that run the protocol
type Engine struct {
	idn   *Identity
	bc    Broadcast
	logs  *log.Logger
	done  chan struct{}
	miner *Miner
	ooo   *OutOfOrder
	chain *Chain
}

//NewEngine creates a new engine
func NewEngine(logw io.Writer, idn *Identity, bc Broadcast, m *Miner, c *Chain) (e *Engine) {
	e = &Engine{
		idn:   idn,
		bc:    bc,
		logs:  log.New(logw, "", 0),
		done:  make(chan struct{}, 1),
		miner: m,
		chain: c,
	}

	e.ooo = NewOutOfOrder(e)

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

			e.ooo.Handle(msg)
		}

		close(e.done) //indicate we've closed down message handling
	}()

	//mined block handling
	go func() {
		for {
			ctx := context.Background()
			b, err := e.miner.Next(ctx)
			if err != nil {
				e.logs.Printf("[ERRO] failed to get next mined block: %v", err)
				break
			}

			//@TODO (optimization) add to our own chain right away?

			err = e.bc.Write(&Msg{Block: b})
			if err != nil {
				e.logs.Printf("[ERRO] failed to write mined block to broadcast: %v", err)
				break
			}
		}
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

	//@TODO wait for mined blocks routine to shutdown also

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return nil
	}
}

//Handle a message
func (e *Engine) Handle(msg *Msg) {
	if msg.Vote != nil {
		go e.HandleVote(msg.Vote)
	} else if msg.Block != nil {
		go e.HandleBlock(msg.Block)
	} else {
		e.logs.Printf("[INFO] received unkown message, ignoring it")
		return //nothing to do
	}
}

//HandleVote will feed votes to the miner
func (e *Engine) HandleVote(v *Vote) {
	//@TODO validate vote, check tip
	//@TODO send vote to miner
	//@TODO relay if vote its for the correct tip
	e.miner.Feed(v)
}

//HandleBlock will add a block to the chain and move the tip
func (e *Engine) HandleBlock(b *Block) {
	//@TODO validate
	//@TODO add to chain, check if new tip
	id := b.Hash()
	e.chain.Append(b)

	//if its a new longest chain, send a vote for it

	e.ooo.Resolve(id)
}
