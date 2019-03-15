package slot_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
	c1 := slot.NewChain()
	e1 := slot.NewEngine(ioutil.Discard, c1, pk1, sk1, ep1, 0, 0)

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
	c1 := slot.NewChain()
	e1 := slot.NewEngine(ioutil.Discard, c1, pk1, sk1, errbc{err1}, 0, 0)
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
	c1 := slot.NewChain()
	e1 := slot.NewEngine(ioutil.Discard, c1, pk1, sk1, ep1, 0, 0)

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
	c1 := slot.NewChain()
	e1 := slot.NewEngine(ioutil.Discard, c1, pk1, sk1, ep1, bt, 1)

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
		test.Equals(t, (*slot.Block)(nil), c1.Read(v1.Vote.BlockHash()))

		//we imagine another voter that signs another vote (side channel)
		v2 := voter.Vote(v1.Vote.Block)

		//this new vote should count, and be relayed
		err = e1.HandleVote(v2)
		test.Ok(t, err)

		//@TODO test that we have a new voter
		//@TODO test if message order is resolved

		//should not have enough votes to have the block be appended
		test.Equals(t, v2.Block, c1.Read(v1.Vote.BlockHash()))

		msgs := <-coll1()
		test.Equals(t, 4, len(msgs))                      //v1, v2 relay, closing vote and proposal for the new round
		test.Equals(t, uint64(2), msgs[3].Proposal.Round) //new proposal, and the cycle starts
	})

	//@TODO create a close method
	time.Sleep(time.Millisecond * 100) //wait for all timers to die down
}

func namePK(pkn []byte, name string) (reset func()) {
	old := slot.PKString
	slot.PKString = func(pk []byte) string {
		if bytes.Equal(pk, pkn) {
			return name
		}

		return old(pk)
	}

	return func() {
		slot.PKString = old
	}
}

func nameID(idhex string, name string) (reset func()) {
	old := slot.BlockName
	id, err := hex.DecodeString(idhex)
	if err != nil {
		panic(err)
	}

	slot.BlockName = func(idd slot.ID) string {
		if bytes.HasPrefix(idd[:], id[:]) {
			return name
		}

		return old(idd)
	}

	return func() {
		slot.BlockName = old
	}
}

func drawPNG(t *testing.T, buf io.Reader, name string) {
	f, err := os.Create(name)
	test.Ok(t, err)
	defer f.Close()

	cmd := exec.Command("dot", "-Tpng")
	cmd.Stdin = buf
	cmd.Stdout = f
	test.Ok(t, cmd.Run())
}

func draw(t testing.TB, c *slot.Chain, w io.Writer) {
	fmt.Fprintln(w, `digraph {`)
	tip := c.Tip()

	test.Ok(t, c.Each(func(id slot.ID, b *slot.Block) error {
		fmt.Fprintf(w, "\t"+`"%.6x" [shape=box,style="filled,solid",label="%.6x:%d"`, id, id, b.Round)

		if id == tip {
			fmt.Fprintf(w, `,fillcolor="#DDDDDD"`)
		} else {
			fmt.Fprintf(w, `,fillcolor="#ffffff"`)
		}

		//@TODO add any styling

		fmt.Fprintf(w, "]\n")
		fmt.Fprintf(w, "\t"+`"%.6x" -> "%.6x";`+"\n", id, b.Prev)

		return nil
	}))

	fmt.Fprintln(w, `}`)
}

func Test2MemberDeadlockAfterBlockTime(t *testing.T) {

	//setup network
	netw := slot.NewMemNetwork()
	coll := collect(t, netw)
	logs := bytes.NewBuffer(nil)

	//prep debug names and deterministic block input
	rnd1 := make([]byte, 32)
	rnd1[0] = 0x01
	rnd2 := make([]byte, 32)
	rnd2[0] = 0x02
	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(rnd1)) //ana
	pk2, sk2, _ := vrf.GenerateKey(bytes.NewReader(rnd2)) //bob
	defer namePK(pk1, "ana")()
	defer namePK(pk2, "bob")()
	defer nameID("20bb90498d6b8d870a910ba8cefd7a06c08a56c2fea7889fea39846291185d5b", "rank1")()
	defer nameID("cf71746031e015cd91273fa52182f1f5e25c2286f50dbc30b39280da15905e55", "rank2")()
	defer nameID("6d9c54dee5660c46886f32d80e57e9dd0ffa57ee0cd2a762b036d9c8e0c3a33a", "genesis")()

	//member 1
	ep1 := netw.Endpoint()
	c1 := slot.NewChain()
	e1 := slot.NewEngine(logs, c1, pk1, sk1, ep1, time.Millisecond*5, 1)
	test.Ok(t, e1.WorkNewTip())

	//member 2
	ep2 := netw.Endpoint()
	c2 := slot.NewChain()
	e2 := slot.NewEngine(logs, c2, pk2, sk2, ep2, time.Millisecond*5, 1)
	test.Ok(t, e2.WorkNewTip())

	//in the timeline for this test we shouldn't deadlock on the fact that proposals
	//only start to come in after the intial blocktime has expired for voters.
	// time.Sleep(time.Millisecond * 10)
	l1 := log.New(logs, "-----", log.Lmicroseconds)
	l1.Println()

	go func() {
		test.Ok(t, e1.Run())
	}()

	go func() {
		test.Ok(t, e2.Run())
	}()

	//wait for a few rounds
	time.Sleep(time.Millisecond * 1000)
	l1.Println()

	// @TODO fix and test many: FAILED TO VERIFY false: invalid vote proof

	//collect all messages
	msgs := <-coll()

	exprounds := uint64(20)
	nround1 := c1.Read(c1.Tip()).Round
	nround2 := c2.Read(c2.Tip()).Round
	if nround1 <= exprounds || nround2 <= exprounds {

		//@TODO when this happens we can see that the protocol stops too early, the last
		//message wasn't send close to the closing

		io.Copy(os.Stderr, logs)

		buf := bytes.NewBuffer(nil)
		draw(t, c1, buf)
		drawPNG(t, buf, "00_2m_c1.png")

		buf = bytes.NewBuffer(nil)
		draw(t, c2, buf)
		drawPNG(t, buf, "00_2m_c2.png")

		t.Fatalf("failed to reach enough rounds, got to: %d/%d", nround1, nround2)

		// === RUN   Test2MemberDeadlockAfterBlockTime
		// ana: 16:07:59.082642 [TRAC] draw ticket with new tip 'genesis' as round 1
		// ana: 16:07:59.084306 [INFO] --- drew proposer ticket! proposing block 'rank2'
		// ana: 16:07:59.084366 [INFO] --- drew voter ticket! setup voter for round 1
		// ana: 16:07:59.084370 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// bob: 16:07:59.084387 [TRAC] draw ticket with new tip 'genesis' as round 1
		// ana: 16:07:59.100469 [TRAC] blocktime has passed, and we are still voter, casted 0 votes
		// bob: 16:07:59.104551 [INFO] --- drew proposer ticket! proposing block 'rank1'
		// bob: 16:07:59.104595 [INFO] --- drew voter ticket! setup voter for round 1
		// bob: 16:07:59.104597 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// -----16:07:59.104599
		// bob: 16:07:59.104707 [TRAC] block 'rank1(1)' proposed by 'bob': start handling
		// bob: 16:07:59.104716 [TRAC] block 'rank1(1)' proposed by 'bob' was verified and of the correct round: relaying
		// bob: 16:07:59.107261 [TRAC] block 'rank1(1)' proposed by 'bob' is new highest ranking block for next vote casting
		// ana: 16:07:59.107436 [TRAC] block 'rank2(1)' proposed by 'ana': start handling
		// ana: 16:07:59.107445 [TRAC] block 'rank2(1)' proposed by 'ana' was verified and of the correct round: relaying
		// ana: 16:07:59.110063 [TRAC] block 'rank2(1)' proposed by 'ana' is new highest ranking block for next vote casting
		// ana: 16:07:59.110152 [TRAC] block 'rank1(1)' proposed by 'bob': start handling
		// ana: 16:07:59.110161 [TRAC] block 'rank1(1)' proposed by 'bob' was verified and of the correct round: relaying
		// ana: 16:07:59.112975 [TRAC] block 'rank1(1)' proposed by 'bob' is new highest ranking block for next vote casting
		// ana: 16:07:59.113069 [TRAC] vote from 'ana' for block 'rank2(1)' proposed by 'ana': start handling
		// bob: 16:07:59.117369 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// ana: 16:07:59.117947 [TRAC] verified vote from 'ana' for block 'rank2(1)' proposed by 'ana': relaying
		// ana: 16:07:59.118045 [TRAC] tallied vote from 'ana' for block 'rank2(1)' proposed by 'ana', number of votes: 1
		// ana: 16:07:59.118056 [TRAC] vote from 'ana' for block 'rank2(1)' proposed by 'ana' doesn't cause enough votes (1<2): no progress
		// ana: 16:07:59.118197 [TRAC] vote from 'ana' for block 'rank1(1)' proposed by 'bob': start handling
		// ana: 16:07:59.123316 [TRAC] verified vote from 'ana' for block 'rank1(1)' proposed by 'bob': relaying
		// ana: 16:07:59.123381 [TRAC] tallied vote from 'ana' for block 'rank1(1)' proposed by 'bob', number of votes: 1
		// ana: 16:07:59.123391 [TRAC] vote from 'ana' for block 'rank1(1)' proposed by 'bob' doesn't cause enough votes (1<2): no progress
		// ana: 16:07:59.123484 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob': start handling
		// bob: 16:07:59.126615 [TRAC] block 'rank2(1)' proposed by 'ana': start handling
		// bob: 16:07:59.126626 [TRAC] block 'rank2(1)' proposed by 'ana' was verified and of the correct round: relaying
		// bob: 16:07:59.129241 [TRAC] block 'rank2(1)' proposed by 'ana' was considered but is not the new higest ranking proposal
		// bob: 16:07:59.129340 [TRAC] vote from 'ana' for block 'rank2(1)' proposed by 'ana': start handling
		// bob: 16:07:59.134282 [TRAC] verified vote from 'ana' for block 'rank2(1)' proposed by 'ana': relaying
		// bob: 16:07:59.134348 [TRAC] tallied vote from 'ana' for block 'rank2(1)' proposed by 'ana', number of votes: 1
		// bob: 16:07:59.134358 [TRAC] vote from 'ana' for block 'rank2(1)' proposed by 'ana' doesn't cause enough votes (1<2): no progress
		// bob: 16:07:59.134453 [TRAC] vote from 'ana' for block 'rank1(1)' proposed by 'bob': start handling
		// bob: 16:07:59.139405 [TRAC] verified vote from 'ana' for block 'rank1(1)' proposed by 'bob': relaying
		// bob: 16:07:59.139472 [TRAC] tallied vote from 'ana' for block 'rank1(1)' proposed by 'bob', number of votes: 1
		// bob: 16:07:59.139481 [TRAC] vote from 'ana' for block 'rank1(1)' proposed by 'bob' doesn't cause enough votes (1<2): no progress
		// bob: 16:07:59.139573 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob': start handling
		// bob: 16:07:59.144707 [TRAC] verified vote from 'bob' for block 'rank1(1)' proposed by 'bob': relaying
		// bob: 16:07:59.144772 [TRAC] tallied vote from 'bob' for block 'rank1(1)' proposed by 'bob', number of votes: 2
		// bob: 16:07:59.144782 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' caused enough votes (2>1), progress!
		// bob: 16:07:59.144845 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' while voter was active, casted remaining 1 votes before teardown
		// bob: 16:07:59.144854 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' caused enough votes, appending it's block to chain!
		// bob: 16:07:59.144885 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' has caused a new tip: progress to next round
		// bob: 16:07:59.144886 [TRAC] draw ticket with new tip 'rank1' as round 2
		// bob: 16:07:59.146483 [INFO] --- drew proposer ticket! proposing block 'da1fcdfe'
		// bob: 16:07:59.146531 [INFO] --- drew voter ticket! setup voter for round 2
		// bob: 16:07:59.146533 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// bob: 16:07:59.146624 [TRAC] block 'da1fcdfe(2)' proposed by 'bob': start handling
		// bob: 16:07:59.146634 [TRAC] block 'da1fcdfe(2)' proposed by 'bob' was verified and of the correct round: relaying
		// bob: 16:07:59.149155 [TRAC] block 'da1fcdfe(2)' proposed by 'bob' is new highest ranking block for next vote casting
		// ana: 16:07:59.151542 [TRAC] verified vote from 'bob' for block 'rank1(1)' proposed by 'bob': relaying
		// ana: 16:07:59.151605 [TRAC] tallied vote from 'bob' for block 'rank1(1)' proposed by 'bob', number of votes: 2
		// ana: 16:07:59.151615 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' caused enough votes (2>1), progress!
		// ana: 16:07:59.151679 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' while voter was active, casted remaining 1 votes before teardown
		// ana: 16:07:59.151688 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' caused enough votes, appending it's block to chain!
		// ana: 16:07:59.151716 [TRAC] vote from 'bob' for block 'rank1(1)' proposed by 'bob' has caused a new tip: progress to next round
		// ana: 16:07:59.151718 [TRAC] draw ticket with new tip 'rank1' as round 2
		// bob: 16:07:59.152719 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// bob: 16:07:59.152811 [TRAC] vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob': start handling
		// ana: 16:07:59.153303 [INFO] --- drew proposer ticket! proposing block '12b3d6da'
		// ana: 16:07:59.153345 [INFO] --- drew voter ticket! setup voter for round 2
		// ana: 16:07:59.153346 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// ana: 16:07:59.153440 [TRAC] block 'da1fcdfe(2)' proposed by 'bob': start handling
		// ana: 16:07:59.153449 [TRAC] block 'da1fcdfe(2)' proposed by 'bob' was verified and of the correct round: relaying
		// ana: 16:07:59.155961 [TRAC] block 'da1fcdfe(2)' proposed by 'bob' is new highest ranking block for next vote casting
		// ana: 16:07:59.156054 [TRAC] vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob': start handling
		// bob: 16:07:59.157753 [TRAC] verified vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob': relaying
		// bob: 16:07:59.157818 [TRAC] tallied vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob', number of votes: 1
		// bob: 16:07:59.157827 [TRAC] vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob' doesn't cause enough votes (1<2): no progress
		// bob: 16:07:59.157912 [TRAC] block '12b3d6da(2)' proposed by 'ana': start handling
		// bob: 16:07:59.157922 [TRAC] block '12b3d6da(2)' proposed by 'ana' was verified and of the correct round: relaying
		// bob: 16:07:59.160544 [TRAC] block '12b3d6da(2)' proposed by 'ana' is new highest ranking block for next vote casting
		// bob: 16:07:59.160687 [TRAC] vote from 'bob' for block '12b3d6da(2)' proposed by 'ana': start handling
		// bob: 16:07:59.165672 [TRAC] verified vote from 'bob' for block '12b3d6da(2)' proposed by 'ana': relaying
		// bob: 16:07:59.165740 [TRAC] tallied vote from 'bob' for block '12b3d6da(2)' proposed by 'ana', number of votes: 1
		// bob: 16:07:59.165750 [TRAC] vote from 'bob' for block '12b3d6da(2)' proposed by 'ana' doesn't cause enough votes (1<2): no progress
		// ana: 16:07:59.185957 [TRAC] verified vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob': relaying
		// ana: 16:07:59.186022 [TRAC] tallied vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob', number of votes: 1
		// ana: 16:07:59.186032 [TRAC] vote from 'bob' for block 'da1fcdfe(2)' proposed by 'bob' doesn't cause enough votes (1<2): no progress
		// ana: 16:07:59.186115 [TRAC] block '12b3d6da(2)' proposed by 'ana': start handling
		// ana: 16:07:59.186172 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// ana: 16:07:59.186182 [TRAC] block '12b3d6da(2)' proposed by 'ana' was verified and of the correct round: relaying
		// ana: 16:07:59.188777 [TRAC] block '12b3d6da(2)' proposed by 'ana' is new highest ranking block for next vote casting
		// ana: 16:07:59.188871 [TRAC] vote from 'bob' for block '12b3d6da(2)' proposed by 'ana': start handling
		// ana: 16:07:59.193759 [TRAC] verified vote from 'bob' for block '12b3d6da(2)' proposed by 'ana': relaying
		// ana: 16:07:59.193823 [TRAC] tallied vote from 'bob' for block '12b3d6da(2)' proposed by 'ana', number of votes: 1
		// ana: 16:07:59.193832 [TRAC] vote from 'bob' for block '12b3d6da(2)' proposed by 'ana' doesn't cause enough votes (1<2): no progress
		// ana: 16:07:59.193929 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob': start handling
		// ana: 16:07:59.198850 [TRAC] verified vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob': relaying
		// ana: 16:07:59.198920 [TRAC] tallied vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob', number of votes: 2
		// ana: 16:07:59.198929 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' caused enough votes (2>1), progress!
		// ana: 16:07:59.199002 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' while voter was active, casted remaining 1 votes before teardown
		// ana: 16:07:59.199011 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' caused enough votes, appending it's block to chain!
		// ana: 16:07:59.199045 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' has caused a new tip: progress to next round
		// ana: 16:07:59.199047 [TRAC] draw ticket with new tip 'da1fcdfe' as round 3
		// ana: 16:07:59.200650 [INFO] --- drew proposer ticket! proposing block 'ebe9f65d'
		// ana: 16:07:59.200693 [INFO] --- drew voter ticket! setup voter for round 3
		// ana: 16:07:59.200694 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// ana: 16:07:59.200791 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana': start handling
		// ana: 16:07:59.205773 [TRAC] verified vote from 'ana' for block '12b3d6da(2)' proposed by 'ana': relaying
		// ana: 16:07:59.205840 [TRAC] tallied vote from 'ana' for block '12b3d6da(2)' proposed by 'ana', number of votes: 2
		// ana: 16:07:59.205850 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' caused enough votes (2>1), progress!
		// ana: 16:07:59.205867 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' caused enough votes, appending it's block to chain!
		// ana: 16:07:59.205912 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' has caused a new tip: progress to next round
		// ana: 16:07:59.205913 [TRAC] draw ticket with new tip '12b3d6da' as round 3
		// ana: 16:07:59.207487 [INFO] --- drew proposer ticket! proposing block 'a28e88b8'
		// ana: 16:07:59.207528 [INFO] --- drew voter ticket! setup voter for round 3
		// ana: 16:07:59.207530 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// ana: 16:07:59.207617 [TRAC] block 'ebe9f65d(3)' proposed by 'ana': start handling
		// ana: 16:07:59.207627 [TRAC] block 'ebe9f65d(3)' proposed by 'ana' was verified and of the correct round: relaying
		// bob: 16:07:59.208816 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob': start handling
		// bob: 16:07:59.213709 [TRAC] verified vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob': relaying
		// bob: 16:07:59.213774 [TRAC] tallied vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob', number of votes: 2
		// bob: 16:07:59.213784 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' caused enough votes (2>1), progress!
		// bob: 16:07:59.213847 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' while voter was active, casted remaining 1 votes before teardown
		// bob: 16:07:59.213857 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' caused enough votes, appending it's block to chain!
		// bob: 16:07:59.213888 [TRAC] vote from 'ana' for block 'da1fcdfe(2)' proposed by 'bob' has caused a new tip: progress to next round
		// bob: 16:07:59.213890 [TRAC] draw ticket with new tip 'da1fcdfe' as round 3
		// bob: 16:07:59.215494 [INFO] --- drew proposer ticket! proposing block 'd012ae96'
		// bob: 16:07:59.215538 [INFO] --- drew voter ticket! setup voter for round 3
		// bob: 16:07:59.215539 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// bob: 16:07:59.215635 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana': start handling
		// bob: 16:07:59.220621 [TRAC] verified vote from 'ana' for block '12b3d6da(2)' proposed by 'ana': relaying
		// bob: 16:07:59.220685 [TRAC] tallied vote from 'ana' for block '12b3d6da(2)' proposed by 'ana', number of votes: 2
		// bob: 16:07:59.220695 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' caused enough votes (2>1), progress!
		// bob: 16:07:59.220711 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' caused enough votes, appending it's block to chain!
		// bob: 16:07:59.220755 [TRAC] vote from 'ana' for block '12b3d6da(2)' proposed by 'ana' has caused a new tip: progress to next round
		// bob: 16:07:59.220757 [TRAC] draw ticket with new tip '12b3d6da' as round 3
		// ana: 16:07:59.221716 [TRAC] block 'ebe9f65d(3)' proposed by 'ana' is new highest ranking block for next vote casting
		// ana: 16:07:59.221805 [TRAC] block 'a28e88b8(3)' proposed by 'ana': start handling
		// ana: 16:07:59.221882 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// ana: 16:07:59.221893 [TRAC] block 'a28e88b8(3)' proposed by 'ana' was verified and of the correct round: relaying
		// bob: 16:07:59.222336 [INFO] --- drew proposer ticket! proposing block '276fa2d2'
		// bob: 16:07:59.222379 [INFO] --- drew voter ticket! setup voter for round 3
		// bob: 16:07:59.222380 [TRAC] blocktime is higher then zero, schedule vote casting in 5ms
		// bob: 16:07:59.222467 [TRAC] block 'ebe9f65d(3)' proposed by 'ana': start handling
		// bob: 16:07:59.222476 [TRAC] block 'ebe9f65d(3)' proposed by 'ana' was verified and of the correct round: relaying
		// ana: 16:07:59.224460 [TRAC] block 'a28e88b8(3)' proposed by 'ana' was considered but is not the new higest ranking proposal
		// ana: 16:07:59.224544 [TRAC] block 'd012ae96(3)' proposed by 'bob': start handling
		// ana: 16:07:59.224554 [TRAC] block 'd012ae96(3)' proposed by 'bob' was verified and of the correct round: relaying
		// bob: 16:07:59.224998 [TRAC] block 'ebe9f65d(3)' proposed by 'ana' is new highest ranking block for next vote casting
		// bob: 16:07:59.225082 [TRAC] block 'a28e88b8(3)' proposed by 'ana': start handling
		// bob: 16:07:59.225091 [TRAC] block 'a28e88b8(3)' proposed by 'ana' was verified and of the correct round: relaying
		// bob: 16:07:59.227533 [TRAC] block 'a28e88b8(3)' proposed by 'ana' was considered but is not the new higest ranking proposal
		// bob: 16:07:59.227627 [TRAC] block 'd012ae96(3)' proposed by 'bob': start handling
		// bob: 16:07:59.227636 [TRAC] block 'd012ae96(3)' proposed by 'bob' was verified and of the correct round: relaying
		// ana: 16:07:59.227887 [TRAC] block 'd012ae96(3)' proposed by 'bob' is new highest ranking block for next vote casting
		// ana: 16:07:59.227983 [TRAC] vote from 'ana' for block 'ebe9f65d(3)' proposed by 'ana': start handling
		// ana: 16:07:59.228047 [INFO] failed to verify vote from 'ana' for block 'ebe9f65d(3)' proposed by 'ana': invalid vote proof
		// ana: 16:07:59.228138 [TRAC] block '276fa2d2(3)' proposed by 'bob': start handling
		// ana: 16:07:59.228148 [TRAC] block '276fa2d2(3)' proposed by 'bob' was verified and of the correct round: relaying
		// bob: 16:07:59.230130 [TRAC] block 'd012ae96(3)' proposed by 'bob' is new highest ranking block for next vote casting
		// bob: 16:07:59.230224 [TRAC] vote from 'ana' for block 'ebe9f65d(3)' proposed by 'ana': start handling
		// bob: 16:07:59.230236 [INFO] failed to verify vote from 'ana' for block 'ebe9f65d(3)' proposed by 'ana': invalid vote proof
		// bob: 16:07:59.230318 [TRAC] block '276fa2d2(3)' proposed by 'bob': start handling
		// bob: 16:07:59.230329 [TRAC] block '276fa2d2(3)' proposed by 'bob' was verified and of the correct round: relaying
		// ana: 16:07:59.230759 [TRAC] block '276fa2d2(3)' proposed by 'bob' was considered but is not the new higest ranking proposal
		// ana: 16:07:59.230855 [TRAC] vote from 'ana' for block 'd012ae96(3)' proposed by 'bob': start handling
		// ana: 16:07:59.230868 [INFO] failed to verify vote from 'ana' for block 'd012ae96(3)' proposed by 'bob': invalid vote proof
		// ana: 16:07:59.231403 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// bob: 16:07:59.245346 [TRAC] block '276fa2d2(3)' proposed by 'bob' was considered but is not the new higest ranking proposal
		// bob: 16:07:59.245438 [TRAC] vote from 'ana' for block 'd012ae96(3)' proposed by 'bob': start handling
		// bob: 16:07:59.245498 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// ana: 16:07:59.245586 [TRAC] vote from 'bob' for block 'd012ae96(3)' proposed by 'bob': start handling
		// ana: 16:07:59.245599 [INFO] failed to verify vote from 'bob' for block 'd012ae96(3)' proposed by 'bob': invalid vote proof
		// bob: 16:07:59.245669 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
		// bob: 16:07:59.245682 [INFO] failed to verify vote from 'ana' for block 'd012ae96(3)' proposed by 'bob': invalid vote proof
		// bob: 16:07:59.245772 [TRAC] vote from 'bob' for block 'd012ae96(3)' proposed by 'bob': start handling
		// bob: 16:07:59.245784 [INFO] failed to verify vote from 'bob' for block 'd012ae96(3)' proposed by 'bob': invalid vote proof
		// -----16:08:00.107992

	}

	test.Assert(t, uint64(len(msgs)) > exprounds, "should do a decent amount of messages, got: %d", len(msgs))
}

// func Test2MemberDeadlock0minSize(t *testing.T) {
// 	netw := slot.NewMemNetwork()
// 	coll := collect(t, netw)
//
// 	//prep debug names and deterministic block input
// 	rnd1 := make([]byte, 32)
// 	rnd1[0] = 0x01
// 	rnd2 := make([]byte, 32)
// 	rnd2[0] = 0x02
// 	pk1, sk1, _ := vrf.GenerateKey(bytes.NewReader(rnd1)) //ana
// 	pk2, sk2, _ := vrf.GenerateKey(bytes.NewReader(rnd2)) //bob
// 	defer namePK(pk1, "eve")()
// 	defer namePK(pk2, "kim")()
// 	defer nameID("6d9c54dee566", "genesis")()
// 	defer nameID("20bb90498d6b", "b1")()
// 	defer nameID("cf71746031e0", "b2")()
//
// 	//member 1
// 	ep1 := netw.Endpoint()
// 	c1 := slot.NewChain()
// 	e1 := slot.NewEngine(os.Stderr, c1, pk1, sk1, ep1, time.Millisecond*5, 0)
// 	test.Ok(t, e1.WorkNewTip())
//
// 	//member 2
// 	ep2 := netw.Endpoint()
// 	c2 := slot.NewChain()
// 	e2 := slot.NewEngine(os.Stderr, c2, pk2, sk2, ep2, time.Millisecond*5, 0)
// 	test.Ok(t, e2.WorkNewTip())
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
//
// 	//should see a decent amount of messages if it doesn't deadlock
// 	msgs := <-coll()
// 	test.Assert(t, len(msgs) > 10, "should do a decent amount of messages, got: %d", len(msgs))
//
// 	test.Equals(t, c1.Tip(), c2.Tip())
// 	tipb := c1.Read(c1.Tip())
// 	fmt.Println("ROUND", tipb.Round)
// }
