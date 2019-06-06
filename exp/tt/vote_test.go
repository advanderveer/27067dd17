package tt_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestVoteHashing(t *testing.T) {
	idn1 := tt.NewIdentity([]byte{0x01})
	v1 := idn1.CreateVote(tt.NilID)
	test.Equals(t, "4d9c4a01ea55", fmt.Sprintf("%.6x", v1.Hash()))

	v1.Tip[0] = 0x04
	test.Equals(t, "15484af7d605", fmt.Sprintf("%.6x", v1.Hash()))

	v1.Token[0] = 0x04
	test.Equals(t, "d7dbc2610168", fmt.Sprintf("%.6x", v1.Hash()))

	v1.Proof[0] = 0x04
	test.Equals(t, "98d50161ef5c", fmt.Sprintf("%.6x", v1.Hash()))

	v1.PK[0] = 0x04
	test.Equals(t, "0955588e58d8", fmt.Sprintf("%.6x", v1.Hash()))
}
