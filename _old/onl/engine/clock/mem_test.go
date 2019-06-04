package clock_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/clock"
	"github.com/advanderveer/go-test"
)

var _ engine.Clock = &clock.MemClock{}

func TestMemOscillator(t *testing.T) {
	osc := clock.NewMemOscillator()

	p1 := osc.Clock()
	osc.Fire()
	n, ts, err := p1.Next()
	test.Equals(t, uint64(2), n)
	test.Equals(t, uint64(2), p1.Round())
	test.Assert(t, ts > 0, "ts should be higher then zero")
	test.Ok(t, err)

	test.Ok(t, p1.Close())
	n, ts, err = p1.Next()
	test.Equals(t, uint64(0), n)
	test.Equals(t, uint64(0), ts)
	test.Equals(t, io.EOF, err)

	osc.Fire()             //closed pulse shouldn't panic
	test.Ok(t, p1.Close()) //closed channel shouldn't panic
}
