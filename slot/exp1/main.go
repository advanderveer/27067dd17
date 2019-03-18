package main

import (
	"bytes"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
)

//collect all messages as a new endpoint on the broadcast network, done will close
//the endpoint and return a channel that can be read to get all messages it saw
func collect(netw *slot.MemNetwork) (done func() chan []*slot.Msg) {
	ep := netw.Endpoint()
	donec := make(chan []*slot.Msg)
	go func() {
		msgs := []*slot.Msg{}
		for {
			msg := &slot.Msg{}
			err := ep.Read(msg)
			if err == io.EOF {
				donec <- msgs
				return
			}

			if err != nil {
				panic(err)
			}

			msgs = append(msgs, msg)
		}
	}()

	return func() chan []*slot.Msg {
		err := ep.Close()
		if err != nil {
			panic(err)
		}

		return donec
	}
}

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

	//member 1
	ep1 := netw.Endpoint()
	c1 := slot.NewChain()
	e1 := slot.NewEngine(os.Stderr, c1, pk1, sk1, ep1, time.Millisecond*100, 1)
	err := e1.WorkNewTip()
	if err != nil {
		panic(err)
	}

	//member 2
	ep2 := netw.Endpoint()
	c2 := slot.NewChain()
	e2 := slot.NewEngine(os.Stderr, c2, pk2, sk2, ep2, time.Millisecond*100, 1)
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

	go func() {
		err := e2.Run()
		if err != nil {
			panic(err)
		}
	}()

	// @TODO will eventually deadlock, sometimes it takes 10k rounds but it inevitable
	// how do we fix that

	<-sigCh
}
