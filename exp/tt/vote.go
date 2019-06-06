package tt

import (
	"bytes"
	"crypto/sha256"
)

//VID uniquely identifies a vote
type VID [sha256.Size]byte

var (
	//NilVID is an empty vote ID
	NilVID = VID{}
)

//Vote is a small fixed-sized messages that saturate the network with hints to the
//longest chain.
type Vote struct {
	Token []byte
	Proof []byte
	PK    []byte

	Tip ID
}

//Hash the content of the block into a unique identifier
func (v *Vote) Hash() (id VID) {
	return VID(sha256.Sum256(bytes.Join([][]byte{
		v.Token,
		v.Proof,
		v.PK,
		v.Tip[:],
	}, nil)))
}
