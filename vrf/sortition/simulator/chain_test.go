package simulator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

func TestBlockChainReading(t *testing.T) {
	bc1, err := NewBlockChain()
	test.Ok(t, err)

	gen, err := bc1.Read(bc1.Genesis())
	test.Ok(t, err)
	test.Equals(t, int64(0), gen.Prio().Int64())

	_, err = bc1.Read(NilID)
	test.Equals(t, ErrBlockNotExist, err)
}

func TestBlockChainWalking(t *testing.T) {
	bc1, err := NewBlockChain()
	test.Ok(t, err)

	test.Ok(t, bc1.Walk(bc1.Tip(), func(b *Block) error {
		test.OkEquals(t, bc1.Genesis())(b.Hash())
		return nil
	}))

	err1 := errors.New("foo")
	test.Equals(t, err1, bc1.Walk(bc1.Tip(), func(b *Block) error {
		return err1
	}))

	test.Equals(t, ErrBlockNotExist, bc1.Walk(NilID, func(b *Block) error {
		return err1
	}))
}

func TestBlockVerification(t *testing.T) {
	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	bc, err := NewBlockChain()
	test.Ok(t, err)

	b1, err := bc.Create(sk, bc.Genesis(), 100)
	test.Ok(t, err)

	test.Equals(t, uint64(76), b1.widx)

	ok, err := bc.Verify(pk, 100, b1)
	test.Ok(t, err)
	test.Equals(t, true, ok)

	t.Run("widx too large", func(t *testing.T) {
		ok, err := bc.Verify(pk, 76, b1)
		test.Assert(t, err != nil, "verify should show error")
		test.Equals(t, false, ok)
	})

	t.Run("invalid prio", func(t *testing.T) {
		ok, err := bc.Verify(nil, 10, b1)
		test.Assert(t, err != nil, "verify should show error")
		test.Equals(t, false, ok)

		b1.prev = NilID
		ok, err = bc.Verify(pk, 10, b1)
		test.Assert(t, err != nil, "verify should show error")
		test.Equals(t, false, ok)
	})

	//proof that you have seen N different blocks in the previous round before
	//you can create a block for the new round

	//Problems:
	// - Nothing at Stake 1: every miner will draw for all tips and just broadcast
	//   a block for all of them: not actually choosing a tip
	// - Nothing at Stake 2: miner will draw many blocks for each of their weight
	//   indexes and broadcast a block for each: ddos the network
	//Solution:
	// - Some sort of round/notarization system that windows input from each
	//   miner: discrete
	//   - we introduce height as first class concept:
	//     - Every miner is only allowed 1 block per height. If it decides on a
	//       height too early it may be wasting its ticket: it draws for all tips
	//       and only sends the highest priority drawn?
	//     - Each height has a time window that closes after a certain time
	//  Proof that you have seen N blocks in the previous round

	//@TODO can we verify prio with a snark?
	//@TODO can we introduce finality by asking each proposer to also "notarize"
	//block that is not from itself? What happens with a relay policy that buffers
	//block proposals and only sends the highest one in a blockTime since its first
	//arrival? How to prevent advisaries from not doing this? Vector clock? Show proof
	//of having seen N block proposals (that are lower) for the same block from?
	//@TODO can we make it soo that a user only needs to send a proof of its weight

	//1. Weight based drawing of priorities using VRF, clients draw on the tip with the most strength
	//2. Relay policy that filters low priority blocks that do not result in a new tip to reduce tips (but is not consensus)
	//3. Add a threshold function based on the mean priority in the last N blocks and makes it economical
	//   to wait

	//1. Weight based drawing with prio based PoW
	//2. Need to wait on N lower prio blocks
	//3. Filter lower prio block on relay

	//1. Weight based drawing of prio
	//2. Every block needs to wait on N higher prio blocks

	// wait for majority-relays

	//Lesson: we need some round/rythm setup. PoW is a clock. How can we add a
	//delay window.

	//   @TODO what prevents extreme forking on the N highest priority blocks?

	//@TODO the threshold is instead a wait time after which it becomes economical to start
	//a ProofOfWork? This threshold is based on the mean prioity of the last N blocks
	//@TODO what if no-one draws below the threshold? Make that chance very small?
	//i.e the change of the network not drawing should be multiplied by the block
	//reward of mining. That becomes economical if the network indeed didn't manage to
	//produce a block

	//3. Priority determines on how many higher priority block we need to wait()

	//3. A block is only valid if it include N lower priority blocks

	//Show that is more economical to wait for the next round

	//BONUS: Curved PoW that prevents low priority blocks from being economical to broadcast
	//@TODO proof that this is the case. How does this provide finality?

	//Does filtering prevent waiting for others for a round?

}

func TestBlockChainAppending(t *testing.T) {
	_, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 32)))
	test.Ok(t, err)

	bc, err := NewBlockChain()
	test.Ok(t, err)

	test.Equals(t, bc.Genesis(), bc.Tip())

	b1, err := bc.Create(sk, bc.Genesis(), 10)
	test.Ok(t, err)

	h1, err := b1.Hash()
	test.Ok(t, err)

	_, err = bc.Read(h1)
	test.Equals(t, ErrBlockNotExist, err)

	id1, err := bc.Append(b1)
	test.Ok(t, err)
	test.Equals(t, h1, id1)

	test.Equals(t, id1, bc.Tip())

	b11, err := bc.Read(h1)
	test.Ok(t, err)
	test.Equals(t, b1, b11)

	var visited []ID
	test.Ok(t, bc.Walk(h1, func(b *Block) error {
		h, _ := b.Hash()
		visited = append(visited, h)
		return nil
	}))

	test.Equals(t, []ID{h1, bc.Genesis()}, visited)
}
