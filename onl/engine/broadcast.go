package engine

import (
	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine/sync"
)

//Msg transports information over the broadcast network.
type Msg struct {
	Block *onl.Block
	Write *onl.Write
	Sync  *sync.Sync
}

// Dependency returns what block this message is dependant on the before it can
// be handled (if any).
func (msg *Msg) Dependency() (dep onl.ID, round uint64) {
	if msg.Write != nil || msg.Block == nil {
		return onl.NilID, 0
	}

	return msg.Block.Prev, msg.Block.Round
}

//Broadcast provide reliable message dissemation
type Broadcast interface {
	Read(msg *Msg) (err error)
	Write(msg *Msg) (err error)
	Close() (err error)
}
