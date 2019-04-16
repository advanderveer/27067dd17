package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestChainCreationAndGenesis(t *testing.T) {
	s1, clean := onl.TempBadgerStore()
	defer clean()

	c1 := onl.NewChain(s1)
	g1 := c1.Genesis()

	c2 := onl.NewChain(s1)
	g2 := c2.Genesis()

	test.Equals(t, uint64(0), g2.Round)
	test.Equals(t, []byte("vi veri veniversum vivus vici"), g2.Token)

	test.Equals(t, g2, g1)
}
