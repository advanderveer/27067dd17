package tt

import (
	"context"
	"math/big"
	"sync"
)

//BlockReader allows for reading block info from storage
type BlockReader interface {
	Read(id ID) (b *Block)
	Difficulty(id ID) (target *big.Int, err error)
}

//Miner mines blocks by consuming tip votes
type Miner struct {
	tips map[ID]map[VID]*Vote

	c  chan *Block
	br BlockReader
	mu sync.Mutex
}

//NewMiner sets up the miner
func NewMiner(br BlockReader) (m *Miner) {
	m = &Miner{
		tips: make(map[ID]map[VID]*Vote),
		c:    make(chan *Block, 1),
		br:   br,
	}

	return
}

//Next will wait for one newly mined block or error if the context expires
func (m *Miner) Next(ctx context.Context) (b *Block, err error) {
	select {
	case b = <-m.c:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

//Feed a vote into the miner. It will exhaust all combination of votes for a
//given tip in an attempt to make a block for which the hash passes a certain
//difficulty threshold
func (m *Miner) Feed(v *Vote) {
	//assume: the vote is validates
	//assume: the vote is delivered-in-order
	//assume: the tip exists in the chain

	m.mu.Lock()
	defer m.mu.Unlock()

	//store the new vote
	ex, ok := m.tips[v.Tip]
	if !ok {
		ex = make(map[VID]*Vote)
	}

	ex[v.Hash()] = v
	m.tips[v.Tip] = ex

	//make a copy
	vcopy := make(map[VID]Vote)
	for vid, vv := range ex {
		vcopy[vid] = *vv
	}

	//read difficulty for tip mining
	target, err := m.br.Difficulty(v.Tip)
	if err != nil {
		panic("failed to read mining difficulty: " + err.Error())
	}

	//start mining on the new combination of votes concurrently
	go m.mine(target, v.Tip, vcopy)
}

func (m *Miner) mine(target *big.Int, tip ID, votes map[VID]Vote) {
	b := &Block{Prev: tip, Data: tip[:]}

	//@TODO the mining logic below will still try old combinations that we already
	//know are not gonna solve the puzzle so it could be made more efficient. But
	//the algorithm sould be waiting on new votes most of the time so an optimization
	//shouldn't really profit the miner

	if b.Mine(target, votes) {
		m.c <- b
		//@TODO stop mining this tip, or would miners wanna look for a better deal?
	}
}
