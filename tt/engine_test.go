package tt_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/tt"
	"github.com/advanderveer/go-test"
)

func TestEngineShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	br1 := blockReader() //@TODO make an actual chain?
	m1 := tt.NewMiner(br1)
	idn1 := tt.NewIdentity([]byte{0x01})
	bc1 := tt.NewMemBroadcast(0)
	e1 := tt.NewEngine(ioutil.Discard, idn1, bc1, m1)

	inj1 := tt.NewInjector([]byte{0x02}, 0)
	inj1.To(bc1)
	inj1.Vote(bid1)
	time.Sleep(time.Millisecond)

	test.Ok(t, e1.Shutdown(ctx))
}
