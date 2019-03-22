package rev

// OutOfOrder can defer proposal handling until its dependencies have been
// handled first. Some proposals might never get resolved if their dependencies
// never arrive. This might be due to a member that tried to forge proof or
// because the network failed to replicate it. @TODO in any case, we need a way
// to expire old proposals
type OutOfOrder struct {
	h Handler
	//@TODO we would probably like to make this suitable for concurrent use but
	//that puts a strain on how long each handle function may take

	orphans map[PID]map[PID]*Proposal
	handled PIDSet
}

//NewOutOfOrder will call 'hf' on every proposal that is marke as resolved
func NewOutOfOrder(h Handler) (o *OutOfOrder) {
	o = &OutOfOrder{
		h:       h,
		orphans: make(map[PID]map[PID]*Proposal),
		handled: make(PIDSet),
	}

	return
}

//Size returns how much we're corrently waiting for
func (o *OutOfOrder) Size() (orphans, handled int) {
	return len(o.orphans), len(o.handled)
}

func (o *OutOfOrder) resolve(id PID) {

	//anyone waiting for this id
	orphans, ok := o.orphans[id]
	if !ok {
		return //nothing to de-orphan
	}

	//retry to handle of de-orphaned proposal (recurse)
	for _, orphan := range orphans {
		o.handle(orphan)
	}

	delete(o.orphans, id)
}

func (o *OutOfOrder) handleNow(id PID, p *Proposal) {
	o.h.Handle(p)              //do actual handle
	o.handled[id] = struct{}{} //mark as handled
	o.resolve(id)              //resolve any proposal waiting
}

// Handle will try to handle the proposal but if necessary wait for
func (o *OutOfOrder) Handle(p *Proposal) {
	o.handle(p)
}

func (o *OutOfOrder) handle(p *Proposal) {
	id := p.Hash()
	if len(p.Witness) < 1 {
		o.handleNow(id, p) //no deps, handle right away
		return
	}

	var deps []PID
	for dep := range p.Witness {
		if _, ok := o.handled[dep]; ok {
			continue //already resolved
		}

		deps = append(deps, dep)
	}

	if len(deps) < 1 {
		o.handleNow(id, p) //all deps were handled
		return
	}

	for _, dep := range deps {
		ex, ok := o.orphans[dep]
		if !ok {
			ex = make(map[PID]*Proposal)
		}

		ex[id] = p
		o.orphans[dep] = ex
	}
}
