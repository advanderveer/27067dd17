package topn

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

//PK is our fixed-size public key identity
type PK [32]byte

//Identity represents is an unique Sybil in network
type Identity struct {
	mu   sync.RWMutex
	name string
	pk   []byte
	sk   *[vrf.SecretKeySize]byte
}

//NewIdentity will start a new identity from the provided identity bytes, if nil
//random bytes are used
func NewIdentity(rndid []byte) (idn *Identity) {
	idn = &Identity{}

	var err error
	rndr := rand.Reader
	if rndid != nil {
		rb := make([]byte, 32)
		copy(rb, rndid)
		rndr = bytes.NewReader(rb)
	}

	idn.pk, idn.sk, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys for injector: " + err.Error())
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

	return fmt.Sprintf("%.4x", idn.pk)
}

//CreateBlock will create a block signed by this identity
func (idn *Identity) CreateBlock(round uint64, prev ID) (b *Block) {
	idn.mu.RLock()
	defer idn.mu.RUnlock()
	b = &Block{Round: round, Prev: prev}
	copy(b.PK[:], idn.pk)

	//@TODO (security) we might want to seed the vrf solely on the prev's token
	//value and not on the hash. Else one might be able to mint blocks that they
	//themselves can mint again for a high ranking value the next round. Taking
	//control of the chain forever.

	b.Token, b.Proof = vrf.Prove(prev[:], idn.sk)
	return
}
