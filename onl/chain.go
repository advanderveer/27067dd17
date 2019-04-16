package onl

//Chain links blocks together and reaches conensus by keeping the chain with
//the most weight
type Chain struct {
	store Store
}

//NewChain creates a new Chain
func NewChain(s Store) (c *Chain) {
	c = &Chain{store: s}
	return
}

// Append a block to the chain. If an error is returned the block could not
// added to the chain. If others do decide to add it this means the network
// will fork. Block may be appended long after they have been created, either
// because they took a long time to traverse the network or because it was
// delivered via another channel to sync up the this chain.
func (c *Chain) Append(b *Block) (err error) {

	//check signature
	if !b.VerifySignature() {
		return ErrInvalidSignature
	}

	//check if prev chain exists (and was deemed valid)
	// - read roundtime
	// - read stake from chain
	// - read vrf pk that was committed (if any)
	// - read the vrf threshold (if any)

	// check if timestamp comes after prev timestamp
	// check if prev block makes sense (no other blocks in between)

	// check if finalized prev is valid, it determines the seeds randomness
	// prevent the proposer from grinding finalized prevs for higher token

	// check if the round nr makes sense (together with timestamp?)
	// re-rank the block in its round

	// re-calculate all weights from this round up to the latest
	// - determine new tip

	// distribute stake to all ancestors for finalization
	// - mark any blocks as finalized
	// - keep new finalized tip

	return
}
