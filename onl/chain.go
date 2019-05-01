package onl

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/dgraph-io/badger"
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

	//@TODO move weights to badger store
	weights map[ID]uint64
	wmu     sync.RWMutex

	//state of the tip we're on
	tstate unsafe.Pointer //*State
}

//NewChain creates a new Chain
func NewChain(s Store, genr uint64, genfs ...func(kv *KV)) (c *Chain, gen ID, err error) {
	c = &Chain{
		points:  1000, //@TODO make this configurable?
		store:   s,
		weights: make(map[ID]uint64),
	}

	//genesis prev weight is 0
	c.weights[NilID] = 0

	//try to read genesis block and state
	tx := c.store.CreateTx(true)
	defer tx.Discard()

	//then genesis stuff
	if err := tx.Round(genr, func(id ID, b *Block, stk *Stakes, rank *big.Int) error {
		if c.genesis.Block != nil {
			return fmt.Errorf("there is more then just the genesis block")
		}

		//read genesis block info
		c.genesis.Block = b
		c.genesis.Stakes = stk
		c.genesis.id = id
		return nil
	}); err != nil {
		return nil, gen, fmt.Errorf("failed to read round 0 for genesis block: " + err.Error())
	}

	//if no genesis could be read, create from empty state
	if c.genesis.Block == nil {
		c.genesis.Block = &Block{
			//@TODO the token should be a crypto random token as it determines the
			//randomness of identities that deposit in this block
			Token: []byte("vi veri veniversum vivus vici"),
			Round: genr,
		}

		st, err := NewState(nil)
		if err != nil {
			return nil, gen, fmt.Errorf("failed to setup empty state for genesis update: %v", err)
		}

		var deposits uint64
		for _, genf := range genfs {
			w := st.Update(genf)
			deposits += w.TotalDeposit()
			c.genesis.Block.AppendWrite(w)
		}

		c.genesis.Stakes = &Stakes{Sum: deposits} //@TODO finalize block

		//write the genesis block
		if err := tx.Write(c.genesis.Block, c.genesis.Stakes, big.NewInt(1)); err != nil {
			return nil, gen, fmt.Errorf("failed to write genesis block: %v", err)
		}

		//set the tip tip to be the genesis block
		c.genesis.id = c.genesis.Hash()
		if err := tx.WriteTip(c.genesis.id, 0); err != nil {
			return nil, gen, fmt.Errorf("failed to write genesis tip weight: %v", err)
		}
	}

	//re-run weight calculations for the whole chain
	//@TODO (optimization) we don't want to do this every time the system boots up
	err = c.weigh(tx, genr)
	if err != nil {
		return nil, gen, fmt.Errorf("failed to weigh chain blocks: %v", err)
	}

	//commit the genesis block and tip
	if err := tx.Commit(); err != nil {
		return nil, gen, fmt.Errorf("failed to commit writing of genesis block: %v", err)
	}

	return c, c.genesis.id, nil
}

//Genesis returns the genesis block
func (c *Chain) Genesis() (b *Block) { return c.genesis.Block }

// State returns the state represented by the chain walking back to the genesis.
// If the provided id is the NilID it will create a state from the current tip.
func (c *Chain) State(id ID) (tip ID, s *State, err error) {

	//@TODO (optimization) we would like to keep a cache of the state on the longest
	//finalized chain

	tx := c.store.CreateTx(false)
	defer tx.Discard()
	return c.state(tx, id)
}

func (c *Chain) state(tx Tx, id ID) (tip ID, s *State, err error) {
	tip, _, err = tx.ReadTip()
	if err != nil {
		return NilID, nil, fmt.Errorf("failed to read tip to state from: %v", err)
	}

	if tip == NilID {
		return NilID, nil, fmt.Errorf("no tip available, please specify explicitely")
	}

	if id == NilID {
		id = tip
	}

	var log [][]*Write
	if err = c.walk(tx, id, func(id ID, bb *Block, stk *Stakes, rank *big.Int) error {
		log = append([][]*Write{bb.Writes}, log...)
		return nil
	}); err != nil {
		return NilID, nil, err
	}

	s, err = NewState(log)
	return tip, s, err
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

// View values from the key-value state of the current tip
func (c *Chain) View(f func(kv *KV)) {
	(*State)(atomic.LoadPointer(&c.tstate)).View(f)
}

// Update values on the key-value state of the current tip and return a new write
func (c *Chain) Update(f func(kv *KV)) (w *Write) {
	return (*State)(atomic.LoadPointer(&c.tstate)).Update(f)
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

	// prev/stable blocks
	var (
		stable  *Block
		prev    *Block
		prevStk *Stakes
	)

	// walk prev chain while storing all blocks up to the genesis a log
	if err = c.walk(tx, b.Prev, func(id ID, bb *Block, stk *Stakes, rank *big.Int) error {
		if b.Prev == id {
			prev = bb
			prevStk = stk
		}

		for _, w := range bb.Writes {
			//@TODO make sure the prev block is before the deposit block
			if w.HasDepositFor(b.PK) {
				stable = bb
			}
		}

		//@TODO if both pref and deposit have been found, stop walk early

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk prev chain: %v", err)
	}

	// stable randomness block must exist
	if stable == nil {
		return ErrStableNotInChain
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

	// @TODO check if the round nr makes sense (together with timestamp?) what happens if
	// a very high round nr is proposed with a very recent timestamp (prev+1). Others
	// would check that their timestamp would be after the block, if not they wouldn't
	// vote on it?

	// check if there were other blocks that the proposer should have used as prev
	for r := prev.Round; r < b.Round; r++ {
		//@TODO check if we know of any other block in a round in between the two rounds
		//that could have been used as a prev?
		//@TODO return early so proposers cannot ddos the acceptor
	}

	//reconstruct the state to validate the writes in the new block
	//@TODO we're walking again here but we would like to re-use our code also
	_, state, err := c.state(tx, b.Prev)
	if err != nil {
		return ErrStateReconstruction
	}

	//read dynamic data from rebuild state
	var stake uint64
	var tpk []byte
	state.View(func(kv *KV) {
		stake, tpk = kv.ReadStake(b.PK)
		// @TODO read the vrf threshold (if any)
	})

	//check if there was any token pk comitted
	if tpk == nil {
		return ErrNoTokenPK
	}

	//validate the token
	//@TODO it takes a lot of effort to get to this validation point, can members
	//mis-use this to ddos the network?
	if !b.VerifyToken(tpk, stable.Hash()) {
		return ErrInvalidToken
	}

	//calculate the resulting rank, it must be higher then zero
	rank := b.Rank(stake)
	if rank.Sign() <= 0 {
		return ErrZeroRank
	}

	//validate each write in the block by applying them to the new state
	var deposit uint64
	for _, w := range b.Writes {

		//apply the writes so we can check their validity
		err = state.Apply(w, false)
		if err != nil {
			return err
		}

		//increment thet total amount of stake stored in this block
		deposit += w.TotalDeposit()
	}

	//add the prev's total deposit to this block's deposit
	stk := &Stakes{Sum: prevStk.Sum + deposit}

	// all is well, write the actual block with its rank
	err = tx.Write(b, stk, rank)
	if err != nil {
		return fmt.Errorf("failed to write block: %v", err)
	}

	//re-weigh all rounds upwards
	//@TODO (optimization) we should call weigh in batches, else the cost of running
	//      it grows super fast with tall rounds
	//@TODO (optimization) we should allow for a max nr of top blocks per round, past
	//      the total points we hand out per round it it not really effective to rank them anymore
	//@TODO (optimization) we would like to add this limit using a vrf threshold so
	//      honest members know they don't even need to send it
	//@TODO (optimization) we would rather not calculate the whole state again
	err = c.weigh(tx, b.Round)
	if err != nil {
		return fmt.Errorf("failed to weigh rounds: %v", err)
	}

	// cast stake votes
	// @TODO walk backwards to cast this block's signer's stake on all ancestors as
	//    votes
	// @TODO whenever the sum of unique stake casters surpasses the majority threshold
	//    we can finalize it.

	//finally, attempt to commit
	err = tx.Commit()
	if err != nil {
		if err == badger.ErrConflict {
			return ErrAppendConflict
		}

		return fmt.Errorf("failed to commit append tx: %v", err)
	}

	return
}

//Tip returns the current heaviest chain of blocks
func (c *Chain) Tip() (tip ID) {
	tx := c.store.CreateTx(false)
	defer tx.Discard()
	tip, _, _ = tx.ReadTip()
	return
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

//ForEach will call f for each block in all rounds >= to the start round
func (c *Chain) ForEach(start uint64, f func(id ID, b *Block) (err error)) (err error) {
	tx := c.store.CreateTx(true)
	defer tx.Discard()
	return c.forEach(tx, start, f)
}

func (c *Chain) forEach(tx Tx, start uint64, f func(id ID, b *Block) (err error)) (err error) {
	if start == 0 {
		//when zero, the user either states it is acutally zero in which case the following operation
		//does nothing or the users means it is the min round
		start = tx.MinRound()
	}

	for rn := start; rn <= tx.MaxRound(); rn++ {
		if err = tx.Round(rn, func(id ID, b *Block, stk *Stakes, rank *big.Int) error {
			return f(id, b)
		}); err != nil {
			return fmt.Errorf("failed to read blocks from round %d: %v", rn, err)
		}
	}

	return
}

// Weigh all blocks from the the specified round upwards and change the current
// longest tip to the block with the most weight behind it.
func (c *Chain) Weigh(nr uint64) (err error) {
	tx := c.store.CreateTx(true)
	defer tx.Discard()
	err = c.weigh(tx, nr)
	if err != nil {
		return
	}

	return tx.Commit()
}

func (c *Chain) weigh(tx Tx, nr uint64) (err error) {
	_, tipw, err := tx.ReadTip()
	if err != nil {
		return fmt.Errorf("failed to read current tip: %v", err)
	}

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
			if sumw >= tipw {

				//write the new tip
				err = tx.WriteTip(b.id, sumw)
				if err != nil {
					return fmt.Errorf("failed to write new tip: %v", err)
				}

				//build state for new tip
				_, tstate, err := c.state(tx, NilID)
				if err != nil {
					return fmt.Errorf("failed to re-build state from tip: %v", err)
				}

				//atomically assign the state
				atomic.StorePointer(&c.tstate, unsafe.Pointer(tstate))
			}
		}

		c.wmu.Unlock()
	}

	return
}
