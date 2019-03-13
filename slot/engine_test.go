package slot_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

func TestBasicMessageHandling(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	e1 := slot.NewEngine(pk1, sk1)
	netw := slot.NewMemNetwork()

	ep1 := netw.Endpoint()
	doneCh := make(chan error)
	go func() {
		doneCh <- e1.Run(ep1)
	}()

	for i := uint64(0); i < 100; i++ {
		b := slot.NewBlock(i, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
		test.Ok(t, netw.Write(&slot.Msg{Proposal: b}))
	}

	test.Ok(t, ep1.Close())                           //should cause broadcast to close down
	test.Equals(t, slot.ErrBroadcastClosed, <-doneCh) //allow engine to close

	rx, tx := e1.Stats()
	test.Equals(t, uint64(100), rx)
	test.Equals(t, uint64(0), tx)
}

type errbc struct {
	E error
}

func (e errbc) Read(m *slot.Msg) (err error)  { return e.E }
func (e errbc) Write(m *slot.Msg) (err error) { return e.E }

func TestReadError(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	err1 := errors.New("foo")
	e1 := slot.NewEngine(pk1, sk1)
	err = e1.Run(errbc{err1})
	test.Assert(t, err != nil, "should result in error")

	msge := err.(slot.MsgError)
	test.Equals(t, err1, msge.E)
	test.Equals(t, "failed to read message from broadcast on n=1 (type: 0): foo", msge.Error())

}

func TestHandleError(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	e1 := slot.NewEngine(pk1, sk1)
	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()

	err = ep1.Write(&slot.Msg{}) //should result in unkown message
	test.Ok(t, err)

	err = e1.Run(ep1)
	test.Assert(t, err != nil, "should result in error")

	msge := err.(slot.MsgError)
	test.Equals(t, slot.ErrUnknownMessage, msge.E)
	test.Equals(t, "failed to handle rx message on n=1 (type: 0): read unkown message from broadcast", msge.Error())

}
