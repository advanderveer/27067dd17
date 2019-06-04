package tlog

import (
	"fmt"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestProof2(t *testing.T) {
	test.Equals(t, "h(L=10, K=2)", fmt.Sprint(h{L: 10, K: 2}))

	Decompose(9, 13)

}
