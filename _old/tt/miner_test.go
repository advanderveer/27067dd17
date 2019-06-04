package tt_test

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

var (
	bid1 = tt.ID{}
	bid2 = tt.ID{}
	bid3 = tt.ID{}
	bid4 = tt.ID{}

	vid1 = tt.VID{}
	vid2 = tt.VID{}
	vid3 = tt.VID{}
	vid4 = tt.VID{}
	vid5 = tt.VID{}
	vid6 = tt.VID{}
)

func init() {
	bid1[0] = 0x01
	bid2[0] = 0x02
	bid3[0] = 0x03
	bid4[0] = 0x04

	vid1[0] = 0x01
	vid2[0] = 0x02
	vid3[0] = 0x03
	vid4[0] = 0x04
}

type testBlockReader struct {
	difcs  map[tt.ID]uint
	blocks map[tt.ID]*tt.Block
	mu     sync.RWMutex
}

func blockReader() *testBlockReader {
	return &testBlockReader{difcs: make(map[tt.ID]uint), blocks: make(map[tt.ID]*tt.Block)}
}

func (br *testBlockReader) Add(b *tt.Block, difc uint) {
	br.mu.Lock()
	defer br.mu.Unlock()
	id := b.Hash()
	br.blocks[id] = b
	br.difcs[id] = difc
}

func (br *testBlockReader) Difficulty(id tt.ID) (target *big.Int, err error) {
	br.mu.RLock()
	defer br.mu.RUnlock()

	difc, ok := br.difcs[id]
	if !ok {
		return nil, fmt.Errorf("no difficulty for this block in test reader")
	}

	target = big.NewInt(1)
	target.Lsh(target, uint(256-difc))
	return
}

func (br *testBlockReader) Read(id tt.ID) (b *tt.Block) {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.blocks[id]
}

func TestBlockMining(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	idn1 := tt.NewIdentity([]byte{0x01})
	idn2 := tt.NewIdentity([]byte{0x02})
	idn3 := tt.NewIdentity([]byte{0x03})
	idn4 := tt.NewIdentity([]byte{0x04})

	//invent a block
	b1 := &tt.Block{}
	br := blockReader()
	br.Add(b1, 3) //4 votes should solve this difficulty

	m1 := tt.NewMiner(br)
	m1.Feed(idn1.CreateVote(b1.Hash()))
	m1.Feed(idn2.CreateVote(b1.Hash()))
	m1.Feed(idn3.CreateVote(b1.Hash()))
	m1.Feed(idn4.CreateVote(b1.Hash()))

	b, err := m1.Next(ctx)
	test.Ok(t, err)
	bid := b.Hash()
	res := big.NewInt(0).SetBytes(bid[:])

	target, _ := br.Difficulty(b1.Hash())
	test.Assert(t, res.Cmp(target) < 0, "should solve target difficulty")
}
