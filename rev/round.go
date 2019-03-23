package rev

import (
	"sync"
)

// Round represents a time-bound segmentation of proposals. Members move through
// rounds such that old ones eventually get no messages and can be finalized.
// Each round captures proposals and keeps the highest ranking once such that
// it can validate future witnesses that refer to them.
type Round struct {
	proposals map[PID]*Proposal
	proposed  map[ID]*Proposal
	top       *Proposal
	mu        sync.RWMutex
	chain     *Chain
}

//NewRound sets up a new round
func NewRound(c *Chain, ps ...*Proposal) (r *Round) {
	r = &Round{
		chain:     c,
		proposals: make(map[PID]*Proposal),
		proposed:  make(map[ID]*Proposal),
	}

	for _, p := range ps {
		r.add(p)
	}

	return
}

func (r *Round) add(p *Proposal) {
	r.proposals[p.Hash()] = p
	r.proposed[p.Block.Hash()] = p

	//new top proposal
	if r.top == nil || p.GT(r.top) {
		r.top = p
	}
}

// Observe a proposal for this round. With enough observations we can Prove
// that we saw enough proposals and our conclusion about the top ranking one
// has merit such that we can use the block from it as our 'prev'.
func (r *Round) Observe(p *Proposal) (witness PIDSet, tip ID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//add proposal
	r.add(p)

	//we would like the tip we're building rank as high as possible so we keep
	//a prev ref to the highest rak
	tip = r.top.Block.Hash()

	//figure out if we already witnessed enough to propose a block that
	//builds onto the top proposal's block.
	if len(r.proposals) < r.chain.WitnessThreshold(tip) {
		return nil, tip
	}

	//@TODO (optimization) check if we have can solve the puzzle with this new
	//observation with the diffiulty described by the tip

	//@TODO (optimization) instead of encoding all proposals proof that we have
	//a lot using a mathematical puzzle

	//fill in the witness set with all proposals we currently see
	witness = make(PIDSet, len(r.proposals))
	for pid := range r.proposals {
		witness.Add(pid)
	}

	return witness, tip
}

// Question the witness and verify that it represents a valid proof that the other
// member has correctly waited and correctly ranked proposals on its end.
func (r *Round) Question(prev ID, witness PIDSet) (ok bool, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	//check if we know about the proposal that was brought on as a witness. This is
	//more of a sanity check as the out-of-order shouldn't even serve a proposal with a
	//witness we haven't seen yet.
	var topw *Proposal
	for wid := range witness {
		wp, ok := r.proposals[wid]
		if !ok {
			return false, ErrProposalWitnessUnknown
		}

		if topw == nil || wp.GT(topw) {
			topw = wp
		}
	}

	//@TODO (optimization) must have solved math puzzle above a certain difficulty
	//threshold as dictated by the chain that leads down from prev

	//must include a min amount of witnesses
	if len(witness) < r.chain.WitnessThreshold(prev) {
		return false, ErrNotEnoughWitness
	}

	//the prev block must be proposed by a proposal in this round
	pp, ok := r.proposed[prev]
	if !ok {
		return false, ErrPrevProposalNotFound
	}

	//the previous block must be proposed by the top ranking witness
	if topw.Hash() != pp.Hash() {
		return false, ErrPrevProposalNotTopWitness
	}

	return true, nil
}
