package onl

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/onl/ssi"
)

//WID uniquely identifies a write
type WID [sha256.Size]byte

// Write encodes signed modifications to the keyspace that will be replicated
// on each member through the consensus protocol
type Write struct {
	*ssi.TxData
	//@TODO add signature data: PK, Signature
	//@TODO add signer pk to hash
}

//Hash the operation
func (op *Write) Hash() (id WID) {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, op.TimeStart)
	binary.Write(h, binary.BigEndian, op.TimeCommit)

	for k := range op.ReadRows {
		binary.Write(h, binary.BigEndian, k[:])
	}

	for kh, kv := range op.WriteRows {
		binary.Write(h, binary.BigEndian, kh[:])
		binary.Write(h, binary.BigEndian, kv.K)
		binary.Write(h, binary.BigEndian, kv.V)
	}

	copy(id[:], h.Sum(nil))
	return
}
