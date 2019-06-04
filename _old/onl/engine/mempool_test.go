package engine_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

func TestBasicMemPool(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	p1 := engine.NewMemPool()

	st1, _ := onl.NewState(nil)
	w1 := st1.Update(func(kv *onl.KV) {
		kv.Set([]byte{0x01}, []byte{0x02})
	})

	test.Ok(t, w1.GenerateNonce())
	w1.PK = idn1.PK()
	test.Equals(t, engine.ErrInvalidWriteSignature, p1.Add(w1))

	idn1.SignWrite(w1)

	test.Ok(t, p1.Add(w1))
	test.Equals(t, engine.ErrAlreadyInPool, p1.Add(w1))

	t.Run("should pick if not applied", func(t *testing.T) {
		var picked []*onl.Write
		p1.Pick(st1, func(w *onl.Write) bool {
			picked = append(picked, w)
			return false
		})

		test.Equals(t, []*onl.Write{w1}, picked)
	})

	t.Run("should not pick if applied", func(t *testing.T) {
		test.Ok(t, st1.Apply(w1, false))

		var picked []*onl.Write
		p1.Pick(st1, func(w *onl.Write) bool {
			picked = append(picked, w)
			return false
		})

		test.Equals(t, 0, len(picked))
	})

}
