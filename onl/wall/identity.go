package wall

import (
	"bytes"
	"crypto/rand"
	"fmt"
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

	idn.pk, idn.sk, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys for injector: " + err.Error())
	}

	return
}

// PublicKey returns the public key used by this identity
func (idn *Identity) PublicKey() (pk PK) {
	copy(pk[:], idn.pk[:])
	return
}

// SignBlock will sign the block as this identity as the voter
func (idn *Identity) SignBlock(b *Block, prevt [vrf.Size]byte) *Block {
	b.Vote.Voter = idn.PublicKey()

	//generate ticket
	token, tproof := vrf.Prove(prevt[:], idn.sk)
	copy(b.Ticket.Token[:], token)
	copy(b.Ticket.Proof[:], tproof)

	//sign the vote separately
	vh := b.Vote.Hash()
	b.Vote.Signature = [vrf.Size]byte{}
	b.Vote.Proof = [vrf.ProofSize]byte{}
	vsig, vproof := vrf.Prove(vh[:], idn.sk)
	copy(b.Vote.Signature[:], vsig)
	copy(b.Vote.Proof[:], vproof)

	//empty the existing signature elements before hashing
	b.ID = [vrf.Size]byte{}
	b.Proof = [vrf.ProofSize]byte{}

	// hash the whole block
	h := b.Hash()
	id, proof := vrf.Prove(h[:], idn.sk)

	// copy block level signature elements
	copy(b.ID[:], id)
	copy(b.Proof[:], proof)
	return b
}

// SignTransfer will sign the transfer with this identity as the sender
func (idn *Identity) SignTransfer(tr *Tr) *Tr {

	//set the sender to this identity
	tr.Sender = idn.PublicKey()

	//empty the existing signature elements before hashing
	tr.ID = [vrf.Size]byte{}
	tr.Proof = [vrf.ProofSize]byte{}

	//create a verifiable random id from the transfer's content
	h := tr.Hash()
	id, proof := vrf.Prove(h[:], idn.sk)

	//copy the verfiably random id and the proof of it
	copy(tr.ID[:], id)
	copy(tr.Proof[:], proof)
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

	return fmt.Sprintf("%.4x", idn.pk)
}
