package wall

import (
	"testing"

	"github.com/advanderveer/go-test"
)

func TestBasicIdentityOps(t *testing.T) {
	for i := 0; i < 10; i++ {
		id1 := NewIdentity([]byte{0x01})
		test.Equals(t, "cecc1507", id1.String())

		id1.SetName("alice")
		test.Equals(t, "alice", id1.String())
	}
}
