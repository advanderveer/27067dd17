package simulator

import (
	"bytes"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

func TestBlockEncode(t *testing.T) {
	prev := NilID
	prev[0] = 0x01

	b1 := NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x04}, prev)
	b1.nonce = 5

	test.Equals(t, 0, b1.Prio().Cmp(big.NewInt(1)))

	buf1 := bytes.NewBuffer(nil)
	err := b1.Encode(buf1)
	test.Ok(t, err)
	test.Equals(t, 178, buf1.Len())

	b2, err := DecodeBlock(buf1)
	test.Ok(t, err)

	test.Equals(t, b1, b2)
	test.Equals(t, uint64(5), b2.nonce)
	test.Equals(t, uint64(2), b2.widx)
}

func TestBlockEncodingLargePrio(t *testing.T) {
	prev := NilID
	prev[0] = 0x01

	prio := big.NewInt(math.MaxInt64)
	prio.Exp(prio, big.NewInt(4), nil)

	b1 := NewBlock(prio, 2, []byte{0x03}, []byte{0x04}, prev)
	b1.nonce = 5

	buf1 := bytes.NewBuffer(nil)
	err := b1.Encode(buf1)
	test.Ok(t, err)
	test.Equals(t, 209, buf1.Len())

	b2, err := DecodeBlock(buf1)
	test.Ok(t, err)

	test.Equals(t, b1, b2)
	test.Equals(t, prio, b2.Prio())
	test.Equals(t, 0, prio.Cmp(b2.Prio()))
}

func TestBlockHash(t *testing.T) {
	prev := NilID
	prev[0] = 0x01

	b1 := NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x04}, prev)
	b1.nonce = 5
	h1, err := b1.Hash()
	test.Ok(t, err)

	//same block data should result in same hash
	b2 := NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x04}, prev)
	b2.nonce = 5
	test.Equals(t, b1, b2)
	h2, err := b2.Hash()
	test.Ok(t, err)
	test.Equals(t, h1, h2)

	//change nonce
	b3 := NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x04}, prev)
	b3.nonce = 4
	h3, err := b3.Hash()
	test.Ok(t, err)
	test.Assert(t, h3 != h1, "nonce should change hash")

	//changing any construction value should change hash
	for _, b := range []*Block{
		NewBlock(big.NewInt(2), 2, []byte{0x03}, []byte{0x04}, prev),
		NewBlock(big.NewInt(1), 3, []byte{0x03}, []byte{0x04}, prev),
		NewBlock(big.NewInt(1), 2, []byte{0x04}, []byte{0x04}, prev),
		NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x05}, prev),
		NewBlock(big.NewInt(1), 2, []byte{0x03}, []byte{0x04}, NilID),
	} {
		b.nonce = 5

		h, err := b.Hash()
		test.Ok(t, err)
		test.Assert(t, h1 != h, "changing each block data should change the hash")
	}
}

func TestBlockDifficulty(t *testing.T) {
	prev := NilID
	prev[0] = 0x01

	_, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	seed := []byte{0x01}

	prio1, widx1, ticket1, proof1 := DrawPriority(sk, seed, 1)
	b1 := NewBlock(prio1, widx1, ticket1, proof1, prev)

	//@TODO the weight influence is exponential but difficulty in bits is also?

	max := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(256)), nil)
	test.Equals(t, uint(17), b1.Difficulty(max, 5, 25))
	test.Equals(t, uint(0), b1.Difficulty(max, 0, 0)) //fully PoS
	test.Equals(t, uint(0), b1.Difficulty(max, 0, 1))
	test.Equals(t, uint(24), b1.Difficulty(max, 24, 24)) //fully PoW
	test.Equals(t, uint(1), b1.Difficulty(max, 0, 2))

	prio2, widx2, ticket2, proof2 := DrawPriority(sk, seed, 10)
	b2 := NewBlock(prio2, widx2, ticket2, proof2, prev) //higher weight allows for lower difficulty
	test.Equals(t, uint(5), b2.Difficulty(max, 5, 25))
}

func TestBlockMining(t *testing.T) {
	_, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	max := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(256)), nil)
	nrounds := 100
	names := []string{"a", "b"}
	weights := map[string]uint64{"a": 7, "b": 100}
	avgs := map[string]time.Duration{}

	for _, name := range names {

		prev := NilID
		seed := []byte{0x01}
		prio := big.NewInt(0)
		var widx uint64
		var proof []byte
		var sumt time.Duration
		for i := 0; i < nrounds; i++ {
			prio, widx, seed, proof = DrawPriority(sk, seed, weights[name])

			b := NewBlock(prio, widx, seed, proof, prev)
			t0 := time.Now()
			prev, err = b.Mine(max, 1, 25) //these are all system constants
			test.Ok(t, err)

			sumt += time.Now().Sub(t0)
		}

		avgs[name] = sumt / time.Duration(nrounds)
	}

	//on average, mining time difference should be significant
	//@TODO this is fragile
	test.Assert(t, int(avgs["a"]/avgs["b"]) > 8, "should see major difference, got: %v", int(avgs["a"]/avgs["b"]))
}
