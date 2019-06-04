package ripple_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/ripple"
)

var (
	tx1 = []byte{0x01}
	tx2 = []byte{0x02}
	tx3 = []byte{0x03}
)

func TestRoundConvergence(t *testing.T) {
	netw := ripple.NewMemNetwork()
	coll := ripple.Collect(netw.Endpoint())
	txs1 := ripple.Txs(tx1)
	txs2 := ripple.Txs(tx2)

	r1 := ripple.NewRound(netw.Endpoint(), txs1)
	r2 := ripple.NewRound(netw.Endpoint(), txs2)

	r1.Start()
	r2.Start()

	_ = r1
	_ = r2

	time.Sleep(time.Second)

	msgs := <-coll()
	fmt.Println(msgs)
}
