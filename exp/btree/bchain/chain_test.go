package bchain_test

import (
	"math/big"
	"testing"

	"github.com/advanderveer/27067dd17/exp/btree/bchain"
	"github.com/advanderveer/27067dd17/exp/btree/bchain/bcdb"
	"github.com/advanderveer/go-test"
)

func testChain(t *testing.T) (c *bchain.Chain, clean func()) {
	genb := bchain.NewIdentity([]byte{0x01}, nil).SignBlock(0, &bchain.Block{}, [32]byte{})
	db := bcdb.MustTempBoltDB()
	c, err := bchain.NewChain(db, genb.Header, 1000)
	test.Ok(t, err)
	return c, func() {
		test.Ok(t, db.Close())
		test.Ok(t, db.Destroy())
	}
}

func TestEmptyChain(t *testing.T) {
	c, clean := testChain(t)
	defer clean()

	t.Run("should walk genesis just after creating empty chain", func(t *testing.T) {
		test.Ok(t, c.Walk(bchain.NilBID, func(bh *bchain.BlockHdr) error {
			test.Equals(t, "0-cdee7bd9", bh.ID.String())
			return nil
		}))
	})

	idn1 := bchain.NewIdentity([]byte{0x02}, nil)
	idn2 := bchain.NewIdentity([]byte{0x03}, nil)

	t.Run("should be able to mint after empty chain", func(t *testing.T) {
		b1, sign1, err := c.Mint(bchain.NilBID, 1, idn1)
		test.Ok(t, err)
		test.Equals(t, byte(0x00), b1.Header.Proof[0])
		b1 = sign1()
		test.Equals(t, byte(0x15), b1.Header.Proof[0])

		b2, sign2, err := c.Mint(bchain.NilBID, 1, idn2)
		sign2()
		test.Ok(t, err)

		t.Run("test that one token is bigger", func(t *testing.T) {
			ib1 := big.NewInt(0).SetBytes(b1.Header.Ticket.Token[:])
			ib2 := big.NewInt(0).SetBytes(b2.Header.Ticket.Token[:])
			test.Equals(t, 1, ib2.Cmp(ib1)) //ib2 > ib1
		})

		t.Run("should be able to appent minted block", func(t *testing.T) {
			test.Ok(t, c.Append(b2))
			test.Ok(t, c.Append(b1))

			t.Run("walk should now walk both blocks", func(t *testing.T) {
				var i int
				test.Ok(t, c.Walk(bchain.NilBID, func(bh *bchain.BlockHdr) error {
					if bh.ID.Round() == 1 {
						//round 1 should have elected the second block as becaus its ticket
						//ranks higher (ib2 > ib1)
						test.Equals(t, b2.Header, *bh)
					}

					i++
					return nil
				}))

				test.Equals(t, 2, i)
			})

		})
	})

	// test.Equals(t, nil, c.Mint())

	_ = c
}
