package tt

import (
	"bytes"
	"crypto/sha256"
)

//ID uniquely identifies a block
type ID [sha256.Size]byte

var (
	//NilID is an empty ID
	NilID = ID{}
)

//Block contains the data and can be submitted by the fist member to solve the
//vote puzzle and encodes the data the protocol reach consensus over
type Block struct {
	Prev ID
	Data []byte
}

//B creates a new block
func B(prev ID, d []byte) *Block {
	return &Block{prev, d}
}

//Hash the content of the block into a unique identifier
func (b *Block) Hash() (id ID) {
	return ID(sha256.Sum256(bytes.Join([][]byte{
		b.Prev[:],
		b.Data,
	}, nil)))
}
