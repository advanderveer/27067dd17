package onl

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//Identity represents is an unique Sybil in network
type Identity struct {
	mu     sync.RWMutex
	name   string
	vrfPK  []byte
	vrfSK  *[vrf.SecretKeySize]byte
	signPK *[ed25519.PublicKeySize]byte
	signSK *[ed25519.PrivateKeySize]byte
}

//NewIdentity will start a new identity from the provided identity bytes, if nil
//random bytes are used
func NewIdentity(rndid []byte) (idn *Identity) {
	idn = &Identity{}

	var err error
	rndr := rand.Reader
	if rndid != nil {
		rb := make([]byte, 64)
		copy(rb, rndid)
		copy(rb, rndid)
		rndr = bytes.NewReader(rb)
	}

	idn.vrfPK, idn.vrfSK, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys for injector: " + err.Error())
	}

	idn.signPK, idn.signSK, err = ed25519.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate sign keys for injector: " + err.Error())
	}

	return
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

	return fmt.Sprintf("%.4x", *idn.signPK)
}

//TokenPK returns this identity's verfiable random function public key
func (idn *Identity) TokenPK() []byte { return idn.vrfPK }

//PK returns this identities signing key
func (idn *Identity) PK() []byte { return (*idn.signPK)[:] }

//Sign the block's id which is the blocks's hash
func (idn *Identity) Sign(b *Block) {
	b.Signature = *(ed25519.Sign(idn.signSK, b.Hash().Bytes()))
}

//Mint a new block on the provided tip and round by putting up the identities stake
func (idn *Identity) Mint(c Clock, rt uint64, prev, fPrev ID) (b *Block) {
	idn.mu.RLock()
	defer idn.mu.RUnlock()

	b = &Block{Timestamp: c.ReadUs(), Prev: prev, FinalizedPrev: fPrev}
	copy(b.PK[:], (*idn.signPK)[:])

	//identities are allowed to mint exactly one block per round but is randomized
	//by referencing the hash of the last finalized block. When announcing participation
	//each identity commits to a single sign PK that must be used for vrf generation

	b.Token, b.Proof = vrf.Prove(b.VRFSeed(rt), idn.vrfSK)
	return
}

//Join will create a join operation that can be broadcasted to indicate this
//identity would like to participate in the protocol
func (idn *Identity) Join(deposit uint64) (op *JoinOp) {
	op = &JoinOp{Deposit: deposit}
	copy(op.TokenPK[:], idn.vrfPK)
	copy(op.Identity[:], (*idn.signPK)[:])
	op.Signature = *(ed25519.Sign(idn.signSK, op.Hash().Bytes()))

	return
}
