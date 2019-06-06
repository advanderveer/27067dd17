package slot

import (
	"bytes"
	"encoding/gob"
	"io"
	"sync"
)

// Broadcast2 provides the ability of reading and writing messages
type Broadcast2 interface {
	Read(round uint64, msg *Msg2) (err error)
	BroadcastWriter2
}

// BroadcastWriter2 provide the ability to write to the broadcast layer
type BroadcastWriter2 interface {
	Write(round uint64, msg *Msg2) (err error)
}

//Collect2 will read all messages from its endpoint and store them
func Collect2(round uint64) (bc *MemBroadcast2, done func() chan []*Msg2) {
	bc = NewMemBroadcast2()

	donec := make(chan []*Msg2)
	msgs := []*Msg2{}

	go func() {
		for {
			msg := &Msg2{}
			err := bc.Read(round, msg)
			if err == io.EOF {
				donec <- msgs
				return
			}

			if err != nil {
				panic("collect: failed to read: " + err.Error())
			}

			msgs = append(msgs, msg)
		}
	}()

	return bc, func() chan []*Msg2 {
		err := bc.Close()
		if err != nil {
			panic("collect: failed to close: " + err.Error())
		}

		return donec
	}
}

//MemBroadcast2 is an in-memory broadcast endpoint that can join to other
//endpoint such that everything that is written to this endpoint is send
//to the other endpoints
type MemBroadcast2 struct {
	closed bool
	peers  map[*MemBroadcast2]struct{}
	chans  map[uint64]chan *bytes.Buffer
	mu     sync.RWMutex
}

//NewMemBroadcast2 creates an in-memory broadcast
func NewMemBroadcast2() (bc *MemBroadcast2) {
	bc = &MemBroadcast2{
		peers: make(map[*MemBroadcast2]struct{}),
		chans: make(map[uint64]chan *bytes.Buffer),
	}
	return bc
}

//Relay this broadcast to other peers, relaying any writes to their reads
func (bc *MemBroadcast2) Relay(peers ...*MemBroadcast2) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, o := range peers {
		bc.peers[o] = struct{}{}
	}
}

func (bc *MemBroadcast2) roundChan(round uint64) (c chan *bytes.Buffer) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	c, ok := bc.chans[round]
	if !ok {
		c = make(chan *bytes.Buffer, 1)
		bc.chans[round] = c
	}

	return
}

// Close the broadcast endpoints, all reads will now return EOF
func (bc *MemBroadcast2) Close() (err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, c := range bc.chans {
		close(c)
	}

	bc.closed = true
	return
}

// Read message from broadcast endpoint
func (bc *MemBroadcast2) Read(round uint64, msg *Msg2) (err error) {
	rmsg := <-bc.roundChan(round)
	if rmsg == nil {
		return io.EOF
	}

	dec := gob.NewDecoder(rmsg)
	return dec.Decode(msg)
}

// Write message to broadcast endpoint
func (bc *MemBroadcast2) Write(round uint64, msg *Msg2) (err error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err = enc.Encode(msg)
	if err != nil {
		return err //failed to encode
	}

	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for peer := range bc.peers {
		peer.mu.RLock()
		if peer.closed {
			peer.mu.RUnlock()
			continue
		}

		peer.mu.RUnlock()
		c := peer.roundChan(round)
		c <- bytes.NewBuffer(buf.Bytes())
	}

	return
}
