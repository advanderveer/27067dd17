package slot_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

func TestBasicMessageHandling(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()
	e1 := slot.NewEngine(pk1, sk1, ep1)

	doneCh := make(chan error)
	go func() {
		doneCh <- e1.Run()
	}()

	for i := uint64(0); i < 100; i++ {
		b := slot.NewBlock(i, slot.NilID, slot.NilTicket, slot.NilProof, slot.NilPK)
		test.Ok(t, netw.Write(&slot.Msg{Proposal: b}))
	}

	test.Ok(t, ep1.Close())                           //should cause broadcast to close down
	test.Equals(t, slot.ErrBroadcastClosed, <-doneCh) //allow engine to close

	rx, tx, votes := e1.Stats()
	test.Equals(t, uint64(100), rx)
	test.Equals(t, uint64(0), tx)
	test.Assert(t, votes == nil, "should not be voter")
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
	e1 := slot.NewEngine(pk1, sk1, errbc{err1})
	err = e1.Run()
	test.Assert(t, err != nil, "should result in error")

	msge := err.(slot.MsgError)
	test.Equals(t, err1, msge.E)
	test.Equals(t, "failed to read message from broadcast on n=1 (type: 0): foo", msge.Error())
}

func TestHandleError(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()
	e1 := slot.NewEngine(pk1, sk1, ep1)

	err = ep1.Write(&slot.Msg{}) //should result in unkown message
	test.Ok(t, err)

	err = e1.Run()
	test.Assert(t, err != nil, "should result in error")

	msge := err.(slot.MsgError)
	test.Equals(t, slot.ErrUnknownMessage, msge.E)
	test.Equals(t, "failed to handle rx message on n=1 (type: 0): read unkown message from broadcast", msge.Error())
}

//collect all messages as a new endpoint on the broadcast network, done will close
//the endpoint and return a channel that can be read to get all messages it saw
func collect(t testing.TB, netw *slot.MemNetwork) (done func() chan []*slot.Msg) {
	ep := netw.Endpoint()
	donec := make(chan []*slot.Msg)
	go func() {
		msgs := []*slot.Msg{}
		msg := &slot.Msg{}
		for {
			err := ep.Read(msg)
			if err == io.EOF {
				donec <- msgs
				return
			}

			test.Ok(t, err)
			msgs = append(msgs, msg)
		}
	}()

	return func() chan []*slot.Msg {
		test.Ok(t, ep.Close())
		return donec
	}
}

func TestHandleVoteIntoNewTip(t *testing.T) {
	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()
	e1 := slot.NewEngine(pk1, sk1, ep1)

	t.Run("propose from genesis", func(t *testing.T) {
		coll1 := collect(t, netw)

		//sending 10 messages schouldn't change anything, as they get de-duplicated
		//and no voter is there to turn the proposal into a vote
		n := uint64(1)
		for i := uint64(0); i < n; i++ {
			err := e1.HandleVoteIntoNewTip(ep1)
			test.Ok(t, err)
		}

		msgs := <-coll1()
		test.Equals(t, 1, len(msgs))
		test.Equals(t, uint64(1), msgs[0].Proposal.Round)

		rx, tx, votes := e1.Stats()
		test.Equals(t, uint64(0), rx)
		test.Equals(t, n, tx)
		test.Assert(t, votes != nil, "should be voter")
		test.Equals(t, 0, len(votes)) //no votes
	})

	t.Run("handle proposal", func(t *testing.T) {
		coll1 := collect(t, netw)

		p1 := &slot.Msg{}
		err := ep1.Read(p1)
		test.Ok(t, err)
		test.Equals(t, uint64(1), p1.Proposal.Round) //round 1 proposal read

		err = e1.HandleProposal(p1.Proposal, ep1)
		test.Ok(t, err)

		//@TODO voter should have the proposed block in its highest votes as it
		//is the only one it would have seen

		msgs := <-coll1()
		test.Equals(t, 1, len(msgs))
		test.Equals(t, p1, msgs[0]) //should have relayed the proposal

		rx, _, votes := e1.Stats()
		test.Equals(t, uint64(0), rx)
		test.Equals(t, 1, len(votes)) //1 vote cast
	})
}
