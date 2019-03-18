package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
)

func main() {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)

	//setup network
	netw := slot.NewMemNetwork()

	//prep debug names and deterministic block input
	rnd1 := make([]byte, 32)
	rnd1[0] = 0x01
	rnd2 := make([]byte, 32)
	rnd2[0] = 0x02
	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(rnd1)) //ana
	pk2, sk2, _ := vrf.GenerateKey(bytes.NewReader(rnd2)) //bob
	fmt.Printf("invalid vote proof pk1: %.6x pk2: %.6x\n", pk1, pk2)

	//member 1
	ep1 := netw.Endpoint()
	c1 := slot.NewChain()
	e1 := slot.NewEngine(os.Stdout, c1, pk1, sk1, ep1, time.Microsecond, 0)
	err := e1.WorkNewTip()
	if err != nil {
		panic(err)
	}

	//member 2
	ep2 := netw.Endpoint()
	c2 := slot.NewChain()
	e2 := slot.NewEngine(os.Stdout, c2, pk2, sk2, ep2, time.Microsecond, 0)
	err = e2.WorkNewTip()
	if err != nil {
		panic(err)
	}

	//in the timeline for this test we shouldn't deadlock on the fact that proposals
	//only start to come in after the intial blocktime has expired for voters.
	// time.Sleep(time.Millisecond * 10)
	go func() {
		err := e1.Run()
		if err != nil {
			panic(err)
		}
	}()

	// go func() {
	// 	err := e2.Run()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()

	// @TODO will eventually deadlock, sometimes it takes 10k rounds but it inevitable
	// how do we fix that

	<-sigCh
}
