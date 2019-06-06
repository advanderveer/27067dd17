package tt_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

var _ tt.Handler = &tt.OutOfOrder{}
var _ tt.Handler = tt.HandlerFunc(nil)

func TestOoOHandling(t *testing.T) {
	var handled []*tt.Msg
	h1 := tt.HandlerFunc(func(msg *tt.Msg) { handled = append(handled, msg) })
	o1 := tt.NewOutOfOrder(h1)
	_ = o1

	msg1 := &tt.Msg{}
	o1.Handle(msg1)
	test.Equals(t, []*tt.Msg{&tt.Msg{}}, handled)

	msg2 := &tt.Msg{Block: tt.B(bid1, nil)}
	o1.Handle(msg2)
	test.Equals(t, []*tt.Msg{&tt.Msg{}}, handled) //deferred

	o1.Resolve(bid1)
	test.Equals(t, []*tt.Msg{msg1, msg2}, handled) //now resolved

	o1.Resolve(bid2)
	test.Equals(t, []*tt.Msg{msg1, msg2}, handled) //should have done nothing

	o1.Handle(msg2)
	test.Equals(t, []*tt.Msg{msg1, msg2, msg2}, handled) //already resolved

}
