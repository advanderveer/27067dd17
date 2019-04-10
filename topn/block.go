package topn

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

//ID uniquely identifies a block
type ID [sha256.Size]byte

//Bytes returns the underlying bytes as a slice
func (id ID) Bytes() []byte { return id[:] }

func (id ID) String() string {
	return fmt.Sprintf("%d-%.4x", id.Round(), id[8:])
}

//Round returns the round encoded in the id itself
func (id ID) Round() uint64 {
	return binary.BigEndian.Uint64(id[:8])
}

//Tx encodes a transaction between two identities
type Tx struct {
	FromPK []byte //the public key of the identity that will have its balance subtracted
	ToPK   []byte //the public key of the identity that will have its balance increased
	Amount uint64 //the amount of currency that is transferred, must be <= than the users balance
}

// Hash the transaction
func (tx *Tx) Hash() [sha256.Size]byte {
	amountb := make([]byte, 8)
	binary.BigEndian.PutUint64(amountb, tx.Amount)
	return sha256.Sum256(bytes.Join([][]byte{
		amountb,
		tx.FromPK,
		tx.ToPK,
	}, nil))
}

//Block encodes the data and is chained to reach consensus over
type Block struct {
	Round uint64 //the round this block was proposed in
	Prev  ID     //the tip that member bets on to become the longest chain
	Token []byte //vrf token, seed is based on prev block
	Proof []byte //vrf proof, requires the secret key to create
	PK    PK     //vrf public key, paired to the secret key, can be used for verification

	Txs []*Tx //the transactions of currency between identities

	//@TODO (optimization) can we use something like bitcoins CTOR to not send txs
	//that are already in the peers mempool? https://www.youtube.com/watch?v=GYEZ52WVKEI&vl=en
}

// AppendTx will append transactions without checking if they were already added
func (b *Block) AppendTx(txs ...*Tx) {
	for _, tx := range txs {
		b.Txs = append(b.Txs, tx)
	}
}

// Hash the block returning an unique identifier
func (b *Block) Hash() (id ID) {

	//encode transaction hashes in the id
	var txhs [][]byte
	for _, tx := range b.Txs {
		txh := tx.Hash()
		txhs = append(txhs, txh[:])
	}

	//encode the round in the hash
	roundb := make([]byte, 8)
	binary.BigEndian.PutUint64(roundb, b.Round)

	id = ID(sha256.Sum256(bytes.Join([][]byte{
		roundb,
		b.Prev[:],
		b.Token,
		b.Proof,
		b.PK[:],
		bytes.Join(txhs, nil),
	}, nil)))

	//set first 8 bytes to round for efficient indexing
	//@TODO (security) reason if 24bits (32-8) of hash collision resistance is good enough
	copy(id[:], roundb)
	return
}

// String to a human readable short version of the block's identity
func (b *Block) String() string {
	id := b.Hash()
	return fmt.Sprint(id)
}

// Rank the block's in a round
func (b *Block) Rank(stake uint64) (rank *big.Int) {
	rank = big.NewInt(0).SetBytes(b.Token)
	rank.Mul(rank, big.NewInt(int64(stake)))
	return
}
