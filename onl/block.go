package onl

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//PK is a fixed-size public key identity
type PK [32]byte

//ID of a block is determined by hashing it
type ID [sha256.Size]byte

//Bytes returns the underlying bytes as a slice
func (id ID) Bytes() []byte { return id[:] }

//Block holds the data that is send between members to reach consensus. Each
//block is assigned to a round based on its timestamp, ranked using a VRF token
//and linked together to form a chain.
type Block struct {

	// Microseconds since unix epoch at which this block was made, measured on
	// the (untrused) proposer's clock. Using a modulus operation this timestamp
	// is used to determine in which round the block will be observed. We hope that
	// microsecond precision will be usefull once network and clock syncing technology
	// becomes better
	Timestamp uint64

	// The verifiable random token that determines (together with stake)
	// the ranking of the block in each round. Higher ranking blocks are more likely
	// to be included in the longest chain and the proposer will be rewarded.
	Token []byte

	// Proof that the token is indeed valid and represents a random value specific
	// to this identity produced with the correct input (seed).
	Proof []byte

	// The primary key that identifies the identity that proposed this block. It
	// must have previously announced that it wanted to participate in the protocol
	// and put up stake as a security deposity while committing to a token pk. The
	// size of this stake determines (together with the drawn token) the ranking of this block
	PK PK

	// The previous block that this block 'votes' on. It formes a chain of blocks
	// and stake from the proposer of this blocks "flows" through it to
	// anncestors to figure out if a majority of stake agreed on a certain chain.
	// If members observe a majority stake voting (indirectly) on a block it can
	// be finalized and transactions in it will not be reverted.
	Prev ID

	// A stable prev function where the majority stake has invested in. It used to
	// make the token unpredictable such that it cannot be precalculated to mine
	// very valuable VRF keys.
	FinalizedPrev ID

	//Signature of the block, signed by the identity of PK such that it can be
	//used to that the block's data hasn't been tampered with.
	Signature [ed25519.SignatureSize]byte

	//The operations that this identity proposes to the network, in this order.
	//other identities will vote on this block by referencing it and strenghen
	//the networks believe in the ops being correct in this order.
	Ops []Op
}

// Hash the block returning an unique identifier
func (b *Block) Hash() (id ID) {
	tsb := make([]byte, 8)
	binary.BigEndian.PutUint64(tsb, b.Timestamp)

	//encode transaction hashes in the id
	var opshs [][]byte
	for _, ops := range b.Ops {
		opsh := ops.Hash()
		opshs = append(opshs, opsh[:])
	}

	id = ID(sha256.Sum256(bytes.Join([][]byte{
		b.FinalizedPrev.Bytes(),
		b.Prev.Bytes(),
		b.PK[:],
		b.Proof,
		b.Token,
		tsb,
		bytes.Join(opshs, nil),
	}, nil)))

	return
}

//VRFSeed returns the input for the verifiable random token
func (b *Block) VRFSeed(rt uint64) []byte {
	seed := b.FinalizedPrev[:]      //brings long-term uncertainty about a pk's worth
	seed = append(seed, b.PK[:]...) //the pk that the proposer must have committed to
	roundb := make([]byte, 8)       //round nr as the epoc dividd by the round time

	binary.BigEndian.PutUint64(roundb, b.Timestamp/rt)
	seed = append(seed, roundb...)
	return seed
}

//VerifyCrypto will verify the cryptographic elements of the block. The TokenPK is
//not included in the message but must have been pre-committed when the stake deposit
//was placed.
func (b *Block) VerifyCrypto(tokenPK []byte, rt uint64) (ok bool, err error) {
	pk := [32]byte(b.PK)
	ok = ed25519.Verify(&pk, b.Hash().Bytes(), &b.Signature)
	if !ok {
		return false, ErrInvalidSignature
	}

	ok = vrf.Verify(tokenPK, b.VRFSeed(rt), b.Token, b.Proof)
	if !ok {
		return false, ErrInvalidToken
	}

	return true, nil
}

//AppendOps will append operations the the block it doesn't check if for duplicates
func (b *Block) AppendOps(ops ...Op) {
	for _, op := range ops {
		b.Ops = append(b.Ops, op)
	}
}
