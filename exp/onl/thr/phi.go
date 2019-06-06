package thr

import (
	"math/big"

	"github.com/cockroachdb/apd"
)

// base returns the base of the threshold exponentation and is parameterized by
// f. f ∈ (0, 1] is called the active slots coefficient [...] f is the
// probability that a hypothetical party controlling all 100% of the stake would
// be elected leader.
func base(c *apd.Context, f *apd.Decimal) (b *apd.Decimal) {
	b = new(apd.Decimal)
	_, err := c.Sub(b, apd.New(1, 0), f)
	if err != nil {
		panic("thr/base: " + err.Error())
	}

	return
}

// alpha returns 'α', the fraction of stake that is associated with this party out of all
// stake. it takes 'a' as the amount of stake of an identity and 'T' as the total
// amount of stake
func alpha(c *apd.Context, a, T uint64) (α *apd.Decimal) {
	if a > T {
		panic("the total stake should be larger then the identity's stake")
	}

	α = new(apd.Decimal)
	ad := apd.NewWithBigInt(new(big.Int).SetUint64(a), 0)
	Td := apd.NewWithBigInt(new(big.Int).SetUint64(T), 0)

	_, err := c.Quo(α, ad, Td)
	if err != nil {
		panic("thr/alpha: " + err.Error())
	}

	return
}

// Phi returns φ(α) returns wheter a party with relative stake α ∈ (0, 1] becomes
// a slot leader in a particular slot with probability φf (α), independently of
// all other parties.
func Phi(c *apd.Context, f *apd.Decimal, a, T uint64) (φ *apd.Decimal) {
	d := new(apd.Decimal)
	_, err := c.Pow(d, base(c, f), alpha(c, a, T))
	if err != nil {
		panic("thr/phi: " + err.Error())
	}

	φ = new(apd.Decimal)
	_, err = c.Sub(φ, apd.New(1, 0), d)
	if err != nil {
		panic("thr/phi: " + err.Error())
	}

	return
}
