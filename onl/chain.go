package onl

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
)

//Chain links blocks together and reaches consensus by keeping the chain with
//the most weight
type Chain struct {
	points  uint64
	store   Store
	genesis struct {
		*Block
		*Stakes
		id ID
	}

	//@TODO move weights and locking to a store so on boot we can just continue
	//where we left off
	tip     ID
	tipw    uint64
	weights map[ID]uint64
	wmu     sync.RWMutex
}

//NewChain creates a new Chain
func NewChain(s Store, ws ...*Write) (c *Chain, gen ID, err error) {
	c = &Chain{
		points:  1000, //@TODO make this configurable?
		store:   s,
		weights: make(map[ID]uint64),
	}

	//genesis prev weight is 0
	c.weights[NilID] = 0

	//try to read genesis
	tx := c.store.CreateTx(true)
	defer tx.Discard()

	if err := tx.Round(0, func(id ID, b *Block, stk *Stakes, rank *big.Int) error {
		if c.genesis.Block != nil {
			return fmt.Errorf("there is more then just the genesis block")
		}

		c.genesis.Block = b
		c.genesis.Stakes = stk
		c.genesis.id = id
		return nil
	}); err != nil {
		return nil, gen, fmt.Errorf("failed to read round 0 for genesis block: " + err.Error())
	}

	//if no genesis could be read, create
	if c.genesis.Block == nil {
		c.genesis.Block = &Block{Token: []byte("vi veri veniversum vivus vici")}
		c.genesis.Block.Append(ws...)

		c.genesis.Stakes = &Stakes{} //@TODO finalize block
		if err := tx.Write(c.genesis.Block, c.genesis.Stakes, big.NewInt(1)); err != nil {
			return nil, gen, fmt.Errorf("failed to write genesis block: " + err.Error())
		}

		if err := tx.Commit(); err != nil {
			return nil, gen, fmt.Errorf("failed to commit writing of genesis block: %v", err)
		}

		c.genesis.id = c.genesis.Hash()
	}

	//set the tip to the genesis block
	c.tip = c.genesis.id

	//re-run weight calculations for the whole chain
	//@TODO (optimization) we don't want to do this every time for long chains
	err = c.weigh(tx, 0)
	if err != nil {
		return nil, gen, fmt.Errorf("failed to weigh chain blocks: %v", err)
	}

	return c, c.genesis.id, nil
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
	if err = c.walk(tx, id, func(id ID, bb *Block, stk *Stakes, rank *big.Int) error {
		log = append([][]*Write{bb.Writes}, log...)
		return nil
	}); err != nil {
		return nil, err
	}

	return NewState(log)
}

// Walk a chain from 'id' towards the genesis.
func (c *Chain) Walk(id ID, f func(id ID, b *Block, stk *Stakes, rank *big.Int) error) (err error) {
	tx := c.store.CreateTx(false)
	defer tx.Discard()
	return c.walk(tx, id, f)
}

func (c *Chain) walk(tx Tx, id ID, f func(id ID, b *Block, stk *Stakes, rank *big.Int) error) (err error) {

	//@TODO (optimization) would it be possible to do a key only walk on the lsm index

	for {
		b, stk, rank, err := tx.Read(id)
		if err != nil {
			return err
		}

		err = f(id, b, stk, rank)
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
	_, _, _, err = tx.Read(id)
	if err != ErrBlockNotExist {
		return ErrBlockExist
	}

	// prev blocks
	var (
		fprev *Block
		prev  *Block
	)

	// walk prev chain while storing all blocks up to the genesis a log
	if err = c.walk(tx, b.Prev, func(id ID, bb *Block, stk *Stakes, rank *big.Int) error {
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

		//@TODO if both pref and fprev have been found, stop walk early
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
		return ErrTimestampNotAfterPrev
	}

	// the blocks round must be after the prev round
	if prev.Round >= b.Round {
		return ErrRoundNrNotAfterPrev
	}

	// check if there were other blocks that the propser should have used as prev
	for r := prev.Round; r < b.Round; r++ {
		//@TODO check if we know of any other block in a round in between the two rounds
		//that could have been used as a prev?
		//@TODO is this check still important if token randomness is based on fprev
		//@TODO return early so proposers cannot ddos the acceptor
	}

	//calculate the resulting rank, it must be higher then zero
	rank := b.Rank(stake)
	if rank.Sign() <= 0 {
		return ErrZeroRank
	}

	// @TODO check if the round nr makes sense (together with timestamp?) what happens if
	// a very high round nr is proposed with a very recent timestamp (prev+1). Others
	// would check that their timestamp would be after the block, if not they wouldn't
	// vote on it?

	// write the actual block and rank
	err = tx.Write(b, nil, rank)
	if err != nil {
		return fmt.Errorf("failed to write block: %v", err)
	}

	//re-weigh all rounds upwards
	//@TODO (optimization) we should call weigh in batches, else the cost of running
	//it grows super fast with tall rounds
	//@TODO (optimization) we should allow for a max nr of top blocks per round, past
	//the total points we hand out per round it it not really effective to rank them anymore
	//@TODO (optimization) we would like to add this limit using a vrf threshold so
	//members know they don't even need to send it
	err = c.weigh(tx, b.Round)
	if err != nil {
		return fmt.Errorf("failed to weigh rounds: %v", err)
	}

	// [MAJOR] distribute stake to all ancestors for finalization
	// - figure out what the total stake deposit is for the network
	// - check if this block provides the majority stake for the prev block
	// - if so, finalize this block and all blocks before it
	// - update our current state to this finalized chain

	return tx.Commit()
}

//Tip returns the current heaviest chain of blocks
func (c *Chain) Tip() ID {
	c.wmu.RLock()
	defer c.wmu.RUnlock()
	return c.tip
}

//Read a block from the chain
func (c *Chain) Read(id ID) (b *Block, weight uint64, err error) {
	tx := c.store.CreateTx(false)
	defer tx.Discard()
	b, _, _, err = tx.Read(id)
	if err != nil {
		return nil, 0, err
	}

	c.wmu.RLock()
	defer c.wmu.RUnlock()
	w, ok := c.weights[id]
	if !ok {
		return nil, 0, ErrNotWeighted
	}

	return b, w, nil
}

// Weigh all blocks from the the specified round upwards and change the current
// longest tip to the block with the most weight behind it.
func (c *Chain) Weigh(nr uint64) (err error) {
	tx := c.store.CreateTx(false)
	defer tx.Discard()
	return c.weigh(tx, nr)
}

func (c *Chain) weigh(tx Tx, nr uint64) (err error) {
	for rn := nr; rn <= tx.MaxRound(); rn++ {
		type bl struct {
			prev ID
			rank *big.Int
			id   ID
		}

		//read blocks of the round, specifically the rank
		var blocks []*bl
		if err = tx.Round(rn, func(id ID, b *Block, stk *Stakes, rank *big.Int) error {
			blocks = append(blocks, &bl{rank: rank, prev: b.Prev, id: id})
			return nil
		}); err != nil {
			return fmt.Errorf("failed to read blocks from round: %v", err)
		}

		//sort by rank
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].rank.Cmp(blocks[j].rank) > 0
		})

		//now with the new pos, determine weight
		c.wmu.Lock()
		for i, b := range blocks {
			w := c.points / uint64(i+1)
			prevw, ok := c.weights[b.prev]
			if !ok {
				c.wmu.Unlock()
				return fmt.Errorf("encountered a prev block '%.10x' without a weight", b.prev)
			}

			sumw := prevw + w
			c.weights[b.id] = sumw

			//if sum-weight heigher or equal the the current tip sum-weight use that
			//as the new tip. By also replacing on equal we prefer newly calculated
			//weights over the old maximum
			if sumw >= c.tipw {
				c.tip = b.id
				c.tipw = sumw
			}
		}

		c.wmu.Unlock()
	}

	return
}
