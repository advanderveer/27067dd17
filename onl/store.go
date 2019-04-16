package onl

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dgraph-io/badger"
)

//Store stores blocks
type Store interface {
	CreateTx(writable bool) Tx
	Close() (err error)
}

//Tx is an ACID interaction with the store
type Tx interface{}

//BadgerStore is a store implemtation that resides solely in memory
type BadgerStore struct {
	db *badger.DB
}

//BadgerTx is an transaction on the badger store
type BadgerTx struct {
	btx *badger.Txn
}

//TempBadgerStore will return a temporary store that will be fully cleaned up when
//the 'clean' func is called. It panics of any of the operations fails is mostly
//used for testing purposes
func TempBadgerStore() (s *BadgerStore, clean func()) {
	dir, err := ioutil.TempDir("", "onl_")
	if err != nil {
		panic("failed to create tempdir: " + err.Error())
	}

	s, err = NewBadgerStore(dir)
	if err != nil {
		panic("failed to create store: " + err.Error())
	}

	return s, func() {
		err = s.Close()
		if err != nil {
			panic("faild to close store: " + err.Error())
		}

		err = os.RemoveAll(dir)
		if err != nil {
			panic("faild to remove dir: " + err.Error())
		}
	}
}

//NewBadgerStore creates a badger powered store
func NewBadgerStore(dir string) (s *BadgerStore, err error) {
	s = &BadgerStore{}

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	s.db, err = badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("faild to open db: %v", err)
	}

	return
}

//CreateTx sets up the transaction
func (s *BadgerStore) CreateTx(writable bool) (tx Tx) {
	mtx := &BadgerTx{btx: s.db.NewTransaction(writable)}
	return mtx
}

//Close the store, removing any open resources
func (s *BadgerStore) Close() (err error) {
	return s.db.Close()
}
