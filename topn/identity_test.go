package topn_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/topn"
	"github.com/advanderveer/go-test"
)

func TestVoteCreation(t *testing.T) {
	idn1 := topn.NewIdentity([]byte{0x01})
	id1 := topn.ID{}
	id1[0] = 0x02
	b1 := idn1.Mint(1, id1)

	test.Equals(t, "02000000", fmt.Sprintf("%.4x", b1.Prev.Bytes()))
	test.Equals(t, "1e3d9d3b", fmt.Sprintf("%.4x", b1.Token))
	test.Equals(t, "3a181d8d", fmt.Sprintf("%.4x", b1.Proof))
	test.Equals(t, "4762ad64", fmt.Sprintf("%.4x", b1.PK))
}

func TestPrinting(t *testing.T) {
	idn1 := topn.NewIdentity([]byte{0x01})
	test.Equals(t, "4762ad64", idn1.String())
	idn1.SetName("bob")
	test.Equals(t, "bob", idn1.String())
}
