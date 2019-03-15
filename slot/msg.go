package slot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math/big"
	"sort"
)

//MsgType tags messages
type MsgType uint

const (
	//MsgTypeUnkown is a message that is unkown
	MsgTypeUnkown MsgType = iota

	//MsgTypeVote is a block voting message
	MsgTypeVote

	//MsgTypeProposal is a block proposal message
	MsgTypeProposal
)

//Msg holds messages holds passed around between members
type Msg struct {
	Proposal *Block
	Vote     *Vote
}

//Type returns the message type
func (m *Msg) Type() MsgType {
	switch true {
	case m.Proposal != nil:
		return MsgTypeProposal
	case m.Vote != nil:
		return MsgTypeVote
	default:
		return MsgTypeUnkown
	}
}

func (m *Msg) String() string {
	switch m.Type() {
	case MsgTypeProposal:
		return fmt.Sprintf("P: block:%s prev:%s proposer:%s", BlockName(m.Proposal.Hash()), BlockName(m.Proposal.Prev), PKString(m.Proposal.PK[:]))
	case MsgTypeVote:
		return fmt.Sprintf("V: block:%s prev:%s proposer:%s voter:%s", BlockName(m.Vote.Block.Hash()), BlockName(m.Vote.Block.Prev), PKString(m.Vote.Block.PK[:]), PKString(m.Vote.VotePK[:]))
	default:
		panic("not implemented")
	}
}

const (
	//IDSize is the size of a block ID
	IDSize = 32

	//TicketSize is the size of a block ticket
	TicketSize = 32

	//ProofSize is the size of ticket proof
	ProofSize = 96

	//PKSize is the size of the ticket public key
	PKSize = 32
)

//ID uniquly identifies a block
type ID [IDSize]byte

var (
	//NilID is an empty ID
	NilID = ID{}

	//NilTicket is an empty ticket
	NilTicket = make([]byte, TicketSize)

	//NilProof is an empty ID
	NilProof = make([]byte, ProofSize)

	//NilPK is an empty ID
	NilPK = make([]byte, PKSize)
)

//NewBlockHash is used for any hash creation
var NewBlockHash = func() hash.Hash { return sha256.New() }

//Rank a slice of blocks according to their ticket
func Rank(blocks []*Block) {
	sort.Slice(blocks, func(i, j int) bool {
		ii := big.NewInt(0)
		ii.SetBytes(blocks[i].Ticket[:])

		ji := big.NewInt(0)
		ji.SetBytes(blocks[j].Ticket[:])
		return ii.Cmp(ji) > 0
	})
}

//Vote holds a endorsement for a block, signed by a voter and comes with
//a copy of the block itself
type Vote struct {
	*Block

	//@TODO think about whether the block needs to be part of some signature to
	//validate the vote?
	VoteTicket [TicketSize]byte
	VoteProof  [ProofSize]byte
	VotePK     [PKSize]byte
}

//Hash the vote
func (v *Vote) Hash() (id ID) { panic("not implemented") }

//BlockHash hashes the block contained in the vote
func (v *Vote) BlockHash() (id ID) {
	return v.Block.Hash()
}

// String the vote into something human readable
func (v *Vote) String() string {
	return fmt.Sprintf("vote from '%s' for %s", PKString(v.VotePK[:]), v.Block.String())
}

//Block holds the data the algorithm is trying to reach consensus over.
type Block struct {
	Round  uint64
	Prev   ID
	Ticket [TicketSize]byte
	Proof  [ProofSize]byte
	PK     [PKSize]byte
}

//NewBlock will allocate a fixed size block
func NewBlock(round uint64, prev ID, ticket, proof, pk []byte) (b *Block) {
	b = &Block{Round: round, Prev: prev}
	n := copy(b.Ticket[:], ticket)
	if n != len(ticket) || n != TicketSize {
		panic("unexpected ticket size")
	}

	n = copy(b.Proof[:], proof)
	if n != len(proof) || n != ProofSize {
		panic("unexpected proof size")
	}

	n = copy(b.PK[:], pk)
	if n != len(pk) || n != PKSize {
		panic("unexpected public key size")
	}

	return
}

// DecodeBlock decodes a block from reader 'r' and return it
func DecodeBlock(r io.Reader) (b *Block, err error) {
	b = &Block{}
	for _, v := range []interface{}{
		&b.Round,
		&b.Prev,
		&b.Ticket,
		&b.Proof,
		&b.PK,
	} {
		err := binary.Read(r, binary.LittleEndian, v)
		if err != nil {
			return nil, fmt.Errorf("failed to read binary data: %v", err)
		}
	}

	return
}

// String the block into something human readable
func (b *Block) String() string {
	return fmt.Sprintf("block '%s(%d)' proposed by '%s'", BlockName(b.Hash()), b.Round, PKString(b.PK[:]))
}

// Encode the block to the provide writer
func (b *Block) Encode(w io.Writer) (err error) {
	for _, v := range []interface{}{
		b.Round,
		b.Prev,
		b.Ticket,
		b.Proof,
		b.PK,
	} {
		err := binary.Write(w, binary.LittleEndian, v)
		if err != nil {

			// bizarly, this panics sometimes while running the engine test:
			// panic: failed to encode block: failed to write binary data: foo
			//
			// goroutine 65 [running]:
			// github.com/advanderveer/27067dd17/slot.(*Block).Hash(0xc000264000, 0x0, 0x0, 0x0, 0x0)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/msg.go:183 +0x18d
			// github.com/advanderveer/27067dd17/slot.(*Chain).Tally(0xc0000801e0, 0xc0002c6000, 0x0)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/chain.go:64 +0x94
			// github.com/advanderveer/27067dd17/slot.(*Engine).HandleVote(0xc000306540, 0xc0002c6000, 0x12034c0, 0xc000303b60, 0xc000168658, 0xc00027fe01)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/engine.go:138 +0xd8
			// github.com/advanderveer/27067dd17/slot.(*Engine).Handle(0xc000306540, 0xc00016ae80, 0x12034c0, 0xc000303b60, 0x91dbf802cc5aea8e, 0xaaf14237d9832616)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/engine.go:94 +0x117
			// github.com/advanderveer/27067dd17/slot.(*OutOfOrder).Handle(0xc000271a00, 0xc00016ae80, 0x0, 0x0)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/ooo.go:96 +0x32d
			// github.com/advanderveer/27067dd17/slot.(*Engine).Run(0xc000306540, 0x0, 0x0)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/engine.go:83 +0xcc
			// github.com/advanderveer/27067dd17/slot_test.Test2MemberSeveralRounds.func1(0xc00030c000, 0xc000306540)
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/engine_test.go:256 +0x2f
			// created by github.com/advanderveer/27067dd17/slot_test.Test2MemberSeveralRounds
			// 	/Users/adam/Projects/go/src/github.com/advanderveer/27067dd17/slot/engine_test.go:255 +0x33a
			// exit status 2
			// FAIL	github.com/advanderveer/27067dd17/slot	0.664s

			return fmt.Errorf("failed to write binary data: %v", err)
		}
	}

	return
}

// Hash the block bytes and return it as a Block's ID
func (b *Block) Hash() (id ID) {
	h := NewBlockHash()
	err := b.Encode(h)
	if err != nil {
		panic("failed to encode block: " + err.Error())
	}

	copy(id[:], h.Sum(nil))
	return
}

// Strength is the blocks ticket expressed as a rational devided by its rank
func (b *Block) Strength(rank int) (s *big.Rat) {
	ss := new(big.Int)
	ss.SetBytes(b.Ticket[:])

	//@the strength becomes a rational value of a blocks rank in the round
	//rank 1 = 1/1, rank 2 = 1/2, rank 3 is 1/3 etc. @TODO difinity uses a
	//another formulate to rank, check if that is needed
	s = new(big.Rat)
	s.SetFrac(ss, big.NewInt(int64(rank)))
	return
}
