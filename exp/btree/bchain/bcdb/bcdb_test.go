package bcdb_test

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/exp/btree/bchain"
	"github.com/advanderveer/27067dd17/exp/btree/bchain/bcdb"
	"github.com/advanderveer/go-test"
)

func TestBoltDB(t *testing.T) {
	idn1 := bchain.NewIdentity([]byte{0x01}, nil)
	b1 := idn1.SignBlock(0, &bchain.Block{}, [32]byte{})

	//perform test for each implementation
	for _, c := range []struct{ db bchain.DB }{
		{db: bcdb.MustTempBoltDB()},
	} {
		t.Run(fmt.Sprintf("%T", c.db), func(t *testing.T) {
			t.Run("read non-existing block", func(t *testing.T) {
				test.Ok(t, c.db.View(func(tx bchain.Tx) (err error) {
					_, err = tx.ReadBlockHdr(bchain.BID{})
					test.Equals(t, bchain.ErrBlockNotExist, err)

					w1 := tx.ReadWeight(bchain.BID{})
					test.Equals(t, uint64(0), w1)
					return nil
				}))
			})

			t.Run("tip doesnt exist", func(t *testing.T) {
				test.Ok(t, c.db.View(func(tx bchain.Tx) (err error) {
					_, _, err = tx.ReadTip()
					test.Equals(t, bchain.ErrTipNotExist, err)
					return nil
				}))
			})

			t.Run("write a block", func(t *testing.T) {
				test.Ok(t, c.db.Update(func(tx bchain.Tx) (err error) {
					err = tx.WriteBlock(b1)
					test.Ok(t, err)

					err = tx.WriteWeight(b1.ID(), 10)
					test.Ok(t, err)

					bh1, err := tx.ReadBlockHdr(b1.Header.ID)
					test.Ok(t, err)

					w1 := tx.ReadWeight(b1.ID())
					test.Equals(t, b1.Header, *bh1)
					test.Equals(t, uint64(10), w1)

					test.Ok(t, tx.WriteTip(b1.ID(), 100))
					tip1, w2, err := tx.ReadTip()
					test.Ok(t, err)
					test.Equals(t, uint64(100), w2)
					test.Equals(t, b1.ID(), tip1)

					return
				}))
			})

			t.Run("view after write", func(t *testing.T) {
				test.Ok(t, c.db.View(func(tx bchain.Tx) (err error) {
					bh1, err := tx.ReadBlockHdr(b1.Header.ID)
					test.Ok(t, err)
					test.Equals(t, b1.Header, *bh1)
					w1 := tx.ReadWeight(b1.ID())
					test.Equals(t, uint64(10), w1)

					tip2, w3, err := tx.ReadTip()
					test.Ok(t, err)
					test.Equals(t, uint64(100), w3)
					test.Equals(t, b1.ID(), tip2)
					return
				}))
			})

			t.Run("interacting with a complete chain", func(t *testing.T) {
				height := uint64(100)
				width := 5
				rnd := rand.New(rand.NewSource(1))

				//make identities
				idns := make([]*bchain.Identity, width)
				for i := 0; i < width; i++ {
					idns[i] = bchain.NewIdentity([]byte(strconv.Itoa(i)), nil)
				}

				//init blocks
				t0 := time.Now()
				rounds := map[uint64][]*bchain.Block{}
				for i := uint64(1); i < height; i++ {
					rounds[i] = []*bchain.Block{}
					for j := 0; j < width; j++ {
						prev := b1
						if i > 1 {
							prev = rounds[i-1][rnd.Intn(len(rounds[i-1]))]
						}

						b := bchain.NewBlock(&prev.Header)
						b = idns[j].SignBlock(i, b, prev.Token())
						rounds[i] = append(rounds[i], b)
					}
				}
				fmt.Println("init:", time.Now().Sub(t0))

				//write blocks
				t1 := time.Now()
				test.Ok(t, c.db.Update(func(tx bchain.Tx) (err error) {
					for _, round := range rounds {
						for _, b := range round {
							test.Ok(t, tx.WriteBlock(b))
						}
					}

					return
				}))
				fmt.Println("writing:", time.Now().Sub(t1))

				t2 := time.Now()
				var i uint64
				test.Ok(t, c.db.View(func(tx bchain.Tx) (err error) {
					test.Ok(t, tx.EachBlock(0, math.MaxUint64, func(n uint64, hdrs []*bchain.BlockHdr) error {
						if n == 0 {
							test.Equals(t, 1, len(hdrs))
						} else {
							test.Equals(t, width, len(hdrs))
						}

						for _, hdr := range hdrs {
							test.Equals(t, n, hdr.ID.Round())
						}

						i++
						return nil
					}))

					return
				}))

				test.Equals(t, height, i)
				fmt.Println("reading:", time.Now().Sub(t2))

			})

			test.Ok(t, c.db.Close())   //close the database
			test.Ok(t, c.db.Destroy()) //destroy the database
		})
	}
}
