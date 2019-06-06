package tt

import (
	"math/big"
	"sync"
)

//Chain stores blocks
type Chain struct {
	blocks map[ID]*Block
	mu     sync.RWMutex
	target *big.Int
}

//NewChain creates the chain structure
func NewChain(difc uint) (c *Chain) {
	c = &Chain{blocks: make(map[ID]*Block)}

	c.target = big.NewInt(1)
	c.target.Lsh(c.target, uint(256-difc))
	return c
}

//Difficulty returns the target difficulty for the proof of work
func (c *Chain) Difficulty(id ID) (target *big.Int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	//@TODO replace this with a deterministic algorithm based on chain history
	return c.target, nil
}

//Read a block from the chain
func (c *Chain) Read(id ID) (b *Block) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.blocks[id]
}

//Append a block
func (c *Chain) Append(b *Block) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := b.Hash()
	c.blocks[id] = b
}
