package onl

import "fmt"

//Chain links blocks together and reaches consensus by keeping the chain with
//the most weight
type Chain struct {
	store Store

	genesis struct {
		*Block
		*Stakes
		id ID
	}
}

//NewChain creates a new Chain
func NewChain(s Store) (c *Chain) {
	c = &Chain{store: s}

	//try to read genesis
	tx := c.store.CreateTx(true)
	defer tx.Discard()

	if err := tx.Round(0, func(b *Block, stk *Stakes) error {
		if c.genesis.Block != nil {
			panic("more then 1 block in round 0, expected only the genesis block")
		}

		c.genesis.Block = b
		c.genesis.Stakes = stk
		return nil
	}); err != nil {
		panic("failed to read round 0 for genesis block: " + err.Error())
	}

	//if no genesis could be read, create
	if c.genesis.Block == nil {
		c.genesis.Block = &Block{Token: []byte("vi veri veniversum vivus vici")}

		c.genesis.Stakes = &Stakes{} //@TODO finalize block
		if err := tx.Write(c.genesis.Block, c.genesis.Stakes); err != nil {
			panic("failed to write genesis block: " + err.Error())
		}

		if err := tx.Commit(); err != nil {
			panic("failed to commit writing of genesis block: " + err.Error())
		}
	}

	c.genesis.id = c.genesis.Hash()
	return
}

//Genesis returns the genesis block
func (c *Chain) Genesis() (b *Block) { return c.genesis.Block }

// Walk a chain from 'id' towards the genesis.
func (c *Chain) Walk(id ID) (err error) {

	//@TODO read in one tx?

	return
}

// Append a block to the chain. If an error is returned the block could not
// added to the chain. If others do decide to add it this means the network
// will fork. Block may be appended long after they have been created, either
// because they took a long time to traverse the network or because it was
// delivered via another channel to sync up the this chain.
func (c *Chain) Append(b *Block) (err error) {

	//check signature, make sure it hasn't been tampered with since signed
	if !b.VerifySignature() {
		return ErrInvalidSignature
	}

	//make sure round is > 0
	if b.Round < 1 {
		return ErrZeroRound
	}

	//open our store tx
	tx := c.store.CreateTx(true)
	defer tx.Discard()

	//check if the block already exists
	id := b.Hash()
	_, _, err = tx.Read(id)
	if err != ErrBlockNotExist {
		return ErrBlockExist
	}

	//check if prev chain exists (and was deemed valid)
	// - finalized prev, must be in prev chain
	// - must be no other finalized prev after the selected finalized prev
	// - update finalization info
	// - read roundtime
	// - read stake from chain
	// - read vrf pk that was committed (if any)
	// - read the vrf threshold (if any)

	// check if timestamp comes after prev timestamp
	// check if prev block makes sense (no other blocks in between)

	// check if finalized prev is valid, it determines the seeds randomness
	// prevent the proposer from grinding finalized prevs for higher token

	// write the actual block
	err = tx.Write(b, nil)
	if err != nil {
		return fmt.Errorf("failed to write block: %v", err)
	}

	// check if the round nr makes sense (together with timestamp?)
	// re-rank the block in its round

	// re-calculate all weights from this round up to the latest
	// - determine new tip

	// distribute stake to all ancestors for finalization
	// - add this blocks proser's stake to each ancestor
	// - mark any blocks as finalized
	// - keep new finalized tip

	return tx.Commit()
}
