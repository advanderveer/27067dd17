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
