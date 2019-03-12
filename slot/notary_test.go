package slot_test

import (
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	test "github.com/advanderveer/go-test"
)

type blocks map[slot.ID]*slot.Block

func (r blocks) Read(id slot.ID) (b *slot.Block) {
	b = r[id]
	return
}

func TestNotarization(t *testing.T) {
	t1 := make([]byte, slot.TicketSize)
	t1[0] = 0x01
	p1 := make([]byte, slot.ProofSize)
	p1[0] = 0x01
	pk1 := make([]byte, slot.PKSize)
	pk1[0] = 0x01

	r1 := blocks{}
	n1 := slot.NewNotary(1, r1, slot.Ticket{Data: t1, Proof: p1}, pk1)
	b1 := slot.NewBlock(2, slot.NilID, ticketS1[:], slot.NilProof, slot.NilPK)
	b2 := slot.NewBlock(3, slot.NilID, ticketS2[:], slot.NilProof, slot.NilPK)
	b3 := slot.NewBlock(4, slot.NilID, ticketS2[:], slot.NilProof, slot.NilPK)

	ok, n := n1.Propose(b1)
	test.Equals(t, true, ok)
	test.Equals(t, 1, n)

	ok, n = n1.Propose(b1)
	test.Equals(t, false, ok) //nothing changed
	test.Equals(t, 1, n)

	ok, n = n1.Propose(b2)
	test.Equals(t, true, ok) //should replace proposal
	test.Equals(t, 1, n)

	ok, n = n1.Propose(b3)
	test.Equals(t, true, ok) //should augment proposal with
	test.Equals(t, 2, n)

	ok, n = n1.Propose(b2)
	test.Equals(t, false, ok) //should do nothing
	test.Equals(t, 2, n)

	nots := n1.Notarize()
	for _, not := range nots {
		test.Equals(t, t1, not.NtTicket[:])
		test.Equals(t, p1, not.NtProof[:])
		test.Equals(t, pk1, not.NtPK[:])
	}
}
