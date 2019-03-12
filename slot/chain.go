package slot

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/advanderveer/27067dd17/vrf"
)

//Chain describes a chain of blocks, split up in discrete rounds
type Chain struct {
	tip    ID                         //block wich currently has the highest strength
	round  uint64                     //current round this chain is in
	blocks map[ID]*Block              //stores all blocks this chain knows of
	rounds map[uint64]map[ID]struct{} //maps blocks to rounds
	ranks  map[ID]int                 //stores the rank of each block in its respective round
	mu     sync.RWMutex
}

//NewChain intializes a chain
func NewChain() (c *Chain, err error) {
	c = &Chain{
		tip:    NilID,
		round:  0,
		blocks: make(map[ID]*Block),
		rounds: make(map[uint64]map[ID]struct{}),
		ranks:  make(map[ID]int),
	}

	//create genesis block in round 0
	genb := NewBlock(c.round, NilID, NilTicket, NilProof, NilPK)
	c.tip = genb.Hash()
	c.blocks[c.tip] = genb
	c.rounds[c.round] = map[ID]struct{}{c.tip: struct{}{}}
	c.ranks[c.tip] = 1

	//then start in round1
	c.Progress(1)

	return
}

//Progress the chain to another round
func (c *Chain) Progress(round uint64) {
	atomic.StoreUint64(&c.round, round)
}

//Round returns the current round
func (c *Chain) Round() uint64 {
	return atomic.LoadUint64(&c.round)
}

//Tip returns the current tip
func (c *Chain) Tip() (tip ID) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttip()
}

func (c *Chain) ttip() (tip ID) {
	return c.tip
}

//Verify the block with an error explaining what went wrong
func (c *Chain) Verify(b *Block) (ok bool, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	//@TODO b.Prev block is not protected by a signature?
	//@TODO b.Round is not protected by a signature?

	//get prev for block, must exist to validate
	prevb, _ := c.read(b.Prev)
	if prevb == nil {
		return false, ErrPrevNotExist
	}

	//check if round is current round
	//@TODO unless it is a block notarization? this is allowed in any current round
	if b.Round != c.round {
		return false, ErrWrongRound
	}

	//check if the prevb was the blocks round -1
	//@TODO except if a round didn't yield a block, check instead if there was no
	//block between this round and the round as referenced by prev
	if prevb.Round != b.Round-1 {
		return false, ErrPrevWrongRound
	}

	//Verify the vrf
	if !vrf.Verify(b.PK[:], Seed(prevb, b.Round), b.Ticket[:], b.Proof[:]) {
		return false, ErrInvalidVRF
	}

	//check that the ticket has indeed granted access to to ticket proposal or notarization
	//check if there is another block at its round that has a higher prio
	//check if the signer of the block already provided another block for the round
	//check if the priority is below the difficulty threshold of the ancestor chain

	return true, nil
}

// Draw a ticket using a VRF based on the current round and randomness from a
// past block.
func (c *Chain) Draw(pk []byte, sk *[vrf.SecretKeySize]byte, prev ID) (ticket, proof []byte, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	prevb, _ := c.read(prev)
	if prevb == nil {
		return nil, nil, ErrPrevNotExist
	}

	//calculate the ticket's seed
	seed := Seed(prevb, c.round)

	//draw a new ticket, use previoub block ticket as seed
	ticket, proof = vrf.Prove(seed, sk)
	return
}

// Grant will look at the threshold difficulty as extracted by the
// chain to see if the ticket grants a specialized role
// func (c *Chain) Grant(prev ID, ticket []byte) {
// 	if len(ticket) != TicketSize {
// 		panic("unexpected ticket size provided")
// 	}
//
// 	//@TODO walk the chain for threshold values
// }

// Strength will calculate the cummulative strength by adding the strength of
// the provided block with the strength of it ancestory
func (c *Chain) Strength(id ID) (s *big.Rat, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strength(id)
}

func (c *Chain) strength(id ID) (s *big.Rat, err error) {
	//@TODO We would prefer to cache the stregth of a block but it can change
	//as adding blocks a few rounds back can change existing ranking and thus
	//score. With some finality this will probably not be a problem as we can
	//close of lower rounds but we need to look at that later

	s = new(big.Rat)
	if err = c.walk(id, func(bid ID, b *Block, rank int) error {
		s.Add(s, b.Strength(rank))
		return nil
	}); err != nil {
		return nil, err
	}

	return
}

// Append a new block unconditionally, in normal operation the block should
// first be verified for syntax and notarization.
func (c *Chain) Append(b *Block) (id ID) {
	id = b.Hash()

	//determine prev strength
	prevStrength, err := c.Strength(b.Prev)
	if err != nil {
		//@TODO instead error gracefully, or ask to verify first?
		panic("failed to determine prev strength: " + err.Error())
	}

	//when considering this block at rank one would it replace the current tip?
	prevStrength.Add(prevStrength, b.Strength(1))

	//mutate block storage protected by write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	//store block
	c.blocks[id] = b

	//store in round
	_, ok := c.rounds[b.Round]
	if !ok {
		c.rounds[b.Round] = make(map[ID]struct{})
	}

	c.rounds[b.Round][id] = struct{}{}

	//(re)calculate ranks of the round
	var round []*Block
	for id := range c.rounds[b.Round] {
		round = append(round, c.blocks[id])
	}

	Rank(round)
	for i, b := range round {
		c.ranks[b.Hash()] = i + 1
	}

	//we need to recalculate the strength after we've added the block as
	//adding the block can influence the strength of the current tip
	tipStrength, err := c.strength(c.ttip())
	if err != nil {
		panic("failed to determine tip strength: " + err.Error())
	}

	if tipStrength.Cmp(prevStrength) < 0 {
		c.tip = id
	}

	return
}

// Walk the chain towards the genesis, calling f for each block. This does not
// operate on a consistent snapshot of the block data. Blocks can be added while
// while walking causing the ranks to be inconsistent.
func (c *Chain) Walk(id ID, f func(bid ID, b *Block, rank int) error) (err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.walk(id, f)
}

func (c *Chain) walk(id ID, f func(bid ID, b *Block, rank int) error) (err error) {
	var b *Block
	var rank int

	for {
		b, rank = c.read(id)
		if b == nil {
			return ErrPrevNotExist
		}

		err = f(id, b, rank)
		if err != nil {
			return err
		}

		id = b.Prev
		if id == NilID {
			return nil //done
		}
	}
}

//Read a block and its rank from the chains, returns nil if the block doesn't exist
func (c *Chain) Read(id ID) (b *Block, rank int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.read(id)
}

func (c *Chain) read(id ID) (b *Block, rank int) {
	b, ok := c.blocks[id]
	if !ok {
		return nil, 0
	}

	rank, ok = c.ranks[id]
	if !ok {
		panic("found block but no rank")
	}

	if rank < 1 {
		panic("read impossible rank")
	}

	return
}
