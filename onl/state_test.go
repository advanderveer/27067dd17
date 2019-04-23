package onl_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestBasicStateHandling(t *testing.T) {
	s1, err := onl.NewState(nil)
	test.Ok(t, err)

	//nil writes to apply should be no-op
	test.Ok(t, s1.Apply(nil, false))

	w1 := s1.Update(func(kv *onl.KV) {
		kv.Set([]byte{0x01}, []byte{0x02})
	})

	s2, err := onl.NewState([][]*onl.Write{{w1}})
	test.Ok(t, err)

	s2.View(func(kv *onl.KV) {
		test.Equals(t, []byte{0x02}, kv.Get([]byte{0x01}))
	})

	//create a write that reads from the key writtine by 'w1', should conflict
	w2 := s1.Update(func(kv *onl.KV) {
		kv.Set([]byte{0x02}, kv.Get([]byte{0x01}))
	})

	_, err = onl.NewState([][]*onl.Write{{w1, w2}})
	test.Equals(t, onl.ErrApplyConflict, err) //yep, conflicts
}
