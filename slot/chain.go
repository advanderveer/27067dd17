package slot

import (
	"math/big"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

//Ticket describes verifiable random lottery draws and the roles that it grands
type Ticket struct {
	Data     []byte //actual vrf data
	Proof    []byte //proof of the data
	Notarize bool   //true if the ticket grants notarization rights
	Propose  bool   //true if the ticket grants block proposer rights
}

// Chain describes the state of our algorithm though blocks linked together and
// placed into discrete rounds. Consensus is reached by every proposer building
// on tips with highest strength which is drawn in private using a VRF.
type Chain struct {
	tip    ID                         //block wich currently has the highest strength
	blocks map[ID]*Block              //stores all blocks this chain knows of
	rounds map[uint64]map[ID]struct{} //maps blocks to rounds
	ranks  map[ID]int                 //stores the rank of each block in its respective round

	mu sync.RWMutex
}

// NewChain intializes a chain
func NewChain() (c *Chain, err error) {
	c = &Chain{
		tip:    NilID,
		blocks: make(map[ID]*Block),
		rounds: make(map[uint64]map[ID]struct{}),
		ranks:  make(map[ID]int),
	}

	//create genesis block in round 0
	genb := NewBlock(0, NilID, NilTicket, NilProof, NilPK)
	c.tip = genb.Hash()
	c.blocks[c.tip] = genb
	c.rounds[0] = map[ID]struct{}{c.tip: struct{}{}}
	c.ranks[c.tip] = 1

	return
}

// Progress the chain to another round
// func (c *Chain) Progress(round uint64) {
// 	atomic.StoreUint64(&c.round, round)
// }

// Round returns the current round
// func (c *Chain) Round() uint64 {
// 	return atomic.LoadUint64(&c.round)
// }

// Tip returns the current tip
func (c *Chain) Tip() (tip ID) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

// Verify the block with an error explaining what went wrong
func (c *Chain) Verify(b *Block) (ok bool, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	//@TODO b.Prev block is not protected by a signature?
	//@TODO b.Round is not protected by a signature?

	//@TODO verify that the picked prev is not too far in the past, this
	//could result in an artifical low threshold that the user may force itself
	//into for the current round? i.e: create a block for this round based on the
	//genesis block. All open blocks in between cause will cause the threshold to
	//be very low

	//get prev for block, must exist to validate
	prevb := c.read(b.Prev)
	if prevb == nil {
		return false, ErrPrevNotExist
	}

	//check if round is current round
	//@TODO unless it is a block notarization? this is allowed in any current round
	// if b.Round != c.round {
	// 	return false, ErrWrongRound
	// }

	//check if the prevb was the blocks round -1
	//@TODO except if a round didn't yield a block, check instead if there was no
	//block between this round and the round as referenced by prev
	// if prevb.Round != b.Round-1 {
	// 	return false, ErrPrevWrongRound
	// }

	seed := Seed(prevb, b.Round)

	//Verify the notarization proof
	if !vrf.Verify(b.NtPK[:], seed, b.NtTicket[:], b.NtProof[:]) {
		return false, ErrNotarizeProof
	}

	//Verify the proposer proof
	if !vrf.Verify(b.PK[:], seed, b.Ticket[:], b.Proof[:]) {
		return false, ErrProposeProof
	}

	//check that the ticket has indeed granted access to to ticket proposal or notarization
	//check if there is another block at its round that has a higher prio
	//check if the signer of the block already provided another block for the round

	return true, nil
}

// Draw a ticket using a VRF based on the current round and randomness from a
// past block. This represents a lottery that each member of the network can
// run for itself and then present the proof to others.
func (c *Chain) Draw(pk []byte, sk *[vrf.SecretKeySize]byte, prev ID, round uint64) (t Ticket, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	prevb := c.read(prev)
	if prevb == nil {
		return t, ErrPrevNotExist
	}

	//@TODO consider adding the prev hash as seed as well. It might be possible
	//for an advisary to forge a block ticket that will win him the next round
	//also? Wouldn't this be the case with having the prev hash as well?

	//calculate the ticket's seed
	seed := Seed(prevb, round)

	//draw a new ticket, use previoub block ticket as seed
	t.Data, t.Proof = vrf.Prove(seed, sk)

	//@TODO base role on the threshold function
	_ = c.Threshold(10, prev)
	t.Propose = true
	t.Notarize = true
	return
}

// Threshold walks the ancestory for the last N blocks and determines the threshold
// value for the next ticket draw.
func (c *Chain) Threshold(depth uint64, id ID) (t [TicketSize]byte) {

	//@TODO implement, for now the threshold is always the ticket zero value, e.g
	//everyone will be selected

	//the basic logic is: if we got many members in the network the priority will
	//be low so we can move the priority to that value as a moving average. If we
	//we didn't get any blocks for a given time we conclude that it became too hard
	//for proposers to draw enough value. this means we need the concept of
	//rounds to capture the absence of any proposer

	//we start to look at the distance between high priority tickets and the threshold
	//at that time. While walking the chain look at the rounds and capture gaps as
	//as the value on or over the threshold.

	return t
}

// Strength will calculate the cummulative strength by adding the strength of
// the provided block with the strength of it ancestory
func (c *Chain) Strength(id ID) (s *big.Rat, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strength(id)
}

func (c *Chain) strength(id ID) (s *big.Rat, err error) {

	//@TODO We would prefer to cache the strength of a block but it can change
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
	tipStrength, err := c.strength(c.tip)
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

	for {
		b = c.read(id)
		if b == nil {
			return ErrPrevNotExist
		}

		err = f(id, b, c.rank(id))
		if err != nil {
			return err
		}

		id = b.Prev
		if id == NilID {
			return nil //done
		}
	}
}

// Rank returns the rank of a given block in its round
func (c *Chain) Rank(id ID) (rank int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rank(id)
}

func (c *Chain) rank(id ID) (rank int) {
	var ok bool
	rank, ok = c.ranks[id]
	if !ok {
		panic("found block but no rank")
	}

	if rank < 1 {
		panic("read impossible rank")
	}

	return
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
