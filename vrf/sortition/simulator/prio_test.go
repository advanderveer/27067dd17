package simulator

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

func TestBasicPriorityDrawAndVerification(t *testing.T) {
	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	seed := []byte{0x01}

	prio, widx, ticket, proof := DrawPriority(sk, seed, 0)
	test.Assert(t, prio.Int64() == 0, "prio should be nil")
	test.Equals(t, uint64(0), widx)
	test.Equals(t, 96, len(proof))
	test.Equals(t, 32, len(ticket))

	prio, widx, ticket, proof = DrawPriority(sk, seed, 1)
	test.Equals(t, "42207894229259356219642474069058778023504130928133797121372040495940115196880", prio.String())
	test.Equals(t, uint64(0), widx)
	test.Equals(t, 96, len(proof))
	test.Equals(t, 32, len(ticket))

	prio, widx, ticket, proof = DrawPriority(sk, seed, 2) //shoud prefer the old one, new draw is lower
	test.Equals(t, "42207894229259356219642474069058778023504130928133797121372040495940115196880", prio.String())
	test.Equals(t, uint64(0), widx)
	test.Equals(t, 96, len(proof))
	test.Equals(t, 32, len(ticket))

	prio, widx, ticket, proof = DrawPriority(sk, seed, 3) //shoud prefer the third one, it is higher
	test.Equals(t, "82586484205483746716696419634216281917129789806732735421185591415625649998439", prio.String())
	test.Equals(t, uint64(2), widx)
	test.Equals(t, 96, len(proof))
	test.Equals(t, 32, len(ticket))

	ok := VerifyPriority(pk, seed, ticket, proof, widx, prio)
	test.Equals(t, true, ok)

	ok = VerifyPriority(pk, seed, append(ticket, 0x01), proof, widx, prio)
	test.Equals(t, false, ok)

	ok = VerifyPriority(pk, seed, ticket, append(proof, 0x01), widx, prio)
	test.Equals(t, false, ok)

	ok = VerifyPriority(pk, append(seed, 0x01), ticket, proof, widx, prio)
	test.Equals(t, false, ok)

	ok = VerifyPriority(append(pk, 0x01), seed, ticket, proof, widx, prio)
	test.Equals(t, false, ok)

	ok = VerifyPriority(pk, seed, ticket, proof, 1, prio)
	test.Equals(t, false, ok)

	ok = VerifyPriority(pk, seed, ticket, proof, widx, prio.Add(prio, big.NewInt(1)))
	test.Equals(t, false, ok)
}

// type Charter map[]

func TestScenarios(t *testing.T) {
	_, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	//some none-random seed
	seed := make([]byte, 32)

	//max prio for a 32 byte hash
	order := []string{"a", "b"}
	weights := map[string]uint64{"a": 10, "b": 1}
	nrounds := 100
	results := map[string]uint64{"a": 0, "b": 0}

	//try n rounds and see which of the weights get selected more often
	for i := 0; i < nrounds; i++ {

		highestname := ""
		highest := big.NewInt(0)
		curr := big.NewInt(0)

		//order is important because it changes the seed the next draws will be
		//based on
		for _, name := range order {
			curr, _, seed, _ = DrawPriority(sk, seed, weights[name])

			if curr.Cmp(highest) > 0 {
				highest.SetBytes(curr.Bytes())
				highestname = name
			}
		}

		results[highestname]++
	}

	//on average, with a weight 10 times higher we expect it to be selected as
	//leader a lot more: @TODO how much?
	test.Equals(t, uint64(85), results["a"])
	test.Equals(t, uint64(15), results["b"])

	// the threshold function should prevent the network from ddos/saturating itself
	// - if the network is small enough it should allow anyone to forge a block
	// - if the network is large it should flat out at a certain max.
	// - if the threshold function causes no-leader to be selected

	// total := new(big.Rat)
	// // for _, w := range []uint64{1, 100} {
	// //
	// var prio *big.Int
	// n := int64(1000)
	// for i := int64(0); i < n; i++ {
	// 	prio, _, seed, _ = DrawPriority(sk, seed, 5)
	// 	norm := new(big.Rat).SetFrac(prio, max)
	// 	total.Add(total, norm)
	// }

	// the weight influences the how far in the top precentile
	// a member can get quiete heavily:
	// 0:    ~0.0
	// 1:    ~0.5
	//10:    ~0.9    (1-1/10)
	//100:   ~0.99   (1-1/100)
	//1000:  ~0.999  (1-1/1000)
	//10000: ~0.9999 (1-1/10000)

	// so increasing their weight (level) should equally hard as well. e.g diminishing
	// returns with an asymptoot that is ifinitely hard to reach. And increasing it must
	// be inversely proportional to the largest difference in the economy?

	//at a high level, given a network of two members. If one has weight 10, and the
	//other has weight 1. One should become leader 10 times more often. Ore 1 is
	//10 times more likely to be. So threshold is 1/11

	// total.Quo(total, big.NewRat(n, 1))
	// fmt.Println(total.FloatString(10)) //on average: 0.5, makes sense

	//given the total weight, what is the threshold function such that, on average
	//a leader is picked N times. For small charcters the threshold function returns
	//something super easy (everyone will draw). For other heights the threshold becomes
	//more difficult. But we need always, only one

	// can we measure the threshold based on the block history, the required difficulty
	// - as a function of how many different agents submitted? if it only the same
	// we lower the difficulty. If too many new agents make the difficulty higher?

	// if a has a certain nr of weight

	// given a charter with N members, and the passing of X blocks. With a given
	// seed in the beginning should result in this amount of blocks being proposed
	// to the network. Determine what method results in a sensible threshold for
	// for networks of various sizes works

	// given a charter with 1 pk: always draws itself as leader
	// given a charcter with 2,3 pk: always draw everyone
	// given a charter with many 100.000+ drawing should be performant

	// given a certain weight distribution, how often do we expect someone to be
	// drawn

}
