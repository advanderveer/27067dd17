package thr

import (
	"testing"

	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
)

var max = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
var min = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func TestDecRepresentation(t *testing.T) {
	c := apd.BaseContext.WithPrecision(50) //casual 50 point decimal

	test.Equals(t, 0, Dec(c, []byte{0xff}).Cmp(apd.New(1, 0)))
	test.Equals(t, 0, Dec(c, []byte{0xff}).CmpTotal(apd.New(1, 0)))
	test.Equals(t, 0, Dec(c, max).CmpTotal(apd.New(1, 0)))

	test.Equals(t, 0, Dec(c, []byte{0x00}).Cmp(apd.New(0, 0)))
	test.Equals(t, 0, Dec(c, []byte{0x00}).CmpTotal(apd.New(0, 0)))
	test.Equals(t, 0, Dec(c, min).CmpTotal(apd.New(0, 0)))

	test.Equals(t, "0.50196078431372549019607843137254901960784313725490", Dec(c, []byte{0x80}).Text('f'))
	test.Equals(t, "0.49803921568627450980392156862745098039215686274510", Dec(c, []byte{0x7f}).Text('f'))
}

func TestThresholdValidation(t *testing.T) {
	c := apd.BaseContext.WithPrecision(50) //casual 50 point decimal
	f := apd.New(5, -1)                    //0.5

	_, _, ok1 := Thr(c, f, 2, 2, []byte{0x00})
	test.Equals(t, true, ok1)
	_, _, ok2 := Thr(c, f, 2, 2, []byte{0xff})
	test.Equals(t, false, ok2)
	_, _, ok3 := Thr(c, f, 2, 2, []byte{0x80}) //just too high, see test above
	test.Equals(t, false, ok3)
	_, _, ok4 := Thr(c, f, 2, 2, []byte{0x7f}) //just low enough, see test above
	test.Equals(t, true, ok4)
}
