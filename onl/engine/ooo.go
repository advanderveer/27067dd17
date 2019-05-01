package engine

import (
	"sync"

	"github.com/advanderveer/27067dd17/onl"
)

//Handler handles messages
type Handler interface {
	Handle(msg *Msg)
}

//HandlerFunc can be used to implement handler
type HandlerFunc func(msg *Msg)

//Handle implements the handler interface
func (h HandlerFunc) Handle(msg *Msg) { h(msg) }

//OutOfOrder allows for calling a handler that handles messages depending on
//another block to be called later
type OutOfOrder struct {
	handler  Handler
	mu       sync.RWMutex
	onBlocks map[onl.ID][]*Msg
	onRounds map[uint64][]*Msg
}

//NewOutOfOrder creates a new OutOfOrder
func NewOutOfOrder(h Handler) *OutOfOrder {
	return &OutOfOrder{
		handler:  h,
		onBlocks: make(map[onl.ID][]*Msg),
		onRounds: make(map[uint64][]*Msg),
	}
}

//ResolveRound will resolve blocks that are waiting on the round to start
func (o *OutOfOrder) ResolveRound(nr uint64) {
	o.mu.Lock()
	defers, ok := o.onRounds[nr]
	o.onRounds[nr] = nil
	o.mu.Unlock()

	if ok {
		for _, msg := range defers {
			o.Handle(msg)
		}
	}
}

//Resolve will handle any messages that depended on this block
func (o *OutOfOrder) Resolve(id onl.ID) {
	o.mu.Lock()
	defers, ok := o.onBlocks[id]
	o.onBlocks[id] = nil
	o.mu.Unlock()

	if ok {
		for _, msg := range defers {
			o.Handle(msg)
		}
	}
}

//Handle will try to handle the message unless it waits for a block or round
//to resolve first
func (o *OutOfOrder) Handle(msg *Msg) {
	var (
		bdepResolved bool
		rdepResolved bool
	)

	//aks for block and round deps
	bdep, rdep := msg.Dependency()

	o.mu.Lock()

	//if there is a block dependency, check if it was resolved
	//already or else add it to the block defer. In that case
	//it will not be resolved
	if bdep != onl.NilID {
		ex, ok := o.onBlocks[bdep]
		if ok && ex == nil {
			bdepResolved = true
		} else if !ok { //only add if not exist
			ex = append(ex, msg)
			o.onBlocks[bdep] = ex
		}
	} else {
		bdepResolved = true //no block dep, always resolved
	}

	//if there is a round dependency, check if it was already resolved
	//or else schedule it for resolving and don't mark it as such
	if rdep > 0 {
		ex, ok := o.onRounds[rdep]
		if ok && ex == nil {
			rdepResolved = true
		} else if !ok { //only add if ot exist
			ex = append(ex, msg)
			o.onRounds[rdep] = ex
		}
	} else {
		rdepResolved = true //no round dep, always resolved
	}

	o.mu.Unlock()

	//if both are resolved we can finally call the handle
	if rdepResolved && bdepResolved {
		go o.handler.Handle(msg)
	}
}
