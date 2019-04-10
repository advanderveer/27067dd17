package topn

import (
	"math/big"
	"sort"
)

type roundb struct {
	id    ID
	rank  *big.Int
	block *Block
}

//@TODO optimization, make storing millions of identities
//and blocks more efficient, bloom filter or bitmap?
type round struct {
	blocks map[ID]*roundb
	idns   map[PK]struct{}
}

func newRound() *round {
	return &round{
		blocks: make(map[ID]*roundb),
		idns:   make(map[PK]struct{}),
	}
}

func (r *round) Set(id ID, b *Block, rank *big.Int) {
	r.blocks[id] = &roundb{id: id, rank: rank, block: b}
	r.idns[b.PK] = struct{}{}
}

func (r *round) HasBlock(id ID) (ok bool) {
	_, ok = r.blocks[id]
	return
}

func (r *round) SawIdentity(pk PK) (ok bool) {
	_, ok = r.idns[pk]
	return
}

func (r *round) Ranked(f func(pos int, id ID, b *Block)) {
	blocks := make([]*roundb, 0, len(r.blocks))
	for _, b := range r.blocks {
		blocks = append(blocks, b)
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].rank.Cmp(blocks[j].rank) > 0
	})

	for i, b := range blocks {
		f(i+1, b.id, b.block)
	}
}
