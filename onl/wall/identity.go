package wall

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//SignPK is the signing public key
type SignPK [32]byte

//Bytes returns a byte slice of the signing pk
func (pk SignPK) Bytes() []byte { return pk[:] }

//RandPK is the verfiable randomness public key
type RandPK [32]byte

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
		copy(rb[32:], rndid)
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

// SignPK returns the public key used for signing
func (idn *Identity) SignPK() (pk SignPK) {
	copy(pk[:], idn.signPK[:])
	return
}

// SignTransfer will sign the transfer with the identities signing key and set
// the sender to this identity.
func (idn *Identity) SignTransfer(tr *Tr) *Tr {

	//set the sender to this identity
	tr.Sender = idn.SignPK()

	//empty the existing signature before hashing
	tr.Signature = [64]byte{}

	//finally, sign the transfer
	tr.Signature = *(ed25519.Sign(idn.signSK, tr.Hash().Bytes()))
	return tr
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
