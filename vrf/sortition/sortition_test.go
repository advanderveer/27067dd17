package sortition

import (
	"math"
	"math/big"
	"testing"

	test "github.com/advanderveer/go-test"
)

// func TestFactorial(t *testing.T) {
// 	test.Equals(t, int64(1), fact(0).Int64())
// 	test.Equals(t, int64(1), fact(1).Int64())
// 	test.Equals(t, int64(2432902008176640000), fact(20).Int64())
// 	test.Equals(t, int64(-4249290049419214848), fact(21).Int64()) //overflowed
// }
//
// func TestDiv(t *testing.T) {
// 	test.Equals(t, int64(1)/int64(1), div1(fact(1), fact(1)).Int64())
// 	test.Equals(t, int64(1)/int64(2), div1(fact(1), fact(2)).Int64())
//
// 	test.Equals(t, int64(20)/int64(3)/int64(2), div1(div1(big.NewInt(20), big.NewInt(3)), big.NewInt(2)).Int64())
// }

func TestBinomial2(t *testing.T) {
	p := 0.3 //from: https://en.wikipedia.org/wiki/Binomial_distribution#Example
	test.Equals(t, "0.117649", binom(0, 6, p).String())
	test.Equals(t, "0.302526", binom(1, 6, p).String())
	test.Equals(t, "0.324135", binom(2, 6, p).String())
	// //...
	test.Equals(t, "0.000729", binom(6, 6, p).String())
	test.Equals(t, "1.302622713e-05", binom(50, 100, p).String())

	test.Equals(t, "1", binomSum(6, 6, p).String())
	test.Equals(t, "0.420175", binomSum(1, 6, p).String())
	test.Equals(t, "0.74431", binomSum(2, 6, p).String())

	// in our sortition case, we would, get high j's, this can become rather slow
	test.Equals(t, "1.197250832e-63", binomSum(100, 1100, p).String())

	//with some serious large numbers this would take seconds to calculate
	test.Equals(t, "0", binom(10000, math.MaxInt64, p).String())
}

// func TestBinomial(t *testing.T) {
// 	p := 0.3 //from: https://en.wikipedia.org/wiki/Binomial_distribution#Example
// 	test.Equals(t, 0.11764899999999995, binom1(0, 6, p))
// 	test.Equals(t, 0.30252599999999985, binom1(1, 6, p))
// 	test.Equals(t, 0.32413499999999984, binom1(2, 6, p))
// 	//...
// 	test.Equals(t, 0.0007289999999999999, binom1(6, 6, p))
//
// 	// test.Equals(t, 0.30252599999999985, binom1(50, 100, p)) //@TODO too big
//
// 	//should be able to do big number binoms without panic
// 	// fmt.Println(binom1(math.MaxInt64, math.MaxInt64, 0.3))
// }

func TestBinomSum(t *testing.T) {
	// p := 0.1

	//@TODO can return NaN on heigh ns
	// test.Equals(t, 1.0, binomSum(999999, p, 10))
}

// func TestDivision(t *testing.T) {
// 	test.Equals(t, "0.5", div1(fact(1), fact(2)).String())
// 	test.Equals(t, "2", div1(fact(2), fact(1)).String())
//
//   test.Equals()
//
// 	comb :=
// 	fmt.(comb)
//
// }

// func TestBinomial(t *testing.T) {
// 	test.Equals(t, int64(2), fact(2))
// 	test.Equals(t, int64(1), fact(1))
// 	test.Equals(t, int64(1), fact(0))
//
// 	p := 0.1                            //probability of the front face (change of a certain role in algorand)
// 	test.Equals(t, 0.9, binom(0, 1, p)) //get front 0 out of 1
// 	test.Equals(t, 0.1, binom(1, 1, p)) //get front 1 out of 1
//
// 	//the sum of the change of rolling front face, 0 times, 1 time, 2 times etc
// 	//is always 1: you role something once
// 	test.Equals(t, 1.0, binomSum(1, p, 1))
// 	test.Equals(t, 1.0, binomSum(65, p, 1))
// 	test.Equals(t, 1.0, binomSum(66, p, 1))
// 	// test.Equals(t, 1.0, binomSum(999999, p, 1))
//
// 	p = 0.3
//
// 	test.Equals(t, 0.7, binom(0, 1, p)) //get front 0 out of 1
// 	test.Equals(t, 0.3, binom(1, 1, p)) //get front 1 out of 1
// }

func TestBinom3(t *testing.T) {
	p := big.NewRat(3, 10) //0.3

	test.Equals(t, p, exp(p, 1))
	test.Equals(t, "0.0900000000", exp(p, 2).FloatString(10))
	test.Equals(t, "1.0000000000", exp(p, 0).FloatString(10))

	test.Equals(t, "0.117649", binom3(0, 6, p).FloatString(6))
	test.Equals(t, "0.302526", binom3(1, 6, p).FloatString(6))
	test.Equals(t, "0.324135", binom3(2, 6, p).FloatString(6))
	// //...
	test.Equals(t, "0.000729", binom3(6, 6, p).FloatString(6))
	// test.Equals(t, "0.00001302622713144536", binom3(50, 100, p).FloatString(20))

	//with some serious large numbers this would take seconds to calculate
	//@TODO large binoms are slow
	// test.Equals(t, "0.0000000000", binom3(500, 1000, p).FloatString(10))
	//
	// test.Equals(t, "1/1", binomSum3(6, 6, p).String())
	// test.Equals(t, "0.420175", binomSum3(1, 6, p).FloatString(6))
	// test.Equals(t, "0.744310", binomSum3(2, 6, p).FloatString(6))
}

func TestSubs(t *testing.T) {

	//@TODO binominal approach seems flawed:
	// -- scales poorly with weight in the system, computational expensive to compute
	// -- unclear (to me) how the distribution works in practice

	// h1 := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	// w := int64(200)       // @TODO this scales poorly this: miners weight in the pool
	// τ := int64(1000000)   // threshold, the expected number 1 of users
	// W := int64(100000000) // total weight
	//
	// fmt.Println(subs(h1, τ, w, W))

}
