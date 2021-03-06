package chain

import (
	"sort"
	"sync"

	"github.com/advanderveer/27067dd17/onl/thr"
	"github.com/advanderveer/27067dd17/onl/wall"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/pkg/errors"
)

//GenesisBlock will initialize a genesis block
func GenesisBlock(t [vrf.Size]byte, god *wall.Identity) (b *wall.Block) {
	b = &wall.Block{
		ID: wall.BID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}

	return god.SignBlock(b, t)
}

//MemChain implements a blockchain that lives purely in memory
type MemChain struct {
	genesis wall.BID
	params  *wall.Params

	//mutable state
	tip     wall.BID
	rounds  map[uint64][]wall.BID
	maxr    uint64
	idns    map[uint64]map[wall.PK]struct{}
	blocks  map[wall.BID]*wall.Block
	weights map[wall.BID]uint64
	mu      sync.RWMutex
}

//NewMemChain creates an in-memory block chain
func NewMemChain(p *wall.Params) (c *MemChain) {
	c = &MemChain{
		params:  p,
		blocks:  make(map[wall.BID]*wall.Block),
		rounds:  make(map[uint64][]wall.BID),
		idns:    make(map[uint64]map[wall.PK]struct{}),
		weights: make(map[wall.BID]uint64),
	}

	//create a genesis block, signed by set of well-known non-random keys
	gen := GenesisBlock(p.GenesisTicket, wall.NewIdentity(p.GenesisTicket[:], nil))
	c.genesis = gen.ID
	c.blocks[gen.ID] = gen
	c.rounds[0] = []wall.BID{gen.ID}
	c.tip = gen.ID
	c.weights[gen.ID] = c.params.WeightPoints
	c.weights[gen.Vote.Prev] = 0

	return
}

// Tip returns the current tip for the block chain
func (c *MemChain) Tip() wall.BID { return c.tip }

// Genesis returns the genesis block id
func (c *MemChain) Genesis() wall.BID { return c.genesis }

// Mint a block that will extend the chain from the provided prev, it returns a
// block that can be further modified before calling the signf to finalize it.
func (c *MemChain) Mint(prev wall.BID, round, ts uint64, idn *wall.Identity) (b *wall.Block, signf func() *wall.Block, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	//if zero bid mint on our current tip
	if prev == wall.NilBID {
		prev = c.tip
	}

	//read the prev block, it must exist
	prevb, err := c.read(prev)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read the given prev block")
	}

	//find the deposit transfer and output index
	dtr, di := c.findDeposit(prev, round, idn)
	if dtr == nil {
		return nil, nil, ErrDepositNotFound
	}

	//create new block
	b = &wall.Block{
		Vote: wall.Vote{
			Round:     round,
			Prev:      prevb.ID,
			Deposit:   wall.Ref(dtr.ID, di),
			Timestamp: ts,
		},
	}

	//add witnesses
	if prevr, ok := c.rounds[prevb.Vote.Round]; ok {
		for _, wid := range prevr {
			if wid == prev {
				continue //prev block cannot also be a witness
			}

			wb, err := c.read(wid)
			if err != nil {
				panic("chain: failed to read a round block: " + err.Error())
			}

			//add the prev's block sibling as witness
			b.Witness = append(b.Witness, &wb.Vote)
		}
	}

	return b, func() *wall.Block {
		return idn.SignBlock(b, prevb.Ticket.Token)
	}, nil
}

// Walk the chain from the provided block towards the genesis
func (c *MemChain) Walk(id wall.BID, update bool, f func(b *wall.Block) error) (err error) {
	if update {
		c.mu.Lock()
		defer c.mu.Unlock()
	} else {
		c.mu.RLock()
		defer c.mu.RUnlock()
	}

	return c.walk(id, f)
}

func (c *MemChain) walk(id wall.BID, f func(b *wall.Block) error) (err error) {
	for {
		b, err := c.read(id)
		if err != nil {
			return err
		}

		err = f(b)
		if err != nil {
			return err
		}

		if id == c.genesis {
			return nil //we reached the genesis
		}

		id = b.Vote.Prev
	}
}

// Read a specific block from the chain
func (c *MemChain) Read(id wall.BID) (b *wall.Block, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.read(id)
}

func (c *MemChain) read(id wall.BID) (b *wall.Block, err error) {
	b, ok := c.blocks[id]
	if !ok {
		return nil, ErrBlockNotExist
	}

	return b, nil
}

// Append a block to the chain
func (c *MemChain) Append(b *wall.Block) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// cheap verification step
	if ok, err := b.VerifyCheap(); !ok {
		return errors.Wrap(err, "failed to verify cheap fields")
	}

	// the block itself must not exist
	existing, _ := c.read(b.ID)
	if existing != nil {
		return ErrBlockExists
	}

	// the previous block must exist
	prevb, err := c.read(b.Vote.Prev)
	if err != nil {
		return errors.Wrap(err, "failed to read prev block")
	}

	// verification based on prev block
	if ok, err := b.VerifyAgainstPrev(&prevb.Vote, prevb.Ticket.Token); !ok {
		return errors.Wrap(err, "failed to verify against prev")
	}

	// each voter is only allowed one vote per round
	if _, ok := c.idns[b.Vote.Round][b.Vote.Voter]; ok {
		return ErrVoterAlreadyVoted
	}

	// index the unspent transfers to verify the block
	utro := c.indexUTRO(prevb.ID)

	// verify the remainder of the block
	if ok, err := b.VerifyAgainstUTRO(utro, c.params); !ok {
		return errors.Wrap(err, "failed to verify against utro")
	}

	// finally, write the block to our chain
	c.writeBlock(b)

	// re-select the longest chain
	c.selectLongestChain(b.Vote.Round)

	return
}

// private functionality

func (c *MemChain) selectLongestChain(n uint64) {
	tipw := c.weights[c.tip]

	for rn := n; rn <= c.maxr; rn++ {
		round := c.rounds[rn]
		if round == nil {
			continue //no such round
		}

		//sort by decimal representation of the ticket's token
		sort.Slice(round, func(i, j int) bool {
			ib := c.blocks[round[i]]
			jb := c.blocks[round[j]]
			if jb == nil || ib == nil {
				panic("chain: missing block for round sorting")
			}

			it := thr.Dec(c.params.DecimalContext, ib.Ticket.Token[:])
			jt := thr.Dec(c.params.DecimalContext, jb.Ticket.Token[:])

			return it.Cmp(jt) > 0
		})

		//assign each block in the round a weight based on its sorted position
		for i, bid := range round {
			b := c.blocks[bid] //missing blocks will panic on sort

			// weight is determined by the ranking in the round
			w := c.params.WeightPoints / uint64(i+1)
			prevw, ok := c.weights[b.Vote.Prev]
			if !ok {
				panic("chain: missing weight for tip selection")
			}

			// sum weight is the previous weight plus its own weight
			sumw := prevw + w
			c.weights[bid] = sumw

			//if sum-weight heigher or equal the the current tip sum-weight use that
			//as the new tip. By also replacing on equal we prefer newly calculated
			//weights over the old maximum
			if sumw >= tipw {
				c.tip = bid
				tipw = c.weights[c.tip]
			}
		}
	}
}

func (c *MemChain) indexUTRO(id wall.BID) (utro *wall.UTRO) {
	utro = wall.NewUTRO()
	spent := map[wall.OID]struct{}{}
	c.walk(id, func(b *wall.Block) (err error) {
		for i := len(b.Transfers) - 1; i >= 0; i-- {
			tr := b.Transfers[i]

			//keep a map of all spent inputs
			for _, in := range tr.Inputs {
				spent[in] = struct{}{}
			}

			//any output that is not in that map can be considered unspent
			for oi, out := range tr.Outputs {
				oid := wall.Ref(tr.ID, uint64(oi))
				if _, ok := spent[oid]; ok {
					continue //output is spent
				}

				utro.Put(oid, out)
			}
		}

		return
	})

	return
}

func (c *MemChain) findDeposit(bid wall.BID, round uint64, idn *wall.Identity) (dtr *wall.Tr, outi uint64) {
	c.walk(bid, func(b *wall.Block) (err error) {
		depth := round - b.Vote.Round

		if depth >= c.params.MaxDepositTTL {
			return // too deep for any valid deposit to appear
		}

		for _, tr := range b.Transfers {
			if tr.Sender != idn.PublicKey() {
				continue // deposit is always sent by ourselves
			}

			for i, out := range tr.Outputs {
				if out.Receiver != idn.PublicKey() {
					continue // deposit must be sent to ourselves
				}

				if ok, _ := out.UsableDepositFor(round, c.params.MaxDepositTTL); !ok {
					continue
				}

				dtr = tr
				outi = uint64(i)
				return nil
			}
		}

		return nil
	})

	return
}

func (c *MemChain) writeBlock(b *wall.Block) {
	c.blocks[b.ID] = b
	c.weights[b.ID] = 0

	round, _ := c.rounds[b.Vote.Round]
	round = append(round, b.ID)
	c.rounds[b.Vote.Round] = round

	indround, _ := c.idns[b.Vote.Round]
	if indround == nil {
		indround = make(map[wall.PK]struct{})
		if b.Vote.Round > c.maxr {
			c.maxr = b.Vote.Round
		}
	}

	indround[b.Vote.Voter] = struct{}{}
	c.idns[b.Vote.Round] = indround
}
