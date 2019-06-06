package rev_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/rev"
	"github.com/advanderveer/go-test"
)

func TestEmptyProposalIgnore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	idn := rev.NewIdentity([]byte{0x01})
	logb := bytes.NewBuffer(nil)
	bc1 := rev.NewMemBroadcast()
	c1 := rev.NewChain(1)
	e1 := rev.NewEngine(logb, idn, bc1, c1)

	inj1 := rev.NewInjector([]byte{0x01})
	inj1.To(bc1)

	test.Ok(t, inj1.Write(&rev.Msg{}))
	time.Sleep(time.Millisecond)
	test.Equals(t, "[INFO] received message without a proposal, ignoring it\n", logb.String())

	test.Ok(t, e1.Shutdown(ctx))
}

func TestProposalHandlingValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	idn := rev.NewIdentity([]byte{0x02})
	bc1 := rev.NewMemBroadcast()
	c1 := rev.NewChain(1)
	e1 := rev.NewEngine(ioutil.Discard, idn, bc1, c1)

	inj1 := rev.NewInjector([]byte{0x01})
	inj1.To(bc1)
	bc1.To(inj1.MemBroadcast)

	//invalid proposal, genesis is invalid proposal
	//@TODO the invalid proposal is now marked as "handled", which makes the assertion
	//noted below work (by coincidence).
	test.Ok(t, inj1.Write(&rev.Msg{&rev.Proposal{}}))
	res := <-e1.Result()
	test.Equals(t, rev.ErrProposalTokenInvalid, res.ValidationErr)
	test.Equals(t, false, res.Relayed)

	//round too far in the future
	inj1.Propose(10, rev.B([]byte{0x01}, rev.NilID), e1.Genesis())
	res = <-e1.Result()
	test.Ok(t, res.ValidationErr)
	test.Equals(t, true, res.WitnessRoundTooFarOff)
	test.Equals(t, false, res.Relayed)

	// round 0 wraps to maxuint, which is too big to witness
	inj1.Propose(0, rev.B([]byte{0x01}, rev.NilID), e1.Genesis())
	res = <-e1.Result()
	test.Equals(t, true, res.WitnessRoundTooFarOff)
	test.Equals(t, false, res.Relayed)

	// invalid witness, doesn't exist @TODO this works by coincidence, the witness
	// should never not-exist because the out-of-order prevents us from serving those
	// normally
	inj1.Propose(1, rev.B([]byte{0x01}, rev.NilID), &rev.Proposal{})
	res = <-e1.Result()
	test.Equals(t, false, res.WitnessRoundTooFarOff)
	test.Equals(t, rev.ErrProposalWitnessUnknown, res.InvalidWitnessErr)
	test.Equals(t, false, res.Relayed)

	// valid proposal, start round 1 and relay
	gen := e1.Genesis()
	p1 := inj1.Propose(1, rev.B([]byte{0x01}, gen.Block.Hash()), gen)
	res = <-e1.Result()
	test.Ok(t, res.InvalidWitnessErr)
	test.Equals(t, true, res.OtherEnteredNewRound)
	test.Equals(t, true, res.Relayed)

	// check if we got our new proposal
	msg1 := <-inj1.Collect()
	_ = msg1
	// test.Equals(t, uint64(2), msg1.Proposal.Round) //@TODO re-enable

	// check if we got our relayed preoposal
	msg2 := <-inj1.Collect()
	test.Equals(t, p1, msg2.Proposal)

	test.Ok(t, e1.Shutdown(ctx))
}

func Test2MemberBabySteps(t *testing.T) {

	//algorithm security parameters
	witnessThreshold := 1
	numMembers := 12
	minLatency := time.Millisecond * 1
	maxLatency := time.Millisecond * 2
	to := time.Second * 5

	//context
	ctx := context.Background()
	// ctx, cancel := context.WithTimeout(context.Background(), maxLatency*time.Duration(numMembers))
	// defer cancel()

	//Mangling logs (untested)
	// - added Stringer method to identity
	// - added log line for entering a new round
	// - added log line for creation of proposal
	// - increased default broadcast buffer to 100
	// - added chain append and chain each
	// - added chain as a dependeny that is injected into the engine
	// - added graphing methods in engine test code
	// - added actual chain.Append call to engine handle
	// - added genesis block to chain on engine start
	// - added threshold function to return 1 for the genesis block
	// - added name functionality for memorable printing of identities (setName)
	// - added string function for proposal printing
	// - added 'newp' return parameter and do not relay if its not a new proposal
	// - added strength keeping to the chain
	// - added tip keeping logic to chain and use it when proposing
	// - commented out the check that the prev block needs to be proposed by a proposal in the previous round
	// - added log line that shows when we're stopping message handling due to EOF
	// - the tip we propose to continue on (tip) are taken from our current chain, rounds/proposals are just a vehicle
	// - chain intersection function for asserting consensus
	// - broadcast writing onto peer channel no happens in a different go-routine to prevent blocking
	// - broadcast to peer now happens with a random latency to that peer
	// - broadcast with latency can now configured optionally
	// - added a return value from the observe methods that states if the new proposal was a new top proposal

	// - (root) disabled the "the proposal that holds the prev block is not the top witness" check
	//   solution: do we check that we only add witnesses that are lower then our: current top?
	//   why does it work: sometimes.
	//   SOLUTION: ???
	// - (root) currently no longer chain consensus, need to pick prev block that has more sum weight
	//   instead of simply the higest scoring in the last round.
	//   SOLUTION: Added tip logic to chain and disable prev block check
	// - (root) numb of messages explodes past 2 members, do we relay too liberately. Solution: Do not
	//   relay if we've already seen the block?
	//   SOLUTION: only relay if we haven't seen the proposal
	// - (root) why are there 4 blocks per round?
	//   ANSWER: probably because members are allowed to send multiple per round if
	//   they encounter a higher ranking proposal after they did
	// - (root) why does it go slower with more members?
	//   SOLUTION: because writing to peers didn't happen concurrently

	// every member should be able to make a proposal in a round, independant of
	// what was observed in the previous round? A member can make a proposal for
	// an existing round if it has seen one for that round, or make a proposal for
	// a new round if has seen enough (solved the math puzzle?) of the previous round
	// - why would we want more proposals for a round if we already have one?

	// IDEA 1: Proof of votes counted
	// the math puzzle for a certain round should be easier to solve with more options
	// so everyone can have a max of N proposals combined to get proof to start the
	// next round. The difficulty D should average out how many votes anyone would
	// need to try in order to become a proposer.

	// PROBLEM:
	// what happens if someone finds this proof but he/she is the only one to do
	// so. Everyone moves over but the round ends up with too little proposals to
	// form the next round. But the one that managed to propose has now shown what
	// proposal can be combined for the threshold

	// PROBLEM 2
	// If the proof is dependant on the number of messages then it gets stuck due
	// to any filtering

	// Part 1: Round
	// - (1) one group can propose once they've encountered a new-round message with enough
	// proof. This number is configurable through the vrf random lottery method.

	// - (2) another group needs to prove that it has seen enough (valid) proposals by solving
	// a math puzzle that uses the proposals as input. The more proposals are seen the
	// easier it is too solve.

	// - (3) once any voter has created this proof it can broadcast it and everyone
	// moves to a new round and start at (1) again.

	// IDEA 2: How can someone propose

	// there are 2 pieces to proposing a block: solving the PoP puzzle for the previous
	// round and having a VRF token that passes a threshold.

	// proposers are drawn by a lottery ticket and build on a previous block. they
	// always propose on the strongest tip. If a new tip comes a along that is stronger
	// they will propose on that as well. In short: Propose and broadcast on every tip
	// that they experience.

	// combiners gather any nr of proposals together in any order such that it hashes
	// into a value past a certain theshold. The more proposals they see they more likely
	// it is that they solve the puzzle.

	//setup broadcasts
	var bcs []*rev.MemBroadcast
	inj := rev.NewInjector([]byte{0x00})
	for i := 0; i < numMembers; i++ {
		bc := rev.NewMemBroadcast()
		bc.WithLatency(minLatency, maxLatency)
		bcs = append(bcs, bc)
		inj.To(bc)
	}

	//setup identities
	var engines []*rev.Engine
	var chains []*rev.Chain
	for i := 0; i < numMembers; i++ {
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(i))

		idn := rev.NewIdentity(data)
		idn.SetName(fmt.Sprintf("e%d", i))

		c := rev.NewChain(witnessThreshold)
		e := rev.NewEngine(os.Stderr, idn, bcs[i], c)
		engines = append(engines, e)
		chains = append(chains, c)

		for _, bc := range bcs {
			if bc == bcs[i] {
				continue //skip itself
			}

			bcs[i].To(bc)
		}
	}

	//trigger chain reaction by injecting proposals
	gen := engines[0].Genesis()
	for i := 0; i < witnessThreshold; i++ {
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(i))
		inj.Propose(1, &rev.Block{Prev: gen.Block.Hash(), Data: data}, gen)
	}

	//wait for chain reaction
	time.Sleep(to)

	//shutdown all engines
	for _, e := range engines {
		test.Ok(t, e.Shutdown(ctx))
	}

	//draw individual chains
	for i, c := range chains {
		buf := bytes.NewBuffer(nil)
		draw(t, c, buf)
		drawPNG(t, buf, fmt.Sprintf("00_chain_%d.png", i))
	}

	//draw the chain intersection (consensus)
	inter, n, err := rev.IntersectChain(chains[0], chains[1:]...)
	test.Ok(t, err)
	_ = n
	// test.Assert(t, n > 50, "should have reached consensus on at least this many blocks")

	buf := bytes.NewBuffer(nil)
	draw(t, inter, buf)
	drawPNG(t, buf, fmt.Sprintf("00_census.png"))

}

// ----
// drawing utilities below
// ----

func drawPNG(t *testing.T, buf io.Reader, name string) {
	f, err := os.Create(name)
	test.Ok(t, err)
	defer f.Close()

	cmd := exec.Command("dot", "-Tpng")
	cmd.Stdin = buf
	cmd.Stdout = f
	test.Ok(t, cmd.Run())
}

func draw(t testing.TB, c *rev.Chain, w io.Writer) {
	fmt.Fprintln(w, `digraph {`)

	test.Ok(t, c.Each(func(id rev.ID, b *rev.Block) error {
		fmt.Fprintf(w, "\t"+`"%.6x" [shape=box,style="filled,solid",label="%.6x"`, id, id)

		fmt.Fprintf(w, `,fillcolor="#ffffff"`)

		fmt.Fprintf(w, "]\n")
		fmt.Fprintf(w, "\t"+`"%.6x" -> "%.6x";`+"\n", id, b.Prev)

		return nil
	}))

	fmt.Fprintln(w, `}`)
}
