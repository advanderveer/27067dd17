package rev

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"
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
	reads chan *Msg
	idn   *Identity
	*MemBroadcast
}

//NewInjector creates an injector with identity using the reader for random bytes
func NewInjector(rndid []byte) (inj *Injector) {
	inj = &Injector{
		MemBroadcast: NewMemBroadcast(),
		idn:          NewIdentity(rndid),
		reads:        make(chan *Msg, 1000), //reasonably large buffer
	}

	go func() {
		for {
			msg := &Msg{}
			err := inj.Read(msg)
			if err == io.EOF {
				return
			} else if err != nil {
				panic("injector failed to collect: " + err.Error())
			}

			inj.reads <- msg
		}
	}()

	return
}

//Collect returns a channel that buffers messages written to the injector
func (inj *Injector) Collect() <-chan *Msg {
	return inj.reads
}

// Propose the following or panic by writing to the broadcast
func (inj *Injector) Propose(round uint64, b *Block, witness ...*Proposal) (p *Proposal) {
	msg := &Msg{Proposal: inj.idn.CreateProposal(round)}
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
	peers  map[*MemBroadcast]time.Duration
	mu     sync.RWMutex

	minl time.Duration
	maxl time.Duration
}

//NewMemBroadcast creates an in-memory broadcast endpoint
func NewMemBroadcast() (m *MemBroadcast) {
	m = &MemBroadcast{
		peers: make(map[*MemBroadcast]time.Duration),
		bufc:  make(chan *bytes.Buffer, 1000),
	}
	return
}

//WithLatency will introduce a random latency to each peers that is added
//after this method is called
func (bc *MemBroadcast) WithLatency(min, max time.Duration) {
	bc.minl = min
	bc.maxl = max
}

//To will add another broadcast endpoint this endpoint will write messages to
func (bc *MemBroadcast) To(peers ...*MemBroadcast) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range peers {
		l := time.Duration(0)
		if bc.maxl > bc.minl {
			l = time.Duration(rand.Int63n(int64(bc.maxl) - int64(bc.minl)))
		}

		bc.peers[p] = l
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
	for peer, latency := range bc.peers {

		go func(peer *MemBroadcast, latency time.Duration) {
			if latency > 0 {
				time.Sleep(latency)
			}

			peer.mu.RLock()
			defer peer.mu.RUnlock()
			if peer.closed {
				return
			}

			peer.bufc <- bytes.NewBuffer(buf.Bytes())
		}(peer, latency)
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
