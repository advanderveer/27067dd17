package thr

import (
	"math/big"

	"github.com/cockroachdb/apd"
)

//Thr returns whether the token t is a roll good enough such that it is lower then
//the threshold defined by f, a and T
func Thr(c *apd.Context, f *apd.Decimal, a, T uint64, t []byte) (φ, y *apd.Decimal, ok bool) {
	φ = Phi(c, f, a, T)
	y = Dec(c, t)
	if y.Cmp(φ) < 0 {
		return φ, y, true
	}

	return
}

// Dec returns a decimal representation of a byte slice such that when its filled
// with zeros it corresponds with 0.0 and 1.0 when it is filled with 1s
func Dec(c *apd.Context, t []byte) (d *apd.Decimal) {
	max := make([]byte, len(t))
	for i := range max {
		max[i] = 0xff
	}

	a := apd.NewWithBigInt(new(big.Int).SetBytes(t), 0)
	b := apd.NewWithBigInt(new(big.Int).SetBytes(max), 0)

	d = new(apd.Decimal)
	_, err := c.Quo(d, a, b)
	if err != nil {
		panic("thr/dec: " + err.Error())
	}

	return
}
