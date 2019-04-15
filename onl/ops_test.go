package onl_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

var _ onl.Op = &onl.JoinOp{}

// var _ onl.Op = &onl.LeaveOp{}
// var _ onl.Op = &onl.TransferOp{}

func TestJoinOpHashing(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})

	op1 := idn1.Join(1)
	test.Equals(t, uint64(1), op1.Deposit)
	test.Equals(t, idn1.TokenPK(), []byte(op1.TokenPK[:]))
	test.Equals(t, idn1.PK(), []byte(op1.Identity[:]))
	test.Equals(t, "6d1c3097", fmt.Sprintf("%.4x", op1.Signature))
	test.Equals(t, "d6c5f9ef", fmt.Sprintf("%.4x", op1.Hash()))

	op1.Deposit = 2
	test.Equals(t, "831b50ac", fmt.Sprintf("%.4x", op1.Hash()))

	op1.TokenPK[0] = 0x01
	test.Equals(t, "015f3973", fmt.Sprintf("%.4x", op1.Hash()))

	op1.Identity[0] = 0x01
	test.Equals(t, "ae33a143", fmt.Sprintf("%.4x", op1.Hash()))
}
