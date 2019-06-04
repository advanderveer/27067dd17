package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestPrinting(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	test.Equals(t, "cecc1507", idn1.String())
	idn1.SetName("bob")
	test.Equals(t, "bob", idn1.String())
}
