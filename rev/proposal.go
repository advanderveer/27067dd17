package rev

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sort"

	"github.com/advanderveer/27067dd17/vrf"
)

//PID uniquely identifies a proposal
type PID [sha256.Size]byte

//Proposal represents a member that tries to get the provided block
//included in everyones chain.
type Proposal struct {
	Round uint64
	Token []byte
	Proof []byte
	PK    []byte

	//below fields are not signed into the vrf
	Block   *Block
	Witness PIDSet
}

//NewProposal will setyp an empty proposal with a verifiable random token
func NewProposal(pk []byte, sk *[vrf.SecretKeySize]byte, round uint64) (p *Proposal) {
	p = &Proposal{Round: round, PK: pk}
	p.Token, p.Proof = vrf.Prove(seed(p.Round), sk)
	p.Witness = make(PIDSet)
	return
}

//Hash returns a unique fingerprint that represents the content
func (p *Proposal) Hash() (id PID) {
	r := make([]byte, 8)
	binary.LittleEndian.PutUint64(r, p.Round)
	var bid ID
	if p.Block != nil {
		bid = p.Block.Hash()
	}

	return PID(sha256.Sum256(bytes.Join([][]byte{
		r,
		p.Token,
		p.Proof,
		p.PK,
		bid[:],
		p.Witness.Hash(),
	}, nil)))
}

//PIDSet is a set of proposel IDs
type PIDSet map[PID]struct{}

//PSet creates a proposal id set
func PSet(ps ...PID) (s PIDSet) {
	s = make(PIDSet)
	s.Add(ps...)

	return
}

//Add the given proposal ids
func (ps PIDSet) Add(ids ...PID) {
	for _, id := range ids {
		ps[id] = struct{}{}
	}
}

//Sorted returns all proposal ids in the set sorted by byte order
func (ps PIDSet) Sorted() (pl []PID) {
	for pid := range ps {
		pl = append(pl, pid)
	}

	sort.Slice(pl, func(i, j int) bool {
		return bytes.Compare(pl[i][:], pl[j][:]) < 0
	})

	return
}

//Hash returns a checksum of the pid set by hasing the members in order
func (ps PIDSet) Hash() (sum []byte) {
	h := sha256.New()
	for _, pid := range ps.Sorted() {
		_, err := h.Write(pid[:])
		if err != nil {
			panic("failed to hash pid set: " + err.Error())
		}
	}

	return h.Sum(nil)
}

//HandlerFunc is a function that handles proposals
type HandlerFunc func(p *Proposal)

//Handle to implement handler
func (hf HandlerFunc) Handle(p *Proposal) { hf(p) }

//Handler handles proposals
type Handler interface {
	Handle(p *Proposal)
}

//seed produces a byte slice that we use as vrf seed
func seed(round uint64) (seed []byte) {
	seed = make([]byte, 8)
	binary.LittleEndian.PutUint64(seed, round)
	return
}
