package ssi

import (
	"testing"

	test "github.com/advanderveer/go-test"
)

func TestKeySets(t *testing.T) {
	s1 := make(KeySet)
	s1.Add([]byte("alex"))
	s1.Add([]byte("alex"))
	s1.Add([]byte("bob"))

	test.Equals(t, 2, len(s1))

	s2 := make(KeyChangeSet)
	s2.Add([]byte("alex"), []byte{0x02})
	s2.Add([]byte("alex"), []byte{0x03})
	s2.Add([]byte("bob"), []byte{0x02})

	test.Equals(t, 2, len(s2))

}
