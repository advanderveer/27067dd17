package slot_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/go-test"
)

func TestBasicMessageHandling(t *testing.T) {
	pk1, sk1, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()
	e1 := slot.NewEngine(ioutil.Discard, pk1, sk1, ep1, 0, 0)

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
	test.Equals(t, uint64(100), rx) //@TODO actually 100?
	test.Equals(t, uint64(1), tx)   //@TODO actually zero?
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
	e1 := slot.NewEngine(ioutil.Discard, pk1, sk1, errbc{err1}, 0, 0)
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
	e1 := slot.NewEngine(ioutil.Discard, pk1, sk1, ep1, 0, 0)

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
		for {
			msg := &slot.Msg{}
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

func TestMessageHandlingStepByStep(t *testing.T) {
	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	netw := slot.NewMemNetwork()
	ep1 := netw.Endpoint()
	bt := time.Millisecond * 50
	e1 := slot.NewEngine(ioutil.Discard, pk1, sk1, ep1, bt, 1)

	t.Run("propose a block build up from genesis", func(t *testing.T) {
		coll1 := collect(t, netw)

		//sending 10 messages schouldn't change anything, as they get de-duplicated
		//and no voter is there to turn the proposal into a vote
		n := uint64(1)
		for i := uint64(0); i < n; i++ {
			err := e1.WorkNewTip()
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

	t.Run("handle proposal as potential votes", func(t *testing.T) {
		coll1 := collect(t, netw)

		p1 := &slot.Msg{}
		err := ep1.Read(p1)
		test.Ok(t, err)
		test.Equals(t, uint64(1), p1.Proposal.Round) //round 1 proposal read

		err = e1.HandleProposal(p1.Proposal)
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

	t.Run("test proposals coming in and stackin up, released due to time out", func(t *testing.T) {
		coll1 := collect(t, netw)

		//voter should have written his votes to the network after twice the blocktime
		time.Sleep(bt * 2)

		msgs := <-coll1()
		test.Equals(t, 1, len(msgs))
		test.Equals(t, uint64(1), msgs[0].Vote.Round) //should be a vote for round 1
	})

	var voter *slot.Voter
	t.Run("new proposal that comes (and is higher) turn into a vote right away", func(t *testing.T) {
		coll1 := collect(t, netw)

		//imagine another chain with another pk
		rb := make([]byte, 32)
		rb[0] = 0x07 //this eventually draws a ticket that is higher then the current block
		pk2, sk2, _ := vrf.GenerateKey(bytes.NewReader(rb))
		c2 := slot.NewChain()
		ticket, err := c2.Draw(pk2, sk2, c2.Tip(), 1)
		voter = slot.NewVoter(ioutil.Discard, 1, c2, ticket, pk2)
		test.Ok(t, err)
		b2 := slot.NewBlock(1, c2.Tip(), ticket.Data, ticket.Proof, pk2)

		//the new proposal should turn into a vote right away
		err = e1.HandleProposal(b2)
		test.Ok(t, err)

		msgs := <-coll1()
		test.Equals(t, 2, len(msgs))                            //should see just one proposal
		test.Equals(t, ticket.Data, msgs[0].Proposal.Ticket[:]) //should be relayed
		test.Equals(t, ticket.Data, msgs[1].Vote.Ticket[:])     //should be turned into vote
	})

	t.Run("close round for causing votes to be broadcasted to the network", func(t *testing.T) {
		coll1 := collect(t, netw)

		//read the vote that is there for us
		v1 := &slot.Msg{}
		err := ep1.Read(v1)
		test.Ok(t, err)
		test.Equals(t, uint64(1), v1.Vote.Round)

		//handle vote (1)
		err = e1.HandleVote(v1.Vote)
		test.Ok(t, err)

		//the exact same vote shouldn't count
		err = e1.HandleVote(v1.Vote)
		test.Ok(t, err)

		//@TODO voter not be set to nil yet
		//@TODO any out of order messages are not yet resolved

		//should not have enough votes to have the block be appended
		test.Equals(t, (*slot.Block)(nil), e1.Chain().Read(v1.Vote.BlockHash()))

		//we imagine another voter that signs another vote (side channel)
		v2 := voter.Vote(v1.Vote.Block)

		//this new vote should count, and be relayed
		err = e1.HandleVote(v2)
		test.Ok(t, err)

		//@TODO test that we have a new voter
		//@TODO test if message order is resolved

		//should not have enough votes to have the block be appended
		test.Equals(t, v2.Block, e1.Chain().Read(v1.Vote.BlockHash()))

		msgs := <-coll1()
		test.Equals(t, 4, len(msgs))                      //v1, v2 relay, closing vote and proposal for the new round
		test.Equals(t, uint64(2), msgs[3].Proposal.Round) //new proposal, and the cycle starts
	})

	//@TODO create a close method
	time.Sleep(time.Millisecond * 100) //wait for all timers to die down

}

func Test2MemberDeadlock(t *testing.T) {

	//setup network
	netw := slot.NewMemNetwork()
	coll := collect(t, netw)

	//-------
	//prep debug info: @TODO make utility for this
	//-------
	rnd1 := make([]byte, 32)
	rnd1[0] = 0x01
	rnd2 := make([]byte, 32)
	rnd2[0] = 0x02

	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(rnd1)) //ana
	pk2, sk2, _ := vrf.GenerateKey(bytes.NewBuffer(rnd2)) //bob
	defer func() { slot.PKString = slot.DefaultPKString }()
	slot.PKString = func(pk []byte) string {
		if bytes.Equal(pk1, pk) {
			return "ana"
		} else if bytes.Equal(pk2, pk) {
			return "bob"
		}

		return slot.DefaultPKString(pk)
	}

	defer func() { slot.BlockName = slot.DefaultBlockName }()
	slot.BlockName = func(id slot.ID) string {
		switch hex.EncodeToString(id[:]) {
		case "20bb90498d6b8d870a910ba8cefd7a06c08a56c2fea7889fea39846291185d5b":
			return "rank1"
		case "cf71746031e015cd91273fa52182f1f5e25c2286f50dbc30b39280da15905e55":
			return "rank2"
		case "6d9c54dee5660c46886f32d80e57e9dd0ffa57ee0cd2a762b036d9c8e0c3a33a":
			return "genesis"
		}

		return slot.DefaultBlockName(id)
	}

	//-------
	//end prep debug info: @TODO make a utility for this
	//-------

	//member 1
	ep1 := netw.Endpoint()
	e1 := slot.NewEngine(os.Stderr, pk1, sk1, ep1, time.Millisecond*5, 1)
	test.Ok(t, e1.WorkNewTip())

	//member 2
	ep2 := netw.Endpoint()
	e2 := slot.NewEngine(os.Stderr, pk2, sk2, ep2, time.Millisecond*5, 1)
	test.Ok(t, e2.WorkNewTip())

	//in the timeline for this test we shouldn't deadlock on the fact that proposals
	//only stat to come in after the intial blocktime has expired for voters.
	time.Sleep(time.Millisecond * 10)

	go func() {
		test.Ok(t, e1.Run())
	}()

	go func() {
		test.Ok(t, e2.Run())
	}()

	//wait for a few rounds
	time.Sleep(time.Millisecond * 400)

	//collect all messages
	msgs := <-coll()
	test.Assert(t, len(msgs) > 10, "should do a decent amount of messages, got: %d", len(msgs))
}

// @TODO fix the situation below, it seems to deadlock in the same way even when
// there is no need for min size
// func Test2MemberDeadlock0minSize(t *testing.T) {
// 	netw := slot.NewMemNetwork()
// 	coll := collect(t, netw)
//
// 	fmt.Println("------")
//
// 	//member 1
// 	ep1 := netw.Endpoint()
// 	pk1, sk1, _ := vrf.GenerateKey(rand.Reader)
// 	fmt.Printf("pk1: %.6x\n", pk1)
// 	e1 := slot.NewEngine(pk1, sk1, ep1, time.Millisecond*5, 0)
// 	test.Ok(t, e1.HandleVoteIntoNewTip(ep1))
//
// 	//member 2
// 	ep2 := netw.Endpoint()
// 	pk2, sk2, _ := vrf.GenerateKey(rand.Reader)
// 	fmt.Printf("pk2: %.6x\n", pk2)
// 	e2 := slot.NewEngine(pk2, sk2, ep2, time.Millisecond*5, 0)
// 	test.Ok(t, e2.HandleVoteIntoNewTip(ep2))
//
// 	time.Sleep(time.Millisecond * 10)
//
// 	go func() {
// 		test.Ok(t, e1.Run())
// 	}()
//
// 	go func() {
// 		test.Ok(t, e2.Run())
// 	}()
//
// 	time.Sleep(time.Millisecond * 400)
// 	fmt.Println("e1/e2 round:", e1.Chain().Read(e1.Chain().Tip()).Round, e2.Chain().Read(e2.Chain().Tip()).Round)
//
// 	//@TODO basic test seems to get stuck sometimes on 0,0
// 	//enabling race detector tiggers that situation consistently
// 	msgs := <-coll()
// 	for _, msg := range msgs {
// 		fmt.Println(msg)
// 	}
//
// 	fmt.Println("------")
// 	test.Assert(t, len(msgs) > 10, "should do a decent amount of messages, got: %d", len(msgs))
// }
