package tt

import (
	"context"
	"fmt"
	"io"
	"log"
)

//Engine reads, handles and writes messages that run the propotocol
type Engine struct {
	idn   *Identity
	bc    Broadcast
	logs  *log.Logger
	done  chan struct{}
	miner *Miner
}

//NewEngine creates a new engine
func NewEngine(logw io.Writer, idn *Identity, bc Broadcast, m *Miner) (e *Engine) {
	e = &Engine{
		idn:   idn,
		bc:    bc,
		logs:  log.New(logw, "", 0),
		done:  make(chan struct{}, 1),
		miner: m,
	}

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

			//@TODO out-of-order handle
			//@TODO do handlich concurrently

			if msg.Vote != nil {
				e.HandleVote(msg.Vote)
			} else if msg.Block != nil {
				e.HandleBlock(msg.Block)
			} else {
				e.logs.Printf("[INFO] received unkown message, ignoring it")
				continue //nothing to do
			}
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

//HandleVote will feed votes to the miner
func (e *Engine) HandleVote(v *Vote) {

	//@TODO send vote to miner
	//@TODO relay if vote is for the correct tip

}

//HandleBlock will add a block to the chain and move the tip
func (e *Engine) HandleBlock(b *Block) {

	//@TODO add to block, check if new tip

}
