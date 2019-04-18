package onl

import (
	"fmt"
)

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
func NewChain(s Store, ws ...*Write) (c *Chain, gen ID) {
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
		c.genesis.Block.Append(ws...)

		c.genesis.Stakes = &Stakes{} //@TODO finalize block
		if err := tx.Write(c.genesis.Block, c.genesis.Stakes); err != nil {
			panic("failed to write genesis block: " + err.Error())
		}

		if err := tx.Commit(); err != nil {
			panic("failed to commit writing of genesis block: " + err.Error())
		}
	}

	c.genesis.id = c.genesis.Hash()
	return c, c.genesis.id
}

//Genesis returns the genesis block
func (c *Chain) Genesis() (b *Block) { return c.genesis.Block }

// State returns the state represented by the chain walking back to the genesis
func (c *Chain) State(id ID) (s *State, err error) {

	//@TODO (optimization) we would like to keep a cache of the state on the longest
	//finalized chain

	tx := c.store.CreateTx(false)
	defer tx.Discard()
	return c.state(tx, id)
}

func (c *Chain) state(tx Tx, id ID) (s *State, err error) {
	var log [][]*Write
	if err = c.walk(tx, id, func(id ID, bb *Block, stk *Stakes) error {
		log = append([][]*Write{bb.Writes}, log...)
		return nil
	}); err != nil {
		return nil, err
	}

	return NewState(log)
}

// Walk a chain from 'id' towards the genesis.
func (c *Chain) Walk(id ID, f func(id ID, b *Block, stk *Stakes) error) (err error) {
	tx := c.store.CreateTx(false)
	defer tx.Discard()
	return c.walk(tx, id, f)
}

func (c *Chain) walk(tx Tx, id ID, f func(id ID, b *Block, stk *Stakes) error) (err error) {

	//@TODO (optimization) would it be possible to do a key only walk on the lsm index

	for {
		b, stk, err := tx.Read(id)
		if err != nil {
			return err
		}

		err = f(id, b, stk)
		if err != nil {
			return err
		}

		if id == c.genesis.id {
			return nil //we reached the genesis
		}

		id = b.Prev
	}
}

// Append a block to the chain. If an error is returned the block could not be
// added to the chain. If others do decide to add it this means the network
// will fork. Block may be appended long after they have been created, either
// because they took a long time to traverse the network or because it was
// delivered via another channel to sync up the this chain.
func (c *Chain) Append(b *Block) (err error) {

	// check signature, make sure it hasn't been tampered with since signed
	if !b.VerifySignature() {
		return ErrInvalidSignature
	}

	// make sure round is > 0
	if b.Round < 1 {
		return ErrZeroRound
	}

	// open our store tx
	tx := c.store.CreateTx(true)
	defer tx.Discard()

	// check if the block already exists
	id := b.Hash()
	_, _, err = tx.Read(id)
	if err != ErrBlockNotExist {
		return ErrBlockExist
	}

	// prev blocks
	var (
		fprev *Block
		prev  *Block
	)

	// walk prev chain while storing all blocks up to the genesis a log
	if err = c.walk(tx, b.Prev, func(id ID, bb *Block, stk *Stakes) error {
		if b.Prev == id {
			prev = bb
		}

		// if we see a newer finalized block in the chain then the fprev of this
		// block we do not accept it.
		if fprev == nil && stk.HasMajority() {
			//@TODO test to figure out what this does to the forking rate
			//@TODO can we instead find a more stable finalized prev in the further back?
		}

		if b.FinalizedPrev == id {
			fprev = bb
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk prev chain: %v", err)
	}

	// fprev must exist in the prev chain
	if fprev == nil {
		return ErrFinalizedPrevNotInChain
	}

	//reconstruct the state
	//@TODO we're walking again here but we would like to re-use our code also
	state, err := c.state(tx, b.Prev)
	if err != nil {
		return ErrStateReconstruction
	}

	//read dynamic data from rebuild state
	var stake uint64
	var tpk []byte
	state.Read(func(kv *KV) {
		stake, tpk = kv.ReadStake(b.PK)
		// @TODO read roundtime
		// @TODO read the vrf threshold (if any)
	})

	//check if there was any token pk comitted
	if tpk == nil {
		return ErrNoTokenPK
	}

	//validate the token
	if !b.VerifyToken(tpk) {
		return ErrInvalidToken
	}

	//validate each write in the block by applying them in a dry-run
	for _, w := range b.Writes {
		err = state.Apply(w, true)
		if err != nil {
			return err
		}
	}

	// prev timestamp must be before blocks timestamp, due to the chaining logic
	// it is also ensured that the fprev timestamp is before prev
	if prev.Timestamp >= b.Timestamp {

	}

	// the blocks round must be after the prev round
	if prev.Round >= b.Round {

	}

	// check if there were other blocks that the propser should have used as prev
	for r := prev.Round; r < b.Round; r++ {
		//@TODO check if we know of any other block in a round in between the two rounds
		//that could have been used as a prev?
		//@TODO is this check still important if token randomness is based on fprev
		//@TODO return early so proposers cannot ddos the acceptor
	}

	// write the actual block
	err = tx.Write(b, nil)
	if err != nil {
		return fmt.Errorf("failed to write block: %v", err)
	}

	// check if the round nr makes sense (together with timestamp?) what happens if
	// a very high round nr is proposed with a very recent timestamp (prev+1)

	_ = stake
	// [MAJOR] re-rank the block in its round
	// - (requires) take and vrf pk to be read from chain
	// re-calculate all weights from this round up to the latest
	// - determine new tip
	// - re assign sum weights
	// - we don' need to rank re-sort the whole round

	// [MAJOR] distribute stake to all ancestors for finalization
	// - (requires stake) to be read from chain
	// - add this blocks proser's stake to each ancestor
	// - mark any blocks as finalized
	// - keep new finalized tip as chainstate cache
	// - apply newly finalized blocks

	return tx.Commit()
}
