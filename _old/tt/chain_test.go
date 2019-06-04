package tt_test

import (
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

var _ tt.BlockReader = &tt.Chain{}

func TestChainAppending(t *testing.T) {
	c1 := tt.NewChain(7)
	b1 := tt.B(tt.NilID, []byte{0x01})

	c1.Append(b1)
	b2 := c1.Read(b1.Hash())
	test.Equals(t, b2, b1)

	d, err := c1.Difficulty(b1.Hash())
	test.Ok(t, err)

	test.Equals(t, 1, d.Cmp(big.NewInt(0)))

	_ = c1
}
