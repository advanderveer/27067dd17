package rev_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/go-test"
)

func TestEmptyProposalIgnore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	logb := bytes.NewBuffer(nil)
	bc1 := rev.NewMemBroadcast()
	e1 := rev.NewEngine(logb, bc1)

	inj1 := rev.NewInjector([]byte{0x01})
	inj1.To(bc1)

	test.Ok(t, inj1.Write(&rev.Msg{}))
	time.Sleep(time.Millisecond)
	test.Equals(t, "engine: [ERRO] Received message without a proposal, ignore it\n", logb.String())

	test.Ok(t, e1.Shutdown(ctx))
}
