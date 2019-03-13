package slot

// OutOfOrder is a message handling proxy that will look at certain messages and
// check if the message is only relevant if a previous block has arrived first.
// if this is the case it will not handle it right away but wait for it to get
// handled first.
type OutOfOrder struct {
	handle func(in *Msg, bw BroadcastWriter) (err error)
	bw     BroadcastWriter
}

// NewOutOfOrder creates a new out of order handler. Upon deferred handling it
// will use the provided BroadcastWriter to allow the handler to write output
// to the network
func NewOutOfOrder(h func(in *Msg, bw BroadcastWriter) (err error), bw BroadcastWriter) *OutOfOrder {
	return &OutOfOrder{handle: h, bw: bw}
}

// Handle a single message, it may defer actual handling until its the blocks it
// depends on have been resolved first. Any error that results as part of this
// deferred handling is throws as part of the resolving message in the future
func (o *OutOfOrder) Handle(msg *Msg) (err error) {

	//@TODO implement

	return o.handle(msg, o.bw)
}
