package sortition

import (
	"math/big"

	"github.com/ALTree/bigfloat"
)

// Given a pseudo-random hash 'h', threshold 'tau', own weight 'w' and the total
// weight 'W' of all members combined Return the number of sub-users 'j' the hash
// gives acces to
// ref3: https://github.com/tinychain/algorand/blob/master/algorand.go#L586
func subs(h []byte, τ, w, W int64) (j int64) {
	p := float64(τ) / float64(W)

	hashBig := new(big.Int).SetBytes(h)
	maxHash := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(len(h))*8), nil)
	hash := new(big.Float).SetRat(new(big.Rat).SetFrac(hashBig, maxHash))

	var lower, upper *big.Float
	for j <= w {
		if upper != nil {
			lower = upper
		} else {
			lower = binomSum(j, w, p)
		}

		upper = binomSum(j+1, w, p)

		if hash.Cmp(lower) >= 0 && hash.Cmp(upper) < 0 {
			break
		}

		j++
	}

	return
}

func exp(p *big.Rat, k int64) (r *big.Rat) {
	if k <= 0 {
		return big.NewRat(1, 1)
	}

	//@TODO cover other exp edg cases?

	r = new(big.Rat).Set(p)
	for i := int64(1); i < k; i++ {
		r.Mul(r, p)
	}

	return
}

func binom3(k, n int64, p *big.Rat) (r *big.Rat) {
	// binomial coefficient and using it for binomial distribution
	// https://www.quora.com/Are-binomial-coefficients-and-binomial-distributions-the-same
	bc := big.NewInt(0).Binomial(n, k)
	bcr := new(big.Rat).SetInt(bc) // (𝑛/𝑘)

	//@TODO exp rats is very slow
	pkr := exp(p, k)                                        // (𝑝)^𝑘
	qpkr := exp(new(big.Rat).Sub(big.NewRat(1, 1), p), n-k) // (1−𝑝)^𝑛−𝑘

	r = new(big.Rat)
	r.Mul(bcr, pkr)
	r.Mul(r, qpkr)
	return
}

//chance of 'k' successes in 'n' trials, each having probability 'p' of success
func binom(k, n int64, p float64) *big.Float {

	////binomial coefficient: https://www.quora.com/Are-binomial-coefficients-and-binomial-distributions-the-same
	binoc := big.NewInt(0)
	binoc.Binomial(n, k)

	binocf := big.NewFloat(0)
	binocf.SetInt(binoc)
	kf := big.NewFloat(float64(k))
	nkf := big.NewFloat(float64(n - k))

	//@TODO bigfloat exp/pow is hard: https://github.com/golang/go/issues/14102
	pxf := bigfloat.Pow(big.NewFloat(p), kf)
	qnxf := bigfloat.Pow(big.NewFloat(1.0-p), nkf)

	result := new(big.Float).Mul(binocf, pxf)
	result = result.Mul(result, qnxf)
	return result
}

// sum of all changes of 0..j successes in 'n' trials: B(0, w, p) + ... + B(j, w, p)
func binomSum3(j, n int64, p *big.Rat) (sum *big.Rat) {
	sum = new(big.Rat)
	for k := int64(0); k <= j; k++ {
		sum.Add(sum, binom3(k, n, p))
	}

	return sum
}

// sum of all changes of 0..j successes in 'n' trials: B(0, w, p) + ... + B(j, w, p)
func binomSum(j, n int64, p float64) (sum *big.Float) {
	sum = big.NewFloat(0)
	for k := int64(0); k <= j; k++ {
		sum.Add(sum, binom(k, n, p))
	}

	return sum
}
