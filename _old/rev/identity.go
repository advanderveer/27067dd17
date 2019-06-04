package rev

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

//CreateProposal will create a proposal that is owned by this identity
func (idn *Identity) CreateProposal(round uint64) (p *Proposal) {
	return NewProposal(idn.pk, idn.sk, round)
}
