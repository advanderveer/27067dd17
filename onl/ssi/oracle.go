package ssi

//Oracle represents the status oracle in "A critique of snapshot isolation" [M Yabandeh, 2012]
type Oracle struct {
	time    uint64
	commits map[uint64]uint64
}

//NewOracle creates the status oracle
func NewOracle() *Oracle {
	return &Oracle{
		time:    1,
		commits: make(map[uint64]uint64),
	}
}

//Curr returns the current time kept by the status oracle
func (o *Oracle) Curr() uint64 {
	return o.time
}

//Commit will check if concurrent transactions we're committed that wrote to the
//keys this commit wrote to check for conflicts.
func (o *Oracle) Commit(rr, rw KeySet, ts uint64) (tc uint64) {

	//if if there were any transactions that committed after the start (ts)
	//of this transaction and that wrote to the the rows that this transaction
	//has read from

	for r := range rr {
		if o.commits[r] > ts {
			return 0 //conflict
		}
	}

	//if not, we mark commits with their new write time
	o.time++
	for r := range rw {
		o.commits[r] = o.time //keep the last committed time
	}

	return o.time
}
