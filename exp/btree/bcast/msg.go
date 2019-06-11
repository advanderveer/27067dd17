package bcast

// Msg encodes all messages our members exchange
type Msg struct {
	Foo string //@TODO replace by sensible message
}

// Broadcast describes an endpoint for writng and reading the messages
type Broadcast interface {
	BroadcastReader
	BroadcastWriter
}

// BroadcastReader describes the ability to read messages from the broadcast
type BroadcastReader interface {
	Read() (msg *Msg, err error)
}

// BroadcastWriter describes the ability to write message to the broadcast
type BroadcastWriter interface {
	Write(msg *Msg) (err error)
}
