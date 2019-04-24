package engine

import (
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
	handler Handler
	defers  map[onl.ID][]*Msg
}

//NewOutOfOrder creates a new OutOfOrder
func NewOutOfOrder(h Handler) *OutOfOrder {
	return &OutOfOrder{handler: h, defers: make(map[onl.ID][]*Msg)}
}

//Resolve will handle any messages that depended on this block
func (o *OutOfOrder) Resolve(id onl.ID) {
	defers, ok := o.defers[id]
	if ok {
		for _, msg := range defers {
			o.handler.Handle(msg)
		}
	}

	o.defers[id] = nil //nil elements means the id is resolved
}

//Handle will try to handle the message unless it waits for a block to resolve
func (o *OutOfOrder) Handle(msg *Msg) {
	dep := msg.Dependency()
	if dep == onl.NilID { //no dependency
		o.handler.Handle(msg)
		return
	}

	ex, ok := o.defers[dep]
	if ok && ex == nil { //dep is already resolved
		o.handler.Handle(msg)
		return
	}

	ex = append(ex, msg)
	o.defers[dep] = ex
}
