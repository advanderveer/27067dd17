package rev_test

import (
	"bytes"
	"context"
	"io/ioutil"
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
	e1 := rev.NewEngine(logb, idn, bc1, 1)

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
	e1 := rev.NewEngine(ioutil.Discard, idn, bc1, 1)

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
	test.Equals(t, uint64(2), msg1.Proposal.Round)

	// check if we got our relayed preoposal
	msg2 := <-inj1.Collect()
	test.Equals(t, p1, msg2.Proposal)

	test.Ok(t, e1.Shutdown(ctx))
}
