package onl

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sort"

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

// HasDepositFor returns whether this write writes a deposit for the provided
// identity. This is used to find a random seed for the VRF
func (op *Write) HasDepositFor(pk PK) bool {
	var (
		wroteTPK   bool
		wroteStake bool
	)

	for _, wr := range op.TxData.WriteRows {
		if bytes.Equal(wr.K, tpkey(pk)) {
			wroteTPK = true
			continue
		}

		if bytes.Equal(wr.K, skey(pk)) {
			wroteStake = true
			continue
		}
	}

	if wroteTPK && wroteStake {
		return true
	}

	return false
}

//Hash the operation
func (op *Write) Hash() (id WID) {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, op.TimeStart)
	binary.Write(h, binary.BigEndian, op.TimeCommit)

	//read rows, sorted
	rr := make([]ssi.KH, 0, len(op.ReadRows))
	for k := range op.ReadRows {
		rr = append(rr, k)
	}

	sort.Slice(rr, func(i, j int) bool {
		return bytes.Compare(rr[i][:], rr[j][:]) < 0
	})

	for _, k := range rr {
		binary.Write(h, binary.BigEndian, k[:])
	}

	//Write rows, sorted
	wr := make([]ssi.KH, 0, len(op.WriteRows))
	for k := range op.WriteRows {
		wr = append(wr, k)
	}

	sort.Slice(wr, func(i, j int) bool {
		return bytes.Compare(wr[i][:], wr[j][:]) < 0
	})

	for _, k := range wr {
		binary.Write(h, binary.BigEndian, k[:])
		binary.Write(h, binary.BigEndian, op.WriteRows[k].K)
		binary.Write(h, binary.BigEndian, op.WriteRows[k].V)
	}

	copy(id[:], h.Sum(nil))
	return
}
