package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestStakesFinalization(t *testing.T) {
	st1 := onl.NewStakes(1)
	pk1 := onl.PK{}

	st1.Votes[pk1] = 1

	test.Equals(t, 1.0, st1.Finalization())
}
