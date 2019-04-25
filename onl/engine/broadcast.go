package engine

import (
	"github.com/advanderveer/27067dd17/onl"
)

//Msg transports information over the broadcast network.
type Msg struct {
	Block *onl.Block
	Write *onl.Write
}

// Dependency returns what block this message is dependant on the before it can
// be handled (if any).
func (msg *Msg) Dependency() (dep onl.ID) {
	if msg.Write != nil || msg.Block == nil {
		return onl.NilID
	}

	return msg.Block.Prev
}

//Broadcast provide reliable message dissemation
type Broadcast interface {
	Read(msg *Msg) (err error)
	Write(msg *Msg) (err error)
	Close() (err error)
}
