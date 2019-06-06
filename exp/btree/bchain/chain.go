package bchain

import (
	"math"
	"math/big"
	"sort"

	"github.com/pkg/errors"
)

// Chain implements a round-based block chain data structure
type Chain struct {
	points uint64
	gen    BlockHdr
	db     DB
}

// NewChain creates a blockchain
func NewChain(db DB, genesis BlockHdr, points uint64) (c *Chain, err error) {
	c = &Chain{points: points, gen: genesis, db: db}

	// setup genesis block
	return c, c.db.Update(func(tx Tx) error {
		_, err := tx.ReadBlockHdr(genesis.ID)
		if err == ErrBlockNotExist { //genesis isn't stored yet
			err = tx.WriteBlock(&Block{Header: genesis})
			if err != nil {
				return errors.Wrap(err, "failed to write genesis block")
			}

			err = tx.WriteTip(genesis.ID, points)
			if err != nil {
				return errors.Wrap(err, "failed to write genesis as tip")
			}

		} else if err != nil {
			return err //something else went wrong
		}

		return nil
	})
}

// Walk from the provided block towards the genesis
func (c *Chain) Walk(bid BID, f func(b *BlockHdr) error) (err error) {
	for {
		var b *BlockHdr
		if err = c.db.View(func(tx Tx) error {
			if bid == NilBID {
				bid, _, err = tx.ReadTip()
				if err != nil {
					return errors.Wrap(err, "failed to read tip for walk")
				}
			}

			b, err = tx.ReadBlockHdr(bid)
			if err != nil {
				return errors.Wrap(err, "failed to read block during walk")
			}

			return nil
		}); err != nil {
			return err
		}

		err = f(b)
		if err != nil {
			return err
		}

		if bid == c.gen.ID {
			break
		}

		bid = b.Prev
	}

	return
}

// Mint a new block with the provided prev, if the provided prev is a NilID the
// current tip is read and used as a tip.
func (c *Chain) Mint(prev BID, round uint64, idn *Identity) (b *Block, signf func() *Block, err error) {
	var prevh *BlockHdr
	if err = c.db.View(func(tx Tx) error {
		if prev == NilBID {
			prev, _, err = tx.ReadTip()
			if err != nil {
				return errors.Wrap(err, "failed to read tip")
			}
		}

		prevh, err = tx.ReadBlockHdr(prev)
		if err != nil {
			return errors.Wrap(err, "failed to read prev block")
		}

		// ------ Threshold ------
		// @TODO read and add our PoB rig with the proof of it
		// -----------------------

		return nil
	}); err != nil {
		return nil, nil, err
	}

	b = NewBlock(prevh)

	return b, func() *Block {
		return idn.SignBlock(round, b, prevh.Ticket.Token)
	}, nil
}

// Append a block to the chain if it is considered valid
func (c *Chain) Append(b *Block) (err error) {
	if ok := b.CheckSignature(); !ok {
		return ErrBlockSignatureInvalid
	}

	return c.db.Update(func(tx Tx) error {
		_, err := tx.ReadBlockHdr(b.ID())
		if err == nil {
			return ErrBlockExist
		}

		prev, err := tx.ReadBlockHdr(b.Header.Prev)
		if err != nil {
			return errors.Wrap(err, "failed to read prev block")
		}

		// @TODO check that the proposer didn't already propose a block for this round

		if ok, err := b.CheckAgainstPrev(prev); !ok {
			return errors.Wrap(err, "failed to verify block against prev")
		}

		// ------ Threshold ------
		// @TODO read the PoB rig ref that was encoded in a past block O(1)
		// @TODO with the rig, validate the ticket token
		// @TODO check if the token passes the threshold, if not. Discard
		// -----------------------

		// ------ Oracle ------
		// @TODO validate the writes to check if they do not conflict
		// @TODO (research) how do we keep oracle states efficiently
		//       - without keeping every version
		//       - without recalculating on every block
		// -----------------------

		err = tx.WriteBlock(b)
		if err != nil {
			return errors.Wrap(err, "failed to write block")
		}

		return c.longestChain(tx, b.Round())
	})
}

// longestChain implements the longest chain selection rule that we use. It is
// roughly basedon the DFinity algorithm. It ranks all the known blocks in each
// round and hands out weight proportional to the position in each round. The
// block with the must weight will be the tip.
func (c *Chain) longestChain(tx Tx, round uint64) (err error) {
	_, tipw, err := tx.ReadTip()
	if err != nil {
		return errors.Wrap(err, "failed to read tip")
	}

	return tx.EachBlock(0, math.MaxUint64, func(n uint64, hdrs []*BlockHdr) error {

		// sort blocks in the round by their ticket token's int value
		sort.Slice(hdrs, func(i, j int) bool {
			ii := big.NewInt(0).SetBytes(hdrs[i].Ticket.Token[:])
			ji := big.NewInt(0).SetBytes(hdrs[j].Ticket.Token[:])
			return ii.Cmp(ji) > 0
		})

		// now calculate sumweight by reading the prev's weight and
		// adding the weight from this round by the position in this round
		for i, hdr := range hdrs {
			w := c.points / uint64(i+1)
			prevw := tx.ReadWeight(hdr.Prev)

			// sum weight is prev weight + points divided by this block's rank
			sumw := prevw + w
			err = tx.WriteWeight(hdr.ID, sumw)
			if err != nil {
				return errors.Wrap(err, "failed to write new weight")
			}

			//if sum-weight heigher or equal the the current tip sum-weight use that
			//as the new tip. By also replacing on equal we prefer newly calculated
			//weights over the old maximum
			if sumw >= tipw {
				tipw = sumw

				err = tx.WriteTip(hdr.ID, sumw)
				if err != nil {
					return errors.Wrap(err, "failed to write new tip")
				}
			}
		}

		return nil
	})
}
