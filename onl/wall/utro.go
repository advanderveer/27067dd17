package wall

import "sync"

//UTRO holds unspend transfer outputs, similar to Bitcoin UTXO. It does however
//also keep time locked outputs.
type UTRO struct {
	outputs map[OID]*TrOut
	mu      sync.RWMutex
}

//NewUTRO initiates an utro set
func NewUTRO() *UTRO {
	return &UTRO{outputs: make(map[OID]*TrOut)}
}

//Put (over)writes an output to the utro set
func (utro *UTRO) Put(id OID, out TrOut) {
	utro.mu.Lock()
	defer utro.mu.Unlock()

	utro.outputs[id] = &out
}

//Del removes an item from uto set
func (utro *UTRO) Del(id OID) {
	utro.mu.Lock()
	defer utro.mu.Unlock()

	delete(utro.outputs, id)
}

//Get attemps a read of the utro set
func (utro *UTRO) Get(id OID) (out *TrOut, ok bool) {
	utro.mu.RLock()
	defer utro.mu.RUnlock()

	out, ok = utro.outputs[id]
	if !ok {
		return nil, false
	}

	return out, true
}
