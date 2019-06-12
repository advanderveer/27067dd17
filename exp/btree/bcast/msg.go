package bcast

// Msg represents a broadcast peer message
type Msg struct {
	Foo string
}

// Writer writes messages to the broadcast
type Writer interface {
	Write(msg *Msg) (err error)
}

// Reader reads the next broadcast message
type Reader interface {
	Read() (msg *Msg, err error)
}
