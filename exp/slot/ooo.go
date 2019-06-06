package slot

import "sync"

// OutOfOrder is a message handling proxy that will look at certain messages and
// check if the message is only relevant if a previous block has arrived first.
// if this is the case it will not handle it right away but wait for it to get
// handled first.
type OutOfOrder struct {
	handler MsgHandler
	br      BlockReader
	orphans map[ID][]*Msg
	mu      sync.Mutex

	// @TODO it is possible that a dependency will never get resolved, figure out
	// how we clean that up such that storage/memory doesn't grow unbounded. This
	// is pretty easy o abuse, everyone can make up a message with a 'prev' that
	// doesn't exist. we could add some ttl that cleans up after some time
}

// NewOutOfOrder creates a new out of order handler. Upon deferred handling it
// will use the provided BroadcastWriter to allow the handler to write output
// to the network
func NewOutOfOrder(br BlockReader, h MsgHandler) *OutOfOrder {
	return &OutOfOrder{
		handler: h,
		br:      br,
		orphans: make(map[ID][]*Msg),
	}
}

// Resolve is called when another system has stored a notarized block into our
// chain which might allow us to resolve deferred handles.
func (o *OutOfOrder) Resolve(b *Block) (n int, err error) {
	id := b.Hash()
	o.mu.Lock()
	defer o.mu.Unlock()

	orphans, ok := o.orphans[id]
	if !ok {
		return n, nil //nothing to de-orphan
	}

	var errs []error
	for _, msg := range orphans {
		err = o.handler.Handle(msg)
		n++
		if err != nil {
			errs = append(errs, err)
		}
	}

	//clean orphans, if any caused an error we will not consider
	//de-orphaning them again anway (how ungratefull)
	delete(o.orphans, id)
	if len(errs) > 0 {
		return n, ResolveErr(errs)
	}

	return
}

// Handle a single message, it may defer actual handling until its the blocks it
// depends on have been resolved first. Any error that results as part of this
// deferred handling is throws as part of the resolving message in the future.
func (o *OutOfOrder) Handle(msg *Msg) (err error) {
	var prev ID
	switch msg.Type() {
	case MsgTypeProposal:
		prev = msg.Proposal.Prev
	case MsgTypeVote:
		prev = msg.Vote.Prev
	default:
		return o.handler.Handle(msg)
	}

	//no dependency, handle right away
	if prev == NilID {
		return o.handler.Handle(msg)
	}

	prevb := o.br.Read(prev)
	if prevb == nil {
		o.mu.Lock()
		//append as orphan, note that a message can be added multple times, this is
		//intentional because the user might have a reaon even though the overall
		//procol mostly prevents that
		o.orphans[prev] = append(o.orphans[prev], msg)
		o.mu.Unlock()

		//defer handling until it is resolved
		return nil
	}

	return o.handler.Handle(msg)
}
