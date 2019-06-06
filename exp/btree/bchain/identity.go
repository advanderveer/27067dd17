package bchain

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

//PK is the public key we use for an identity
type PK [vrf.PublicKeySize]byte

//Bytes returns a byte slice of the signing pk
func (pk PK) Bytes() []byte { return pk[:] }

//Identity represents is an unique Sybil in network
type Identity struct {
	mu   sync.RWMutex
	name string
	pk   []byte
	sk   *[vrf.SecretKeySize]byte
}

//NewIdentity will start a new identity from the provided identity bytes, if nil
//random bytes are used
func NewIdentity(rndid []byte, rndr io.Reader) (idn *Identity) {
	idn = &Identity{}

	var err error
	if rndid != nil {
		rb := make([]byte, 64)
		copy(rb, rndid)
		copy(rb[32:], rndid)
		rndr = bytes.NewReader(rb)
	}

	idn.pk, idn.sk, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys for injector: " + err.Error())
	}

	return
}

// PublicKey returns the public key used by this identity
func (idn *Identity) PublicKey() (pk PK) {
	idn.mu.RLock()
	defer idn.mu.RUnlock()
	copy(pk[:], idn.pk[:])
	return
}

func (idn *Identity) drawTicket(b *Block, prevt [vrf.Size]byte) {
	token, tproof := vrf.Prove(prevt[:], idn.sk)
	copy(b.Header.Ticket.Token[:], token)
	copy(b.Header.Ticket.Proof[:], tproof)
}

// SignBlock will complete the block with a signature with this identity as the proposer
func (idn *Identity) SignBlock(round uint64, b *Block, prevt [vrf.Size]byte) *Block {
	b.Header.Proposer = idn.PublicKey()

	//draw random ticket
	idn.drawTicket(b, prevt)

	//empty the existing signature elements before hashing
	b.Header.ID = NewBID(round)
	b.Header.Proof = [vrf.ProofSize]byte{}

	// hash the whole block
	h := b.Hash()
	id, proof := vrf.Prove(h[:], idn.sk)

	// copy block level signature elements
	copy(b.Header.ID[8:], id)
	copy(b.Header.Proof[:], proof)
	return b
}

//SetName allows for showing a memorable name when this identity is printed
func (idn *Identity) SetName(name string) {
	idn.mu.Lock()
	defer idn.mu.Unlock()
	idn.name = name
}

//String returns a human readable identity
func (idn *Identity) String() string {
	idn.mu.RLock()
	defer idn.mu.RUnlock()
	if idn.name != "" {
		return idn.name
	}

	return fmt.Sprintf("%.4x", idn.pk)
}
