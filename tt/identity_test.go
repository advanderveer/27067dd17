package tt_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestVoteCreation(t *testing.T) {
	idn1 := tt.NewIdentity([]byte{0x01})
	id1 := tt.ID{}
	id1[0] = 0x02
	v1 := idn1.CreateVote(id1)

	test.Equals(t, "02000000", fmt.Sprintf("%.4x", v1.Tip))
	test.Equals(t, "1e3d9d3b", fmt.Sprintf("%.4x", v1.Token))
	test.Equals(t, "3a181d8d", fmt.Sprintf("%.4x", v1.Proof))
	test.Equals(t, "4762ad64", fmt.Sprintf("%.4x", v1.PK))
}

func TestPrinting(t *testing.T) {
	idn1 := tt.NewIdentity([]byte{0x01})
	test.Equals(t, "4762ad64", idn1.String())
	idn1.SetName("bob")
	test.Equals(t, "bob", idn1.String())
}
