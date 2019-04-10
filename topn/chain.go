package topn

import (
	"math/big"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

// Chain is our consensus data structure
type Chain struct {
	points  uint64
	rounds  map[uint64]*round
	tip     ID
	tipw    uint64
	curr    uint64
	gen     ID
	mu      sync.RWMutex
	weights map[ID]uint64
}

// NewChain initializes the chain structure
func NewChain(g *Block) (c *Chain) {
	c = &Chain{
		points:  1000, //the amount of points we hand out in each round
		rounds:  make(map[uint64]*round),
		weights: make(map[ID]uint64),
	}

	//setup the genesis block in round 0
	c.curr = 0
	c.rounds[c.curr] = newRound()
	c.gen = g.Hash()
	c.rounds[c.curr].Set(c.gen, g, g.Rank(1))
	c.tip = c.gen
	c.tipw = 0
	c.Advance()

	return
}

// Round returns the current round
func (c *Chain) Round() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.curr
}

// Advance to the next round
func (c *Chain) Advance() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.curr++
	c.rounds[c.curr] = newRound()
}

// Tip returns the current heaviest tip in the chain
func (c *Chain) Tip() (tip ID) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

// Balance returns the account balance of an identity as of the provided tip.
func (c *Chain) Balance(tip ID, pk PK) (b uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.balance(tip, pk)
}

func (c *Chain) balance(tip ID, pk PK) (b uint64) {
	//@TODO walk back into the chain and gather and sum all the account sends
	//into a total balance
	return 1
}

// Threshold return the vrf threshold for a certain tip.
func (c *Chain) Threshold(tip ID) *big.Int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.threshold(tip)
}

func (c *Chain) threshold(tip ID) *big.Int {
	//@TODO read back and see what difficulty we expect based on timing and the
	//chain we're building on
	return big.NewInt(0)
}

// Read block data and current weight
func (c *Chain) Read(id ID) (b *Block, weight uint64, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, weight = c.read(id)
	if b == nil {
		return nil, 0, ErrBlockNotExist
	}

	return
}

func (c *Chain) read(id ID) (b *Block, weight uint64) {
	round, ok := c.rounds[id.Round()]
	if !ok {
		return
	}

	rb, ok := round.blocks[id]
	if !ok {
		return
	}

	return rb.block, c.weights[id]
}

// Append a block to the chain.
func (c *Chain) Append(b *Block) (ok bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// @TODO (security) we need an signature that covers the whole block else
	// anyone relaying could simply change the block before passing it on

	//check vrf token and proof first
	if !vrf.Verify(b.PK[:], b.Prev[:], b.Token, b.Proof) {
		return false, ErrInvalidToken
	}

	//check that the round is larger then zero, only genesis block lives there
	if b.Round < 1 {
		return false, ErrZeroRound
	}

	//check if this blocks round comes after the prev block's round
	if b.Round <= b.Prev.Round() {
		return false, ErrNonIncreasingRound
	}

	//check if the round is not in the future
	if b.Round > c.curr {
		return false, ErrFutureRound
	}

	//check if we know of at least one block that is in a newer round then the
	//the prev block. If that is the case we believe that the member who proposed
	//the block went too far in history and discard the block
	for r := b.Prev.Round(); r < b.Round; r++ {
		//@TODO we might need to put bounds on this to prevent members from sending
		//very old 'prevs' that then DDOS other members by having them iterate from very
		//far back in the chain.
		//@TODO implement, return error
	}

	//try to read prev block
	// @TODO we can remove this check once we'r walking the prev chain for
	// thresholds and balances and those calls will error if the prev doesn't exist
	prev, _ := c.read(b.Prev)
	if prev == nil {
		return false, ErrPrevNotExist
	}

	//get the users balance and rank the block by staking it all
	balance := c.balance(b.Prev, b.PK)
	rank := b.Rank(balance)
	if rank.Sign() == 0 {
		return false, ErrZeroRank
	}

	//get the ranking threshold and check if the block ranks above it
	threshold := c.threshold(b.Prev)
	if rank.Cmp(threshold) <= 0 {
		return false, ErrRankTooLow
	}

	//check if it is for a round we know about, blocks that arrive a round into
	//the future (too early) might cause this
	id := b.Hash()
	round, ok := c.rounds[id.Round()]
	if !ok {
		panic("round doesn't exist")
	}

	//if the block already exists don't add it again
	if round.HasBlock(id) {
		return false, ErrBlockExist
	}

	//if identity already proposed the round, ignore its other block
	if round.SawIdentity(b.PK) {
		return false, ErrDoublePropose
	}

	//@TODO validate all transactions in the block and make sure there is no double
	//spending or lack of balance
	//@TODO figure out how to do this efficiently

	//add to round, return whether it is new highest rank
	round.Set(id, b, rank)

	//for all rounds including and after this block's re-rank the blocks upndate all
	//sumweights of blocks after that round if we run into a block that has a
	//higher sum weight then our current tip set that block as our new tip
	for rn := b.Round; rn <= c.curr; rn++ {
		r := c.rounds[rn]
		if r == nil {
			panic("non-increasing rounds")
		}

		//re-assign sum-weights and keep the top ranking one as tip
		r.Ranked(func(pos int, id ID, b *Block) {
			w := c.points / uint64(pos) //1th get all, 2th: half, 3th a third. etc
			_, prevw := c.read(b.Prev)  //get sumw of prev block
			sumw := prevw + w           //new sumw is the sum of the above

			//re-assing sumweight
			c.weights[id] = sumw

			//if sum-weight heigher or equal the the current tip sum-weight use that
			//as the new tip. By also replacing on equal we prefer newly calculated
			//weights over the old maximum
			if sumw >= c.tipw {
				c.tip = id
				c.tipw = sumw
			}
		})
	}

	//@TODO (optimization) trim all unreferenced blocks in some rounds back to put
	//a bound on the storage requirements

	return
}
