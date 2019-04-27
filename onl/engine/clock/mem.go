package clock

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

//MemClock is an simulated pulse that runs in-memory
type MemClock struct {
	round  uint64
	closed bool
	mu     sync.Mutex
	c      chan uint64
}

// Round will return the current round nr
func (p *MemClock) Round() uint64 {
	return atomic.LoadUint64(&p.round)
}

// ReadUs will read the current microsecond timestamp
func (p *MemClock) ReadUs() (ts uint64, err error) {
	ts = uint64(time.Now().UnixNano() / (10 ^ 6))
	return
}

// Next returns the next pulse or EOF if its closed
func (p *MemClock) Next() (round, ts uint64, err error) {
	rmsg := <-p.c
	if rmsg < 1 {
		return 0, 0, io.EOF
	}

	atomic.StoreUint64(&p.round, rmsg)
	ts, err = p.ReadUs()
	if err != nil {
		return 0, 0, err
	}

	return rmsg, ts, nil
}

// Close this pulse
func (p *MemClock) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}

	close(p.c)
	p.closed = true
	return
}

//MemOscillator is an in-memory oscillator that simulates an synchronized pulse
type MemOscillator struct {
	round  uint64
	pulses map[*MemClock]struct{}
	mu     sync.RWMutex
}

// NewMemOscillator creates an in-memory oscillator
func NewMemOscillator() (osc *MemOscillator) {
	osc = &MemOscillator{
		pulses: make(map[*MemClock]struct{}),
		round:  1,
	}
	return
}

// Fire will signal each pulse to fire
func (osc *MemOscillator) Fire() {
	osc.mu.RLock()
	defer osc.mu.RUnlock()
	round := atomic.AddUint64(&osc.round, 1)
	for p := range osc.pulses {
		p.mu.Lock()

		if p.closed {
			p.mu.Unlock()
			continue
		}

		p.c <- round
		p.mu.Unlock()
	}
}

// Clock creates an pulse that will be fired by this oscillator
func (osc *MemOscillator) Clock() (p *MemClock) {
	p = &MemClock{
		c:     make(chan uint64, 1),
		round: 1,
	}

	osc.mu.Lock()
	defer osc.mu.Unlock()
	osc.pulses[p] = struct{}{}
	return
}
