package sortition

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

type Block struct {
	difficulty uint
	data       []byte
	nonce      int64 //@TODO not really a nonce, why an int?
}

//Hash returns the sha256 of this block
func (b *Block) Hash() (h []byte) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, b.nonce)

	sha := sha256.Sum256(bytes.Join([][]byte{
		b.data,
		buf[:n],
	}, nil))

	return sha[:]
}

//Mine the block, it will set the nonce to a value such
//that the hash of the block is a number that is lower then
//the max (256bit) number minus the difficulty
func (b *Block) Mine() (proof []byte) {
	proofi := big.NewInt(0)
	target := new(big.Int).Lsh(big.NewInt(1), uint(256-b.difficulty))

	// target2 := new(big.Int).Exp(x, y, m)

	fmt.Printf("%x\n", target.Bytes())
	for b.nonce = int64(0); b.nonce < math.MaxInt64; b.nonce++ {
		proof = b.Hash()
		if proofi.SetBytes(proof).Cmp(target) == -1 {
			break
		}
	}

	return
}

func HashDistribution(t *testing.T) {

	// https://burtleburtle.net/bob/hash/birthday.html: "[Chuck Blake pointed out
	// that hash functions produce a binomial distribution in each hash bucket,
	// which in the limit is a Poisson distribution. That is, if there are n buckets
	// and m items hashed into them, and a=m/n, then the probability of a given
	// bucket having k items in it will be about (ake-a)/k! , so the total number
	// of buckets with k items will be about (nake-a)/k!"

	// https://michiel.buddingh.eu/distribution-of-hash-values: A hash function
	// should, insofar possible, generate for any set of inputs, a set of outputs
	// that is uniformly distributed over its output space. If we count the
	// frequency of the number of '1' bits in its outputs, we should get a nice,
	// clear binomial distribution

	// lets imagine we want to create a selection of about 't' members out of a
	// large group of 'W' sub-users.

	// each user 'pk' has a number of tokens 'w', the total number of tokens
	// across all users is 'W'.

	// We normalize the hash on the total number of
	// tokens and use k-ary tree to select the leader

	// the hash determines how often we are allowed to draw from the pool of tokens
	// return the probability that we draw ourselves

	// we want to draw tokens uniformly from the pool of all tokens such that we end
	// up with a selection that proves that we have enough tokens to be leader

	// the Hash determines how often we can draw from the pool

	// lets imagine the desired number of sub-users 't' in the group of all
	// sub-users 'W'. Each sub-user would throw a biased coin with the
	// probability of t/W for success.

	// Using a k-ary tree for efficient sortition?
	// https://medium.com/kleros/an-efficient-data-structure-for-blockchain-sortition-15d202af3247

	// Random ballot, binominal CDF
	// https://en.wikipedia.org/wiki/Random_ballot#Emergent_properties
	// calculat the chance of being in the majority?

	// our hash is uniformily distributed
	// we can walk over it and use every bit/byte to
	// draw from the population

	// sortition also has a 'min' number of members, if the number of miners
	// in the charter is less then this number. Everyone in the charter is
	// eligible for creating a block. Beyond this number the chance that no-one
	// is selected should be very small, and even if then no leader is selected
	// should result in worst case of having to mine the traditional way

	//time slot based Sortition?
	//https://arxiv.org/pdf/1803.08694.pdf

	//just using a security quotient the hash needs to fall below?
	//https://royalsocietypublishing.org/doi/pdf/10.1098/rsos.180422

}

func TestVRFDifficulty(t *testing.T) {

	// pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	pk, sk, err := vrf.GenerateKey(rand.Reader)
	test.Ok(t, err)

	seed := []byte{0x01} //from previous block
	ticket, _ := vrf.Prove(seed, sk)

	//@TODO what if the number of vrf tickets is proportional to the number of
	//weight that the user has? It always takes the lowest value it has?
	var lowest *big.Rat
	for i := uint64(0); i < 100000; i++ {
		// seed := make([]byte, 32)
		// rand.Read(seed)

		//if a block comes along that already passed the threshold we cancel our
		//search. If we do not cancel we run the risk of our block not being accepted
		//becaue everyone is mining on the next block. Our block may outpace that mine
		//or it might not. But chances are slim

		//all users only accept one block per pk, per height, others are discarded by
		//the gossip layer.
		//if a block with a higher score is already seen, discard any other block
		//passing through

		//in practice, the token hashing can be done in parralel, highly effciently
		//so is not really a computational bottleneck that slows down the network for
		//high stake users.

		//do we cancel if the incoming lowest is lower then our own lowest? yes

		//if we also include a max stake. as to limit the computational intensity of
		//the ticket hashing

		//what if it must show that it has tested all its stake? how can others check
		//this efficiently?

		ticketNonce := make([]byte, 8)
		binary.BigEndian.PutUint64(ticketNonce, i)

		ticketh := sha256.Sum256(bytes.Join([][]byte{ticket, ticketNonce}, nil))

		ticB := new(big.Int).SetBytes(ticketh[:])
		maxB := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(len(ticket))*8), nil)
		// minB := new(big.Int).SetBytes(make([]byte, len(ticket)))

		maxB.Sub(maxB, big.NewInt(1)) //@TODO is this valid

		norB := new(big.Rat).SetFrac(ticB, maxB) //normalize

		// fmt.Printf("min: %x\n", minB)
		// fmt.Printf("max: %x\n", maxB)
		// fmt.Printf("tic: %x\n", ticB)

		//@TODO only needs to run until it is below the threshold, if the attacker has
		//specialized hardware it can go lower. If it wants his/her block to have the
		//highest change of being prioritized by the network.

		//as such the must be a max to the stake the someone can use for his block
		//priority

		if lowest == nil || norB.Cmp(lowest) < 0 {
			lowest = norB
		}

		// if norB.Cmp(big.NewRat(1, 100)) < 0 {
		// 	fmt.Printf("nor: %f %s\n", float64(1)/float64(100), norB.FloatString(10))
		// }
	}

	fmt.Println("lowest", lowest.FloatString(10))

	// w := int64(200)   // @TODO this scales poorly this: miners weight in the pool
	// τ := int64(10)    // threshold, the expected number 1 of users
	// W := int64(10000) // total weight
	//
	// t0 := time.Now()
	// fmt.Println("subs:", subs(ticket, τ, w, W))
	// fmt.Println(time.Now().Sub(t0))

	// 1) we normalize the hash to: 0... 1

	// 2) the max value in the space is W: sum of all weights of all participants

	// 3) each actors weight times the max hash, is the max hash space. This is
	// the max outcome of the whole lottery. In that case everyone should get
	// the lowest possible difficulty.

	// 4) given my ticket, times my weight, in what percentile do i fall, if everyone
	// drew an average ticket

	_ = pk
	_ = sk

	//

	b := &Block{
		difficulty: 1,
		data:       []byte{0x01},
	}

	proof := b.Mine()
	fmt.Printf("%x (nonce: %d)", proof, b.nonce)

	// You might also note that shifting left is equivalent to multiplication by powers
	// of 2. So 6 << 1 is equivalent to 6 * 2, and 6 << 3 is equivalent to 6 * 8. A good
	// optimizing compiler will replace multiplications with shifts when possible.

	//target
	// targetBits := uint(255) //max target:
	// target := big.NewInt(1) //1 bytes: 00
	// fmt.Printf("%08b\n", target.Bytes())
	// target.Lsh(target, 256-targetBits)
	// fmt.Printf("%08b (%d)\n", target.Bytes(), len(target.Bytes()))
	//
	// fmt.Println(target.Cmp(big.NewInt(0))) // target >  0
	// fmt.Println(target.Cmp(big.NewInt(1))) // target equals  1
	// fmt.Println(target.Cmp(big.NewInt(2))) // target < 2

	// lowestb := make([]byte, 32)
	// lowesti := big.NewInt(0).SetBytes(make([]byte, 32))
	// fmt.Printf("%08b (%d)\n", lowesti.Bytes(), len(lowesti.Bytes()))

	// targetBits := 254
	// target := big.NewInt(1)
	// fmt.Printf("%x\n", target)
	// target.Lsh(target, uint(256-targetBits))
	// fmt.Printf("targe: %x\n", target)
	// fmt.Printf("targe: %x\n", target.Bytes())
	//
	// b := &Block{data: []byte{0x01}}
	//
	// proof := mine(b, target.Bytes())
	//
	// fmt.Printf("proof: %x\n", proof)

	// 04ba37620962c58763ff085fef0d88f6b47e6f86ba51eb74d0099c1f4d72510a
}
