package topn_test

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/topn"
	"github.com/advanderveer/go-test"
)

func TestChainBlockReading(t *testing.T) {
	g1 := &topn.Block{}
	gid := g1.Hash()
	c1 := topn.NewChain(g1)

	test.Equals(t, gid, c1.Tip())

	b2, w1, err := c1.Read(gid)
	test.Ok(t, err)
	test.Equals(t, uint64(0), w1) //genesis has zero sumweight
	test.Equals(t, g1, b2)        ///should be readable

	t.Run("non existing block because non-existing round", func(t *testing.T) {
		_, _, err = c1.Read(bid1)
		test.Equals(t, topn.ErrBlockNotExist, err)
	})

	t.Run("non existing block with existing round", func(t *testing.T) {
		id2 := topn.ID{}
		binary.BigEndian.PutUint64(id2[:], 0)
		_, _, err = c1.Read(id2)
		test.Equals(t, topn.ErrBlockNotExist, err)
	})
}

func TestChainRoundAdvancing(t *testing.T) {
	c1 := topn.NewChain(&topn.Block{})
	test.Equals(t, uint64(1), c1.Round())

	c1.Advance()
	test.Equals(t, uint64(2), c1.Round())
}

func TestMintAndAppend(t *testing.T) {
	c1 := topn.NewChain(&topn.Block{})
	idn1 := topn.NewIdentity([]byte{0x01})

	b1 := idn1.Mint(1, c1.Tip())
	test.Equals(t, uint64(1), b1.Round)
	test.Equals(t, c1.Tip(), b1.Prev)

	ok, err := c1.Append(b1)
	test.Ok(t, err)
	test.Equals(t, true, ok)

	test.Equals(t, b1.Hash(), c1.Tip())

	b2, w2, err := c1.Read(b1.Hash())
	test.Ok(t, err)
	test.Equals(t, b1, b2)
	test.Equals(t, uint64(1000), w2)

	c1.Advance()

	ok, err = c1.Append(idn1.Mint(2, c1.Tip()))
	test.Ok(t, err)
	test.Equals(t, true, ok)

	_, w3, _ := c1.Read(c1.Tip())
	test.Equals(t, uint64(2000), w3) //sumweight should be stacked
}

func TestSyncConsensus(t *testing.T) {
	nIdns := 10
	nRounds := uint64(20)

	idns := make([]*topn.Identity, nIdns)
	for i := range idns {
		idns[i] = topn.NewIdentity([]byte{byte(i)})
	}

	chain := topn.NewChain(&topn.Block{})
	for j := uint64(0); j < nRounds; j++ {
		tip := chain.Tip()

		for _, idn := range idns {
			ok, err := chain.Append(idn.Mint(j+1, tip))
			test.Ok(t, err)
			test.Equals(t, true, ok)
		}

		chain.Advance()
	}

	//in this synchronous setting we expect the the tip to always be a straight
	//line through all the top ranking blocks.
	b, w, err := chain.Read(chain.Tip())
	test.Ok(t, err)
	test.Equals(t, uint64(nRounds*1000), w)
	test.Equals(t, uint64(nRounds), b.Round)
}

func nextTime(rnd *rand.Rand, r float64) float64 {
	return -math.Log(1.0-rnd.Float64()) / r
}

//returns a channel get's send the time 'n' times following a Poisson distribution
//with an average rate of 'λ' per 'period'
func poisson(seed int64, n int, λ float64, period time.Duration) (c chan time.Time) {
	rnd := rand.New(rand.NewSource(seed))
	c = make(chan time.Time)
	go func() {
		defer close(c)
		for i := 0; i < n; i++ {
			d := time.Duration(float64(period) * nextTime(rnd, λ))
			c <- <-time.After(d)
		}
	}()

	return
}

///////
// Open Test
//////

//@TODO test validation logic
func TestChainAppendValidation(t *testing.T) {

}

//@TODO implement actual logic and test
func TestChainThreshold(t *testing.T) {
	c1 := topn.NewChain(&topn.Block{})
	test.Equals(t, "0", c1.Threshold(c1.Tip()).Text(10))
}

//@TODO implement actual logic and test
func TestChainBalance(t *testing.T) {
	c1 := topn.NewChain(&topn.Block{})
	test.Equals(t, uint64(1), c1.Balance(c1.Tip(), topn.PK{}))
}
