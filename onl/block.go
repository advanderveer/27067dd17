package onl

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//IDLen is the id length
const IDLen = sha256.Size

//PK is a fixed-size public key identity
type PK [32]byte

//NilID is an empty id
var NilID = ID{}

//ID of a block is determined by hashing it
type ID [IDLen]byte

func (id ID) String() string {
	return fmt.Sprintf("%.4x-%d", id[8:], id.Round())
}

//Bytes returns the underlying bytes as a slice
func (id ID) Bytes() []byte { return id[:] }

//Round number that is encoded in the ID
func (id ID) Round() uint64 {
	return math.MaxUint64 - binary.BigEndian.Uint64(id[:])
}

//Block holds the data that is send between members to reach consensus. Each
//block is assigned to a round based on its timestamp, ranked using a VRF token
//and linked together to form a chain.
type Block struct {

	// The Round at which the proposer has chosen to placed this block.
	Round uint64

	// Microseconds since unix epoch at which this block was made. It was measured
	// (or chosen) on the untrused clock of the proposer's. We hope that microsecond
	// precision will be usefull once network and clock syncing technology become better
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

	//The writes that this identity proposes to the network in this order.
	//other identities will vote on this block by referencing it and strenghen
	//the networks believe.
	Writes []*Write
}

// Hash the block returning an unique identifier
func (b *Block) Hash() (id ID) {
	tsb := make([]byte, 8)
	binary.BigEndian.PutUint64(tsb, b.Timestamp)
	roundb := make([]byte, 8)
	binary.BigEndian.PutUint64(roundb, math.MaxUint64-b.Round)

	//encode transaction hashes in the id
	var wrshs [][]byte
	for _, wr := range b.Writes {
		wrsh := wr.Hash()
		wrshs = append(wrshs, wrsh[:])
	}

	//hash the fields and the ops
	id = ID(sha256.Sum256(bytes.Join([][]byte{
		b.FinalizedPrev.Bytes(),
		b.Prev.Bytes(),
		b.PK[:],
		b.Proof,
		b.Token,
		tsb,
		roundb,
		bytes.Join(wrshs, nil),
	}, nil)))

	//prefix the ID with the round, allows round based sorting in the store
	//@TODO (security) what does this the collission resistance 4,7e21 do to security
	copy(id[:8], roundb)
	return
}

//Seed returns the input for the verifiable random token. The token (and the thus
//the blocks ranking) is dependant on this seed.
func (b *Block) Seed() []byte {
	seed := b.FinalizedPrev[:]      //brings long-term uncertainty about a pk's worth
	seed = append(seed, b.PK[:]...) //the pk of the proposer

	roundb := make([]byte, 8) //round nr as the epoc dividd by the round time
	binary.BigEndian.PutUint64(roundb, b.Round)
	seed = append(seed, roundb...)
	return seed
}

//VerifySignature will check the block's signature
func (b *Block) VerifySignature() (ok bool) {
	pk := [32]byte(b.PK)
	return ed25519.Verify(&pk, b.Hash().Bytes(), &b.Signature)
}

//VerifyToken will verify the random function token
func (b *Block) VerifyToken(tokenPK []byte) (ok bool) {
	return vrf.Verify(tokenPK, b.Seed(), b.Token, b.Proof)
}

//Rank returns the block ranking as a function of the identities stake
func (b *Block) Rank(stake uint64) (rank *big.Int) {
	rank = big.NewInt(0).SetBytes(b.Token)
	rank.Mul(rank, big.NewInt(int64(stake)))
	return
}

//AppendWrite will append operations the the block it doesn't check if for duplicates
func (b *Block) AppendWrite(ws ...*Write) {
	for _, w := range ws {
		if w == nil {
			continue
		}

		b.Writes = append(b.Writes, w)
	}
}
