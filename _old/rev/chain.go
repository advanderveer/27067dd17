package rev

import (
	"math/big"
	"sync"
)

//Chain stores linked block such that it can reach consensus on the longest chain
type Chain struct {
	//@TODO for now we configure a fixed threshold but this should be determined by
	//walking the chain and looking at past blocks and proposals
	fixedWitnessThreshold int

	genesis   map[ID]struct{}
	blocks    map[ID]*Block
	strengths map[ID]*big.Int
	mu        sync.RWMutex
	tip       ID
}

//NewChain creates the block chain
func NewChain(t int) (c *Chain) {
	c = &Chain{
		fixedWitnessThreshold: t,
		blocks:                make(map[ID]*Block),
		strengths:             make(map[ID]*big.Int),
		genesis:               make(map[ID]struct{}),
	}
	return
}

//WitnessThreshold will walk the chain and determine what nr of witnesses are
//necessary to use block 'id' as a tip.
func (c *Chain) WitnessThreshold(id ID) (t int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.genesis[id]; ok {
		return 1
	}

	return c.fixedWitnessThreshold
}

//Tip returns the current tip of the chain
func (c *Chain) Tip() ID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

//Append the block to our chain
func (c *Chain) Append(b *Block, strength *big.Int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := b.Hash()
	if b.Prev == NilID {
		c.genesis[id] = struct{}{}
		c.tip = id
	}

	c.blocks[id] = b
	c.strengths[id] = strength

	prevStrength := big.NewInt(0)
	if b.Prev != NilID {
		var err error

		//determine prev strength
		prevStrength, err = c.strength(b.Prev)
		if err != nil {
			//@TODO instead error gracefully, or ask to verify first?
			panic("failed to determine prev strength: " + err.Error())
		}

		//when considering this block at rank one would it replace the current tip?
		prevStrength.Add(prevStrength, strength)
	}

	//we need to recalculate the strength after we've added the block as
	//adding the block can influence the strength of the current tip
	tipStrength, err := c.strength(c.tip)
	if err != nil {
		panic("failed to determine tip strength: " + err.Error())
	}

	if tipStrength.Cmp(prevStrength) < 0 {
		c.tip = id
	}
}

// Each iterates over all blocks in random order, calling f for each block
func (c *Chain) Each(f func(bid ID, b *Block) error) (err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for id, b := range c.blocks {
		err = f(id, b)
		if err != nil {
			return
		}
	}

	return
}

// Walk the chain towards the genesis, calling f for each block. This does not
// operate on a consistent snapshot of the block data. Blocks can be added while
// walking causing the ranks to be inconsistent.
func (c *Chain) Walk(id ID, f func(bid ID, b *Block) error) (err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.walk(id, f)
}

func (c *Chain) walk(id ID, f func(bid ID, b *Block) error) (err error) {
	var b *Block

	for {
		b = c.read(id)
		if b == nil {
			return ErrPrevNotExist
		}

		err = f(id, b)
		if err != nil {
			return err
		}

		id = b.Prev
		if id == NilID {
			return nil //done
		}
	}
}

// Read a block and its rank from the chains, returns nil if the block doesn't exist
func (c *Chain) Read(id ID) (b *Block) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.read(id)
}

func (c *Chain) read(id ID) (b *Block) {
	b, ok := c.blocks[id]
	if !ok {
		return nil
	}

	return
}

// Strength will calculate the cummulative strength by adding the strength of
// the provided block with the strength of it ancestory
func (c *Chain) Strength(id ID) (s *big.Int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strength(id)
}

func (c *Chain) strength(id ID) (s *big.Int, err error) {

	//@TODO We would prefer to cache the strength of a block but it can change
	//as adding blocks a few rounds back can change existing ranking and thus
	//score. With some finality this will probably not be a problem as we can
	//close of lower rounds but we need to look at that later

	s = new(big.Int)
	if err = c.walk(id, func(bid ID, b *Block) error {
		s.Add(s, c.strengths[bid])
		return nil
	}); err != nil {
		return nil, err
	}

	return
}

//IntersectChain returns a new chain that only contains the blocks that are
//ALL provided chains. It does not guarntee that the chain has no gaps.
func IntersectChain(fc *Chain, cc ...*Chain) (ic *Chain, n int, err error) {
	ic = NewChain(fc.fixedWitnessThreshold)
	if err = fc.Each(func(bid ID, b *Block) error {
		for _, oc := range cc {
			bb := oc.read(bid)
			if bb == nil {
				return nil //skip adding this block
			}
		}

		//copy over
		ic.blocks[bid] = b
		ic.strengths[bid] = fc.strengths[bid]
		if b.Prev == NilID {
			ic.genesis[bid] = struct{}{}
		}

		n++
		return nil
	}); err != nil {
		return ic, n, err
	}

	return
}
