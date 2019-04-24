package engine

import (
	"io"
	"sync"
)

//Pulse provides an interface for the system on a synchronized pulse
type Pulse interface {
	Next() (err error)
	Close() (err error)
}

//MemPulse is an simulated pulse that runs in-memory
type MemPulse struct {
	closed bool
	mu     sync.Mutex
	c      chan *struct{}
}

// Next returns the next pulse or EOF if its closed
func (p *MemPulse) Next() (err error) {
	rmsg := <-p.c
	if rmsg == nil {
		return io.EOF
	}

	return nil
}

// Close this pulse
func (p *MemPulse) Close() (err error) {
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
	pulses map[*MemPulse]struct{}
	mu     sync.RWMutex
}

// NewMemOscillator creates an in-memory oscillator
func NewMemOscillator() (osc *MemOscillator) {
	osc = &MemOscillator{
		pulses: make(map[*MemPulse]struct{}),
	}
	return
}

// Fire will signal each pulse to fire
func (osc *MemOscillator) Fire() {
	osc.mu.RLock()
	defer osc.mu.RUnlock()
	for p := range osc.pulses {
		p.mu.Lock()

		if p.closed {
			p.mu.Unlock()
			continue
		}

		p.c <- &struct{}{}
		p.mu.Unlock()
	}
}

// Pulse creates an pulse that will be fired by this oscillator
func (osc *MemOscillator) Pulse() (p *MemPulse) {
	p = &MemPulse{c: make(chan *struct{}, 1)}
	osc.mu.Lock()
	defer osc.mu.Unlock()
	osc.pulses[p] = struct{}{}
	return
}
