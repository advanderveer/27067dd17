package onl

import (
	"fmt"

	"github.com/advanderveer/27067dd17/onl/ssi"
)

//KV abstraction build on top of our block chain
type KV struct{ *ssi.Tx }

// State represents the data stored in a chain suitable for access in constant-time.
// It is created by flattening a chain of blocks and applying each operation in
// total order.
type State struct {
	db *ssi.DB
}

// NewState initialized a state, reconstructing from any existing state from the log
func NewState(log [][]*Write) (s *State, err error) {
	s = &State{db: ssi.NewDB()}

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
//the data would be added but isn't actually.
func (s *State) Apply(w *Write, dry bool) (err error) {
	err = s.db.Commit(w.TxData, dry)
	if err == ssi.ErrConflict {
		return ErrApplyConflict
	}

	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return
}

//Read data from the state, any writes will be ignored
func (s *State) Read(f func(kv *KV)) {
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
