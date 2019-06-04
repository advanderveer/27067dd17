package engine

import (
	"sync"

	"github.com/advanderveer/27067dd17/onl"
)

//MemPool stores pending writes before they are committed into the chain
type MemPool struct {
	writes map[onl.Nonce]*onl.Write
	mu     sync.RWMutex

	//for inspiration about mempool handling in bitcoin:
	//https://blog.kaiko.com/an-in-depth-guide-into-how-the-mempool-works-c758b781c608
	//https://bitcoin.stackexchange.com/questions/59257/what-going-to-happend-with-transactions-that-is-in-both-rejectend-and-accepted-c
}

//NewMemPool creates a new mem pool
func NewMemPool() (p *MemPool) {
	p = &MemPool{writes: make(map[onl.Nonce]*onl.Write)}
	return
}

// Pick writes from the mempool that are not in the provided tip and do not
// cause a conflict in the chosen order. It calls f for every suitable write.
func (p *MemPool) Pick(state *onl.State, f func(w *onl.Write) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, w := range p.writes {

		//apply will check if the write would conflict or is alreayd applied
		//to the provided state
		err := state.Apply(w, true)
		if err != nil {
			continue
		}

		stop := f(w)
		if stop {
			break
		}
	}

	return
}

// Add will attempt to put a write in the mempool if it returns an error it failed
// to do so
func (p *MemPool) Add(w *onl.Write) (err error) {
	if !w.VerifySignature() {
		return ErrInvalidWriteSignature
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, ok := p.writes[w.Nonce]
	if ok {
		return ErrAlreadyInPool
	}

	p.writes[w.Nonce] = w
	return
}
