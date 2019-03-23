package rev

import (
	"bytes"
	"crypto/rand"

	"github.com/advanderveer/27067dd17/vrf"
)

//Identity represents is an unique Sybil in network
type Identity struct {
	pk []byte
	sk *[vrf.SecretKeySize]byte
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

//CreateProposal will create a proposal that is owned by this identity
func (idn *Identity) CreateProposal(round uint64) (p *Proposal) {
	return NewProposal(idn.pk, idn.sk, round)
}
