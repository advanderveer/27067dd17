package broadcast

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/advanderveer/27067dd17/onl/engine"
)

//Mem is an in-memory broadcast implementation
type Mem struct {
	closed bool
	bufc   chan *bytes.Buffer
	peers  map[*Mem]time.Duration
	mu     sync.RWMutex

	minl time.Duration
	maxl time.Duration
}

//NewMem creates an in-memory broadcast endpoint
func NewMem(bufn int) (m *Mem) {
	m = &Mem{
		peers: make(map[*Mem]time.Duration),
		bufc:  make(chan *bytes.Buffer, bufn),
	}
	return
}

//WithLatency will introduce a random latency to each peers that is added
//after this method is called
func (bc *Mem) WithLatency(min, max time.Duration) {
	bc.minl = min
	bc.maxl = max
}

//To will add another broadcast endpoint this endpoint will write messages to
func (bc *Mem) To(peers ...*Mem) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range peers {
		l := time.Duration(0)
		if bc.maxl > bc.minl {
			l = bc.minl + time.Duration(rand.Int63n(int64(bc.maxl)-int64(bc.minl)))
		}

		bc.peers[p] = l
	}
}

//Read a message from the broadcast
func (bc *Mem) Read(msg *engine.Msg) (err error) {
	rmsg := <-bc.bufc
	if rmsg == nil {
		return io.EOF
	}

	dec := gob.NewDecoder(rmsg)
	return dec.Decode(msg)
}

//Write a message to the broadcast
func (bc *Mem) Write(msg *engine.Msg) (err error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err = enc.Encode(msg)
	if err != nil {
		return fmt.Errorf("failed to encode broadcast message: %v", err)
	}

	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for peer, latency := range bc.peers {

		go func(peer *Mem, latency time.Duration) {
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
func (bc *Mem) Close() (err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	close(bc.bufc)

	bc.closed = true
	return
}
