package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

func TestPrinting(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	test.Equals(t, "cecc1507", idn1.String())
	idn1.SetName("bob")
	test.Equals(t, "bob", idn1.String())
}
