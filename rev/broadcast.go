package rev

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
	"sync"

	"github.com/advanderveer/27067dd17/vrf"
)

//Msg transports information over broaddcast
type Msg struct {
	Proposal *Proposal
}

//Broadcast provide reliable message dissemation
type Broadcast interface {
	Read(msg *Msg) (err error)
	Write(msg *Msg) (err error)
	Close() (err error)
}

//Injector allows for writing specific messages to broadcast, mainly usefull for
//black box testing of the protocol
type Injector struct {
	pk []byte
	sk *[vrf.SecretKeySize]byte
	*MemBroadcast
}

//NewInjector creates an injector with identity using the reader for random bytes
func NewInjector(rndid []byte) (inj *Injector) {
	inj = &Injector{MemBroadcast: NewMemBroadcast()}

	var err error
	rndr := rand.Reader
	if rndid != nil {
		rb := make([]byte, 32)
		copy(rb, rndid)
		rndr = bytes.NewReader(rb)
	}

	inj.pk, inj.sk, err = vrf.GenerateKey(rndr)
	if err != nil {
		panic("failed to generate vrf keys for injector: " + err.Error())
	}

	return
}

// Propose the following or panic by writing to the broadcast
func (inj *Injector) Propose(round uint64, b *Block, witness ...*Proposal) (p *Proposal) {
	msg := &Msg{Proposal: NewProposal(inj.pk, inj.sk, round)}
	msg.Proposal.Block = b
	for _, w := range witness {
		msg.Proposal.Witness.Add(w.Hash())
	}

	err := inj.Write(msg)
	if err != nil {
		panic("failed to inject proposal: " + err.Error())
	}

	return msg.Proposal
}

//MemBroadcast is an in-memory broadcast implementation
type MemBroadcast struct {
	closed bool
	bufc   chan *bytes.Buffer
	peers  map[*MemBroadcast]struct{}
	mu     sync.RWMutex
}

//NewMemBroadcast creates an in-memory broadcast endpoint
func NewMemBroadcast() (m *MemBroadcast) {
	m = &MemBroadcast{
		peers: make(map[*MemBroadcast]struct{}),
		bufc:  make(chan *bytes.Buffer, 1),
	}
	return
}

//To will add another broadcast endpoint this endpoint will write messages to
func (bc *MemBroadcast) To(peers ...*MemBroadcast) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range peers {
		bc.peers[p] = struct{}{}
	}
}

//Read a message from the broadcast
func (bc *MemBroadcast) Read(msg *Msg) (err error) {
	rmsg := <-bc.bufc
	if rmsg == nil {
		return io.EOF
	}

	dec := gob.NewDecoder(rmsg)
	return dec.Decode(msg)
}

//Write a message to the broadcast
func (bc *MemBroadcast) Write(msg *Msg) (err error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err = enc.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode broadcast message: %v", err)
	}

	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for p := range bc.peers {
		if p.closed {
			continue
		}

		p.bufc <- bytes.NewBuffer(buf.Bytes())
	}

	return
}

//Close this broadcast endpoint
func (bc *MemBroadcast) Close() (err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	close(bc.bufc)

	bc.closed = true
	return
}
