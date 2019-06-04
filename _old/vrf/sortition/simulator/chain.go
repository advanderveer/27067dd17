package simulator

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

// BlockChain receives blocks and comes to consensus on the longest chain
type BlockChain struct {
	blocks      map[ID]*Block
	rwmu        sync.RWMutex
	genesis     ID
	tip         ID
	tipStrength *big.Int
}

//NewBlockChain creates a new chain
func NewBlockChain() (bc *BlockChain, err error) {
	bc = &BlockChain{
		blocks: make(map[ID]*Block),
	}

	bc.rwmu.Lock()
	defer bc.rwmu.Unlock()

	//@TODO add a random seed for the genesis block

	gen := NewBlock(big.NewInt(0), 0, nil, nil, NilID)
	bc.genesis, err = gen.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to write hash: %v", err)
	}

	bc.blocks[bc.genesis] = gen
	bc.tip = bc.genesis
	bc.tipStrength = gen.Prio()
	return
}

// Create a new block at the current tip and return it
func (bc *BlockChain) Create(sk *[vrf.SecretKeySize]byte, prev ID, weight uint64) (b *Block, err error) {
	prevb, err := bc.Read(prev)
	if err != nil {
		return nil, err
	}

	//Draw our ticket, use the seed of the previous block
	prio, widx, ticket, proof := DrawPriority(sk, prevb.ticket[:], weight)

	//Create the block (unmined)
	b = NewBlock(prio, widx, ticket, proof, prev)
	return
}

// Genesis returns the block ID of the genesis block
func (bc *BlockChain) Genesis() ID {
	bc.rwmu.RLock()
	defer bc.rwmu.RUnlock()

	return bc.genesis
}

// Tip returns the current tip
func (bc *BlockChain) Tip() ID {
	bc.rwmu.RLock()
	defer bc.rwmu.RUnlock()

	return bc.tip
}

// Walk backwards visit all blocks leading up to 'id'
func (bc *BlockChain) Walk(id ID, f func(b *Block) error) (err error) {
	var b *Block

	for {
		b, err = bc.Read(id)
		if err != nil {
			return err
		}

		err = f(b)
		if err != nil {
			return err
		}

		id = b.prev
		if id == NilID {
			return nil //done
		}
	}
}

//Verify checks the integrity of the block
func (bc *BlockChain) Verify(pk []byte, weight uint64, b *Block) (ok bool, err error) {
	prevb, err := bc.Read(b.prev)
	if err != nil {
		return false, fmt.Errorf("previous block could not be read: %v", err)
	}

	ok = VerifyPriority(pk, prevb.ticket[:], b.ticket[:], b.proof[:], b.widx, b.Prio())
	if !ok {
		return false, fmt.Errorf("priority could not be verified")
	}

	//@TODO double check if this doesn't contain an off-by-one error, we now assert
	//on the safe side
	if b.widx >= weight {
		return false, fmt.Errorf("weight index is larger then the signer's weight")
	}

	//@TODO the main insight is that what we adjust over time is a difficulty curve?
	//what happens if we have a variable prio treshold that is adjusted every N
	//blocks?
	//@TODO how do we adjust the systems min/max d, what is the curve between them
	//@TODO past priorities show us several points on the PoW curve, from these points
	//we can solve for what the adjusted curve (and max) should be
	//would we consider a min that is higher then 0? maybe to slow down?
	//@TODO what kind of block time are we adjusting to? What kind of orphan rate
	//would we expect? https://www.reddit.com/r/litecoin/comments/4ovb23/what_is_the_average_orphan_rate_of_the_litecoin/

	//@TODO can we enforce a minimum (expected) priority?
	//@TODO verify proof of work

	return true, nil
}

// Append the block to the chain, if it represents a block with more weight
// then our current tip we stop current mining and switch over to this block
func (bc *BlockChain) Append(b *Block) (id ID, err error) {
	id, err = b.Hash()
	if err != nil {
		return NilID, fmt.Errorf("failed to hash block: %v", err)
	}

	//determine the block's strength
	strength := b.Prio()
	if err = bc.Walk(b.prev, func(b *Block) error {
		strength.Add(strength, b.Prio())
		return nil
	}); err != nil {
		return NilID, fmt.Errorf("failed to determine strength: %v", err)
	}

	bc.rwmu.Lock()
	defer bc.rwmu.Unlock()

	bc.blocks[id] = b
	if bc.tipStrength.Cmp(strength) < 0 {
		bc.tip = id
		bc.tipStrength = strength

		//@TODO stop mining
	}

	return
}

//Read a block with a given hash from the chain.
func (bc *BlockChain) Read(id ID) (b *Block, err error) {
	bc.rwmu.RLock()
	defer bc.rwmu.RUnlock()

	return bc.read(id)
}

func (bc *BlockChain) read(id ID) (b *Block, err error) {
	b, ok := bc.blocks[id]
	if !ok {
		return nil, ErrBlockNotExist
	}

	return
}
