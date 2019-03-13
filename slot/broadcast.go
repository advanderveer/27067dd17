package slot

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"io"
	"sync"
)

// Broadcast represents a asyncornous message transport that is expected to
// reliabily deliver to all members of the network eventually. It will
// dedublicate the messages for each reader to prevent a broadcast storm.
type Broadcast interface {
	Read(m *Msg) (err error)
	BroadcastWriter
}

// BroadcastWriter represents the writing part of the broadcast such that it
// can easily be passed around.
type BroadcastWriter interface {
	Write(m *Msg) (err error)
}

// MemNetwork is an in memory broadcast network
type MemNetwork struct {
	eps map[*MemEndpoint]struct{}
	mu  sync.Mutex
}

// NewMemNetwork creates a new memory network, it will correctly copy messages
// such that readers cannot change values of the writers
func NewMemNetwork() *MemNetwork {
	return &MemNetwork{eps: make(map[*MemEndpoint]struct{})}
}

// Write to the broadcast network
func (netw *MemNetwork) Write(m *Msg) (err error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err = enc.Encode(m)
	if err != nil {
		return err //failed to encode
	}

	for ep := range netw.eps {

		//deduplicate messages
		ep.mu.Lock()
		id := sha256.Sum256(buf.Bytes())
		if _, ok := ep.dedub[id]; ok {
			ep.mu.Unlock()
			continue
		}

		ep.dedub[id] = struct{}{}
		ep.mu.Unlock()

		//schedule for reader to pick up
		ep.rc <- buf
	}

	return nil
}

// Endpoint returns a single broadcast endpoint
func (netw *MemNetwork) Endpoint() (ep *MemEndpoint) {
	netw.mu.Lock()
	defer netw.mu.Unlock()

	ep = &MemEndpoint{
		rc:         make(chan *bytes.Buffer, 1),
		dedub:      make(map[[sha256.Size]byte]struct{}),
		MemNetwork: netw,
	}

	netw.eps[ep] = struct{}{}
	return ep
}

// MemEndpoint is an endpoint in the memory network that can be used for broadcasting
type MemEndpoint struct {
	rc    chan *bytes.Buffer
	dedub map[[sha256.Size]byte]struct{}
	mu    sync.Mutex
	*MemNetwork
}

// Reads a single message from the broadcast network, blocks un
func (ep *MemEndpoint) Read(m *Msg) (err error) {
	rmsg := <-ep.rc
	if rmsg == nil {
		return io.EOF
	}

	dec := gob.NewDecoder(rmsg)
	return dec.Decode(m)
}

// Close will shutdown the endpoint, reads will return EOF and writes
// to the network will no longer be send to this endpoint
func (ep *MemEndpoint) Close() (err error) {
	close(ep.rc)

	ep.MemNetwork.mu.Lock()
	defer ep.MemNetwork.mu.Unlock()
	delete(ep.MemNetwork.eps, ep)
	return
}
