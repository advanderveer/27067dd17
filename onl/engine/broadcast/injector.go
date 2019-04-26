package broadcast

import (
	"io"
	"sync"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
)

//Injector allows for writing specific messages to broadcast, mainly usefull for
//black box testing of the protocol
type Injector struct {
	mu   sync.RWMutex
	coll chan []*engine.Msg
	idn  *onl.Identity
	*Mem
}

//NewInjector creates an injector with identity using the reader for random bytes
func NewInjector(rndid []byte, bufn int) (inj *Injector) {
	inj = &Injector{
		Mem:  NewMem(bufn),
		idn:  onl.NewIdentity(rndid),
		coll: make(chan []*engine.Msg),
	}

	go func() {
		var msgs []*engine.Msg

		for {
			msg := &engine.Msg{}
			err := inj.Read(msg)
			if err == io.EOF {
				inj.coll <- msgs
				return
			} else if err != nil {
				panic("injector failed to collect: " + err.Error())
			}

			msgs = append(msgs, msg)
		}
	}()

	return
}

//Collect closes the injector's message reader and return all messages read
func (inj *Injector) Collect() []*engine.Msg {
	err := inj.Close()
	if err != nil {
		panic("failed to close message reader: " + err.Error())
	}

	return <-inj.coll
}

// Mint a block and broadcast it to the network
func (inj *Injector) Mint(ts uint64, prev, prevf onl.ID, round uint64) (b *onl.Block) {
	b = inj.idn.Mint(ts, prev, prevf, round)
	err := inj.Write(&engine.Msg{Block: b})
	if err != nil {
		panic("failed to inject proposal: " + err.Error())
	}

	return b
}
