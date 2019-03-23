package rev

//Chain stores linked block such that it can reach consensus on the longest chain
type Chain struct {
	//@TODO for now we configure a fixed threshold but this should be determined by
	//walking the chain and looking at past blocks and proposals
	fixedWitnessThreshold int
}

//NewChain creates the block chain
func NewChain(t int) (c *Chain) {
	c = &Chain{fixedWitnessThreshold: t}
	return
}

//WitnessThreshold will walk the chain and determine what nr of witnesses are
//necessary to use block 'id' as a tip.
func (c *Chain) WitnessThreshold(id ID) (t int) {
	return c.fixedWitnessThreshold
}
