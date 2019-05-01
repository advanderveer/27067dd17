package engine_test

import (
	"encoding/binary"
	"sync"
	"testing"
	"time"

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
	var mu sync.Mutex
	var handled []*engine.Msg
	h1 := engine.HandlerFunc(func(msg *engine.Msg) {
		mu.Lock()
		defer mu.Unlock()
		handled = append(handled, msg)
	})
	o1 := engine.NewOutOfOrder(h1)

	msg1 := &engine.Msg{}
	o1.Handle(msg1)
	time.Sleep(time.Millisecond)
	mu.Lock()
	test.Equals(t, []*engine.Msg{&engine.Msg{}}, handled)
	mu.Unlock()

	msg2 := &engine.Msg{Block: &onl.Block{Prev: bid1}}
	o1.Handle(msg2)
	time.Sleep(time.Millisecond)
	mu.Lock()
	test.Equals(t, []*engine.Msg{&engine.Msg{}}, handled) //deferred
	mu.Unlock()

	o1.Resolve(bid1)
	time.Sleep(time.Millisecond)
	mu.Lock()
	test.Equals(t, []*engine.Msg{msg1, msg2}, handled) //now resolved
	mu.Unlock()

	o1.Resolve(bid2)
	time.Sleep(time.Millisecond)
	mu.Lock()
	test.Equals(t, []*engine.Msg{msg1, msg2}, handled) //should have done nothing
	mu.Unlock()

	o1.Handle(msg2)
	time.Sleep(time.Millisecond)
	mu.Lock()
	test.Equals(t, []*engine.Msg{msg1, msg2, msg2}, handled) //already resolved
	mu.Unlock()
}

func TestOutOfOrderBeforeAnyHandle(t *testing.T) {
	var mu sync.Mutex
	var handled []*engine.Msg
	h1 := engine.HandlerFunc(func(msg *engine.Msg) {
		mu.Lock()
		defer mu.Unlock()
		handled = append(handled, msg)
	})
	o1 := engine.NewOutOfOrder(h1)

	o1.Resolve(bid1)
	msg2 := &engine.Msg{Block: &onl.Block{Prev: bid1}}
	o1.Handle(msg2)
	time.Sleep(time.Millisecond)

	mu.Lock()
	test.Equals(t, []*engine.Msg{msg2}, handled) //should have handled the messages
	mu.Unlock()
}

func TestOutOfOrderRoundAndBlock(t *testing.T) {
	t.Run("block then round", func(t *testing.T) {
		var mu sync.Mutex
		var handled []*engine.Msg
		h1 := engine.HandlerFunc(func(msg *engine.Msg) {
			mu.Lock()
			defer mu.Unlock()
			handled = append(handled, msg)
		})
		o1 := engine.NewOutOfOrder(h1)

		msg2 := &engine.Msg{Block: &onl.Block{Round: 1, Prev: bid1}}
		o1.Handle(msg2)

		mu.Lock()
		test.Equals(t, 0, len(handled)) //not handled (round+prev)
		mu.Unlock()
		o1.Resolve(bid1)
		time.Sleep(time.Millisecond)
		mu.Lock()
		test.Equals(t, 0, len(handled)) //not handled, round
		mu.Unlock()
		o1.ResolveRound(1)
		time.Sleep(time.Millisecond)

		mu.Lock()
		test.Equals(t, []*engine.Msg{msg2}, handled)
		mu.Unlock()
	})

	t.Run("round then block", func(t *testing.T) {
		var mu sync.Mutex
		var handled []*engine.Msg
		h1 := engine.HandlerFunc(func(msg *engine.Msg) {
			mu.Lock()
			defer mu.Unlock()
			handled = append(handled, msg)
		})
		o1 := engine.NewOutOfOrder(h1)

		msg2 := &engine.Msg{Block: &onl.Block{Round: 1, Prev: bid1}}
		o1.Handle(msg2)

		mu.Lock()
		test.Equals(t, 0, len(handled)) //not handled (round+prev)
		mu.Unlock()
		o1.ResolveRound(1)
		time.Sleep(time.Millisecond)
		mu.Lock()
		test.Equals(t, 0, len(handled)) //not handled, round
		mu.Unlock()
		o1.Resolve(bid1)
		time.Sleep(time.Millisecond)

		mu.Lock()
		test.Equals(t, []*engine.Msg{msg2}, handled)
		mu.Unlock()
	})
}

func TestOutOfOrderConcurrency(t *testing.T) {
	h1 := engine.HandlerFunc(func(msg *engine.Msg) {})
	o1 := engine.NewOutOfOrder(h1)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		for i := uint64(0); i < 10000; i++ {
			o1.ResolveRound(i)

			var id onl.ID
			binary.BigEndian.PutUint64(id[:], i)
			o1.Resolve(id)
		}

	}()

	go func() {
		defer wg.Done()

		for i := uint64(0); i < 10000; i++ {
			var id onl.ID
			binary.BigEndian.PutUint64(id[:], i)
			o1.Handle(&engine.Msg{Block: &onl.Block{Prev: id, Round: i}})
		}
	}()

	wg.Wait()

}
