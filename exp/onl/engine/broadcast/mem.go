package broadcast

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
)

//Mem is an in-memory broadcast implementation
type Mem struct {
	closed bool
	bufc   chan *mmsg
	peers  map[*Mem]struct{}
	mu     sync.RWMutex

	minl time.Duration
	maxl time.Duration
}

type mmsg struct {
	remote *Mem
	buf    *bytes.Buffer
}

//NewMem creates an in-memory broadcast endpoint
func NewMem(bufn int) (m *Mem) {
	m = &Mem{
		peers: make(map[*Mem]struct{}),
		bufc:  make(chan *mmsg, bufn),
	}
	return
}

//WithLatency will introduce a random latency to each peers that is added
//after this method is called
func (bc *Mem) WithLatency(min, max time.Duration) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.minl = min
	bc.maxl = max
}

func (bc *Mem) latency() (l time.Duration) {
	if bc.maxl > bc.minl {
		l = bc.minl + time.Duration(rand.Int63n(int64(bc.maxl)-int64(bc.minl)))
	}

	return
}

//To will add another broadcast endpoint for this endpoint to write messages to
func (bc *Mem) To(peers ...*Mem) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range peers {
		bc.peers[p] = struct{}{}
	}
}

//Read a message sent by peers
func (bc *Mem) Read(msg *engine.Msg) (err error) {
	rmsg := <-bc.bufc
	if rmsg == nil {
		return io.EOF
	}

	dec := gob.NewDecoder(rmsg.buf)
	err = dec.Decode(msg)
	if err != nil {
		return err
	}

	if msg.Sync != nil {
		msg.Sync.SetWF(func(b *onl.Block) (err error) {
			buf := bytes.NewBuffer(nil)
			enc := gob.NewEncoder(buf)
			err = enc.Encode(&engine.Msg{Block: b})
			if err != nil {
				return fmt.Errorf("failed to encode broadcast message: %v", err)
			}

			//note: latency may be added, synchronous
			return peerWrite(bc, rmsg.remote, buf.Bytes(), bc.latency())
		})
	}

	return
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
	for peer := range bc.peers {
		go peerWrite(bc, peer, buf.Bytes(), bc.latency())
	}

	return
}

func peerWrite(from *Mem, to *Mem, b []byte, latency time.Duration) (err error) {
	if latency > 0 {
		time.Sleep(latency)
	}

	to.mu.RLock()
	defer to.mu.RUnlock()
	if to.closed {
		return ErrClosed
	}

	to.bufc <- &mmsg{
		remote: from,
		buf:    bytes.NewBuffer(b),
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
