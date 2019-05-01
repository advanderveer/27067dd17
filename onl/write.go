package onl

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"sync"

	"github.com/advanderveer/27067dd17/onl/ssi"
	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//WID uniquely identifies a write
type WID [sha256.Size]byte

//Nonce is a number only used once
type Nonce [32]byte

//Bytes returns the slice version
func (wid WID) Bytes() []byte { return wid[:] }

// Write encodes signed modifications to the keyspace that will be replicated
// on each member through the consensus protocol
type Write struct {
	*ssi.TxData

	// The primary key that identifies the identity that proposes the write
	PK PK

	//Nonce is a random large number that should be generated randomly and may only
	//appear once in the longest chain. If a nonce is while another write already has
	//it (either in the current mempool or in the chain) the write will not be accepted.
	Nonce Nonce

	//Signature of the block, signed by the identity of PK such that it can be verified
	//that the block has not been tampered with
	Signature [ed25519.SignatureSize]byte

	//@TODO we rather get rid of the lock
	mu sync.RWMutex
}

//Lock the write
func (w *Write) Lock() { w.mu.Lock() }

//Unlock the write
func (w *Write) Unlock() { w.mu.Unlock() }

//RLock the write
func (w *Write) RLock() { w.mu.RLock() }

//RUnlock the write
func (w *Write) RUnlock() { w.mu.RUnlock() }

// HasDepositFor returns whether this write writes a deposit for the provided
// identity. This is used to find a random seed for the VRF
func (w *Write) HasDepositFor(pk PK) bool {
	var (
		wroteTPK   bool
		wroteStake bool
	)

	for _, wr := range w.TxData.WriteRows {
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
func (w *Write) Hash() (id WID) {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, w.TimeStart)
	binary.Write(h, binary.BigEndian, w.TimeCommit)
	binary.Write(h, binary.BigEndian, w.Nonce)
	binary.Write(h, binary.BigEndian, w.PK)

	//read rows, sorted
	rr := make([]ssi.KH, 0, len(w.ReadRows))
	for k := range w.ReadRows {
		rr = append(rr, k)
	}

	sort.Slice(rr, func(i, j int) bool {
		return bytes.Compare(rr[i][:], rr[j][:]) < 0
	})

	for _, k := range rr {
		binary.Write(h, binary.BigEndian, k[:])
	}

	//Write rows, sorted
	wr := make([]ssi.KH, 0, len(w.WriteRows))
	for k := range w.WriteRows {
		wr = append(wr, k)
	}

	sort.Slice(wr, func(i, j int) bool {
		return bytes.Compare(wr[i][:], wr[j][:]) < 0
	})

	for _, k := range wr {
		binary.Write(h, binary.BigEndian, k[:])
		binary.Write(h, binary.BigEndian, w.WriteRows[k].K)
		binary.Write(h, binary.BigEndian, w.WriteRows[k].V)
	}

	copy(id[:], h.Sum(nil))
	return
}

//VerifySignature will check the block's signature
func (w *Write) VerifySignature() (ok bool) {
	pk := [32]byte(w.PK)
	return ed25519.Verify(&pk, w.Hash().Bytes(), &w.Signature)
}

//GenerateNonce generates a nonce from crypto random bytes
func (w *Write) GenerateNonce() (err error) {
	_, err = rand.Read(w.Nonce[:])
	return
}
