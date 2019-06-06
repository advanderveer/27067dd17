package onl

import (
	"fmt"
	"sync"

	"github.com/advanderveer/27067dd17/onl/ssi"
)

// State represents the data stored in a chain suitable for access in constant-time.
// It is created by flattening a chain of blocks and applying each operation in
// total order.
type State struct {
	db *ssi.DB

	writes map[Nonce]struct{}
	mu     sync.RWMutex
}

// NewState initialized a state, reconstructing from any existing state from the log
func NewState(log [][]*Write) (s *State, err error) {
	s = &State{
		db:     ssi.NewDB(),
		writes: make(map[Nonce]struct{}),
	}

	for _, ws := range log {
		for _, w := range ws {
			err = s.Apply(w, false)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}

//Apply will try to perform the write while making sure no other reads have been
//performed concurrently. It can be applied in a dry run, which only pretends that
//the data would be added but isn't actually. This method is also called in the
//process of replicating block writes. If apply returns an error it will not be
//accepted by peers.
func (s *State) Apply(w *Write, dry bool) (err error) {
	if w == nil {
		return //nil writes can happen if update calls result in zero writes
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	//check if the write was already applied in this state
	if _, ok := s.writes[w.Nonce]; ok {
		return ErrAlreadyApplied
	}

	//@TODO validate how keys are written, authentication and authorization
	//@TODO prevent editing of system keys like "balance" and "stake"
	//@TODO enforce special rules such as balance not becoming negative
	//@TODO some operations can only be done in the genesis block
	//@TODO some operations can only be done with proof of misbehaviour
	//@TODO validate max key and value lengths
	//@TODO check signature

	//commit to ssi db, or return conflict
	//@TODO we lock the write here because in some conditions it is simultaneously
	//being written (read) to the broadcast. This solution is rather in-elegant and
	//we rather solve the root cause of that issue
	w.Lock()
	err = s.db.Commit(w.TxData, dry)
	w.Unlock()
	if err == ssi.ErrConflict {
		return ErrApplyConflict
	}

	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	//mark this write as part of the state
	if !dry {
		s.writes[w.Nonce] = struct{}{}
	}

	return
}

//View data from the state, any writes will be ignored
func (s *State) View(f func(kv *KV)) {
	f(&KV{s.db.NewTx()})
}

//Update the state at the current height of the chain and return a write that
//can be broadcasted to others to reach consensus on.
func (s *State) Update(f func(kv *KV)) (w *Write) {
	tx := s.db.NewTx()
	f(&KV{tx})
	w = &Write{TxData: tx.Data()}
	if len(w.TxData.WriteRows) < 1 {
		return nil //no write rows means an empty op, make it nil
	}

	return
}
