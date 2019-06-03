package tlog

import (
	"fmt"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestAppend(t *testing.T) {
	log1 := NewLog()
	test.Equals(t, uint64(1), log1.Append([]byte("hello, log")))
	test.Equals(t, uint64(2), log1.Append([]byte("log")))

	test.Equals(t, []byte("hello, log"), log1.Read(1))
	test.Equals(t, []byte("log"), log1.Read(2))
}

func TestProving(t *testing.T) {

	log1 := NewLog()
	var size uint64
	for i := 0; i <= 12; i++ {
		r := []byte(fmt.Sprintf("r%d", i))
		size = log1.Append(r)
		test.Equals(t, uint64(i+1), size)
	}

	test.Equals(t, uint64(13), size)

	// rh := sha256.Sum256([]byte("r9"))
	// fmt.Println(log1.Contains(9, size))
}
