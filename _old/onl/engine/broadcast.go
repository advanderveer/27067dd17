package engine

import (
	"github.com/advanderveer/27067dd17/onl"
)

//Msg transports information over the broadcast network.
type Msg struct {
	Block *onl.Block
	Write *onl.Write
	Sync  *Sync
}

// Dependency returns what block this message is dependant on the before it can
// be handled (if any).
func (msg *Msg) Dependency() (dep onl.ID, round uint64) {
	if msg.Write != nil || msg.Block == nil {
		return onl.NilID, 0
	}

	return msg.Block.Prev, msg.Block.Round
}

// Sync is send to peers when a member requires a specific block
type Sync struct {
	IDs []onl.ID

	//wf is set (decorated) by the broadcast layer
	wf func(b *onl.Block) (err error)
}

//SetWF sets the return write function
func (r *Sync) SetWF(f func(b *onl.Block) (err error)) { r.wf = f }

//Push the block to the peer that requested the sync
func (r *Sync) Push(b *onl.Block) (err error) {
	if r.wf == nil {
		panic("sync push without write function")
	}

	return r.wf(b)
}

//Broadcast provide reliable message dissemation
type Broadcast interface {
	Read(msg *Msg) (err error)
	Close() (err error)
	BroadcastWriter
}

//BroadcastWriter is ther writing part of the broadcast
type BroadcastWriter interface {
	Write(msg *Msg) (err error)
}
