package tt

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

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

//CreateVote will create a vote that is owned by the identity
func (idn *Identity) CreateVote(tip ID) (v *Vote) {
	idn.mu.RLock()
	defer idn.mu.RUnlock()
	v = &Vote{Tip: tip, PK: idn.pk}
	v.Token, v.Proof = vrf.Prove(tip[:], idn.sk)
	return
}
