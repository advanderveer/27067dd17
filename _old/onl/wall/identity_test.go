package wall

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestBasicIdentityOps(t *testing.T) {
	for i := 0; i < 10; i++ {
		id1 := NewIdentity([]byte{0x01}, rand.Reader)
		test.Equals(t, "4762ad64", id1.String())

		id1.SetName("alice")
		test.Equals(t, "alice", id1.String())
	}

	t.Run("panic on failure to read random", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should panic on failure to read random bytes")
			}
		}()

		NewIdentity(nil, bytes.NewBuffer(nil))
	})

}
