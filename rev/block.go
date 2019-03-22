package rev

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

//Block describes the information that is encoded in the chain
type Block struct {
	Prev ID
	Data []byte
}

//B creates a new block
func B(d []byte, prev ID) *Block {
	return &Block{prev, d}
}

//Hash the content of the block into a unique identifier
func (b *Block) Hash() (id ID) {
	return ID(sha256.Sum256(bytes.Join([][]byte{
		b.Prev[:],
		b.Data,
	}, nil)))
}
