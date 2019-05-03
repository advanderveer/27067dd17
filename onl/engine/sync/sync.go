package sync

import "github.com/advanderveer/27067dd17/onl"

//@TODO when and how will the out-of-order call our sync logic?
//@TODO how will we deliver blocks to the engine? Through the out-of-order?
//@TODO how do we validate the blocks early, prevent spam etc
//@TODO how do we prevent sync's too often (maybe broadcast is still delivering)?
//@TODO how do we prevent sync's too slow (creating a large backlog of blocks not comming in)
//@TODO how do we prevent the engine from relaying the blocks?

// Sync is send to peers when a member requires a specific block
type Sync struct {
	IDs []onl.ID

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
