package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
)

var _ onl.Store = &onl.BadgerStore{}

func TestBadgerStore(t *testing.T) {
	s, clean := onl.TempBadgerStore()
	defer clean()

	_ = s
}
