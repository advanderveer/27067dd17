package tt_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestBlockHashing(t *testing.T) {
	b1 := tt.B(tt.NilID, []byte{0x01})
	test.Equals(t, "1fd4247443c9", fmt.Sprintf("%.6x", b1.Hash()))

	b1.Data[0] = 0x02
	test.Equals(t, "58cc2f44d3a2", fmt.Sprintf("%.6x", b1.Hash()))

	b1.Prev[0] = 0x02
	test.Equals(t, "a3b7219472e4", fmt.Sprintf("%.6x", b1.Hash()))
}
