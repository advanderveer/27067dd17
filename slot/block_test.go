package slot_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

func TestBlockCreation(t *testing.T) {
	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	seed := make([]byte, 32)
	ticket, proof := vrf.Prove(seed, sk)
	prev := slot.NilID
	prev[0] = 0x01

	b1 := slot.NewBlock(1, prev, ticket, proof, pk)
	test.Equals(t, uint64(1), b1.Round)
	test.Equals(t, pk, b1.PK[:])
	test.Equals(t, ticket, b1.Ticket[:])
	test.Equals(t, proof, b1.Proof[:])
	test.Equals(t, prev, b1.Prev)

	t.Run("invalid ticket", func(t *testing.T) {
		t.Run("too short", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, 1), make([]byte, slot.ProofSize), make([]byte, slot.PKSize))
		})

		t.Run("too long", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, slot.TicketSize+1), make([]byte, slot.ProofSize), make([]byte, slot.PKSize))
		})
	})

	t.Run("invalid proof", func(t *testing.T) {
		t.Run("too short", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, slot.TicketSize), make([]byte, 1), make([]byte, slot.PKSize))
		})

		t.Run("too long", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, slot.TicketSize), make([]byte, slot.ProofSize+1), make([]byte, slot.PKSize))
		})
	})

	t.Run("invalid pk", func(t *testing.T) {
		t.Run("too short", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, slot.TicketSize), make([]byte, slot.ProofSize), make([]byte, 1))
		})

		t.Run("too long", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()

			slot.NewBlock(1, slot.NilID, make([]byte, slot.TicketSize), make([]byte, slot.ProofSize), make([]byte, slot.PKSize+1))
		})
	})
}

func testBlock(t testing.TB) (b *slot.Block) {
	pk, sk, err := vrf.GenerateKey(bytes.NewReader(make([]byte, 33)))
	test.Ok(t, err)

	seed := make([]byte, 32)
	ticket, proof := vrf.Prove(seed, sk)
	prev := slot.NilID
	prev[0] = 0x01

	return slot.NewBlock(1, prev, ticket, proof, pk)
}

type errwriter struct {
	E error
}

func (ew errwriter) Write(p []byte) (n int, err error) {
	return 0, ew.E
}

func TestEncoding(t *testing.T) {
	b1 := testBlock(t)
	buf1 := bytes.NewBuffer(nil)
	err := b1.Encode(buf1)
	test.Ok(t, err)
	test.Equals(t, slot.IDSize+slot.TicketSize+slot.ProofSize+slot.PKSize+8+slot.TicketSize+slot.ProofSize+slot.PKSize, buf1.Len())

	b2, err := slot.DecodeBlock(buf1)
	test.Ok(t, err)
	test.Equals(t, b1, b2)

	t.Run("test encoding error", func(t *testing.T) {
		ew := &errwriter{E: fmt.Errorf("foo")}
		err = b1.Encode(ew)
		test.Assert(t, strings.Contains(err.Error(), "foo"), "should show error")
	})

	t.Run("test EOF decoding", func(t *testing.T) {
		buf2 := bytes.NewReader(nil)
		_, err = slot.DecodeBlock(buf2)
		test.Assert(t, strings.Contains(err.Error(), "EOF"), "should show error")
	})
}

type errhash struct {
	E error
	hash.Hash
}

func (ew errhash) Write(p []byte) (n int, err error) {
	return 0, ew.E
}

func TestHashing(t *testing.T) {
	b := testBlock(t)
	h1 := b.Hash()
	test.Equals(t, "9d875bef7b29f62be5c44aafe8c0e9c9d0d17911241d04b24833ad03374574c0", hex.EncodeToString(h1[:]))

	oldf := slot.NewBlockHash
	t.Run("hash error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()

		slot.NewBlockHash = func() hash.Hash { return &errhash{fmt.Errorf("foo"), sha256.New()} }
		b.Hash()
	})

	//reset hash function
	slot.NewBlockHash = oldf

	//any field mutation should give a unique hash
	ids := make(map[slot.ID]struct{})
	mutations := []func(){
		func() { b.Round = 2 },
		func() { b.Prev[0] = 0x02 },
		func() { b.Ticket[0] = 0x02 },
		func() { b.Proof[0] = 0x02 },
		func() { b.PK[0] = 0x02 },
	}
	for i, mut := range mutations {
		mut()          //mutate
		id := b.Hash() //hash
		if _, ok := ids[id]; ok {
			t.Fatalf("mutation %d should give unique id", i)
		}

		ids[id] = struct{}{}
	}

	test.Equals(t, len(mutations), len(ids))
}

func TestRanking(t *testing.T) {
	t1 := slot.NilTicket                // 0
	t2 := make([]byte, slot.TicketSize) // higher then 0
	t2[0] = 0x01                        //should now outrank

	b1 := slot.NewBlock(1, slot.NilID, t1, slot.NilProof, slot.NilPK)
	b2 := slot.NewBlock(1, slot.NilID, t2, slot.NilProof, slot.NilPK)

	r2 := []*slot.Block{b1, b2}
	slot.Rank(r2)
	test.Equals(t, []*slot.Block{b2, b1}, r2) //expect b2 to be frist
}
