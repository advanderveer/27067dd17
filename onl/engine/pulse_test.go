package engine_test

import (
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

var _ engine.Pulse = &engine.MemPulse{}

func TestMemOscillator(t *testing.T) {
	osc := engine.NewMemOscillator()

	p1 := osc.Pulse()
	osc.Fire()
	test.Ok(t, p1.Next())

	test.Ok(t, p1.Close())
	test.Equals(t, io.EOF, p1.Next())

	osc.Fire()             //closed pulse shouldn't panic
	test.Ok(t, p1.Close()) //closed channel shouldn't panic
}
