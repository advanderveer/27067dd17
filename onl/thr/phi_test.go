package thr

import (
	"math"
	"testing"

	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
)

func TestBase2(t *testing.T) {
	c := apd.BaseContext.WithPrecision(50)
	test.Equals(t, "1", base(c, apd.New(0, 0)).Text('f'))
	test.Equals(t, "0", base(c, apd.New(1, 0)).Text('f'))
}

func TestAlpha2(t *testing.T) {
	c := apd.BaseContext.WithPrecision(50)
	test.Equals(t, "0.5", alpha(c, 5, 10).Text('f'))
	test.Equals(t, "1", alpha(c, 10, 10).Text('f'))

	t.Run("panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should panic if identity's stake larger then total stake")
			}
		}()

		test.Equals(t, "1", alpha(c, 10, 2).Text('f'))
	})
}

func assertIndepAggr(t *testing.T, c *apd.Context, comb, sep *apd.Decimal) {

	// the product of (1 - Phi(1/3)) * (1 - Phi(1/3))
	phiOneSub := new(apd.Decimal)
	c.Sub(phiOneSub, apd.New(1, 0), sep)
	phiProd := new(apd.Decimal)
	c.Mul(phiProd, phiOneSub, phiOneSub)

	// must equal 1- Phi(2/3)
	phiOneSubComb := new(apd.Decimal)
	c.Sub(phiOneSubComb, apd.New(1, 0), comb)

	//once rounded to 49 decimal places they will be equal
	phiProdR := new(apd.Decimal)
	c.Quantize(phiProdR, phiProd, -49)
	phiOneSubCombR := new(apd.Decimal)
	c.Quantize(phiOneSubCombR, phiOneSubComb, -49)

	//finally, assert that the "independent aggregation" holds up
	test.Equals(t, 0, phiProdR.CmpTotal(phiOneSubCombR))
}

func TestPhi2(t *testing.T) {
	c := apd.BaseContext.WithPrecision(50)
	f, _ := new(apd.Decimal).SetFloat64(0.5)

	//1 stake out of max uint64 very small
	test.Equals(t, "0.00000000000000000003757558395076474551398066717012", Phi(c, f, 1, math.MaxUint64).Text('f'))

	// having all stake means just a chance equal to f
	test.Equals(t, "0.5", Phi(c, f, math.MaxUint64, math.MaxUint64).Text('f'))

	//two phi functions we want to compare, we expect the two draw seperate stake
	//to be equal in value to the combined stake draw.
	assertIndepAggr(t, c, Phi(c, f, 2, 2), Phi(c, f, 1, 2))
	assertIndepAggr(t, c, Phi(c, f, 2, 100), Phi(c, f, 1, 100))
	assertIndepAggr(t, c, Phi(c, f, 2, math.MaxUint64), Phi(c, f, 1, math.MaxUint64))
}
