package engine_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

var _ engine.Handler = &engine.OutOfOrder{}
var _ engine.Handler = engine.HandlerFunc(nil)

var (
	//some test ids
	bid1 = onl.ID{}
	bid2 = onl.ID{}
	bid3 = onl.ID{}
	bid4 = onl.ID{}
)

func init() {
	bid1[0] = 0x01
	bid2[0] = 0x02
	bid3[0] = 0x03
	bid4[0] = 0x04
}

func TestOoOHandling(t *testing.T) {
	var handled []*engine.Msg
	h1 := engine.HandlerFunc(func(msg *engine.Msg) { handled = append(handled, msg) })
	o1 := engine.NewOutOfOrder(h1)

	msg1 := &engine.Msg{}
	o1.Handle(msg1)
	test.Equals(t, []*engine.Msg{&engine.Msg{}}, handled)

	msg2 := &engine.Msg{Block: &onl.Block{Prev: bid1}}
	o1.Handle(msg2)
	test.Equals(t, []*engine.Msg{&engine.Msg{}}, handled) //deferred

	o1.Resolve(bid1)
	test.Equals(t, []*engine.Msg{msg1, msg2}, handled) //now resolved

	o1.Resolve(bid2)
	test.Equals(t, []*engine.Msg{msg1, msg2}, handled) //should have done nothing

	o1.Handle(msg2)
	test.Equals(t, []*engine.Msg{msg1, msg2, msg2}, handled) //already resolved
}

func TestOutOfOrderBeforeAnyHandle(t *testing.T) {
	var handled []*engine.Msg
	h1 := engine.HandlerFunc(func(msg *engine.Msg) { handled = append(handled, msg) })
	o1 := engine.NewOutOfOrder(h1)

	o1.Resolve(bid1)
	msg2 := &engine.Msg{Block: &onl.Block{Prev: bid1}}
	o1.Handle(msg2)

	test.Equals(t, []*engine.Msg{msg2}, handled) //should have handled the messages
}

func TestOutOfOrderRoundAndBlock(t *testing.T) {

	t.Run("block then round", func(t *testing.T) {
		var handled []*engine.Msg
		h1 := engine.HandlerFunc(func(msg *engine.Msg) { handled = append(handled, msg) })
		o1 := engine.NewOutOfOrder(h1)

		msg2 := &engine.Msg{Block: &onl.Block{Round: 1, Prev: bid1}}
		o1.Handle(msg2)

		test.Equals(t, 0, len(handled)) //not handled (round+prev)
		o1.Resolve(bid1)
		test.Equals(t, 0, len(handled)) //not handled, round
		o1.ResolveRound(1)

		test.Equals(t, []*engine.Msg{msg2}, handled)
	})

	t.Run("round then block", func(t *testing.T) {
		var handled []*engine.Msg
		h1 := engine.HandlerFunc(func(msg *engine.Msg) { handled = append(handled, msg) })
		o1 := engine.NewOutOfOrder(h1)

		msg2 := &engine.Msg{Block: &onl.Block{Round: 1, Prev: bid1}}
		o1.Handle(msg2)

		test.Equals(t, 0, len(handled)) //not handled (round+prev)
		o1.ResolveRound(1)
		test.Equals(t, 0, len(handled)) //not handled, round
		o1.Resolve(bid1)

		test.Equals(t, []*engine.Msg{msg2}, handled)
	})
}
