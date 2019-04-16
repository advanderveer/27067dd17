package onl

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dgraph-io/badger"
)

//Stakes describes the stake distribution as observed by a member
type Stakes struct {
	//@TODO add fields
}

//Store stores blocks
type Store interface {
	CreateTx(writable bool) Tx
	Close() (err error)
}

//Tx is an ACID interaction with the store
type Tx interface {
	Write(b *Block, stk *Stakes) (err error)
	Read(id ID) (b *Block, stk *Stakes, err error)
	Round(nr uint64, f func(b *Block, stk *Stakes) error) (err error)

	Commit() (err error)
	Discard()
}

//BadgerStore is a store implemtation that resides solely in memory
type BadgerStore struct {
	db *badger.DB
}

//BadgerTx is an transaction on the badger store
type BadgerTx struct {
	btx *badger.Txn
}

//Round calls f in lexicographically order of the id for each block in round 'nr'.
func (tx *BadgerTx) Round(nr uint64, f func(b *Block, stk *Stakes) error) (err error) {
	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = 10

	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, nr)

	iter := tx.btx.NewIterator(opt)
	defer iter.Close()

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		item := iter.Item()
		key := item.Key()
		if !bytes.HasPrefix(key, prefix) {
			break
		}

		d, err := item.Value()
		if err != nil {
			return fmt.Errorf("failed to read block value: %v", err)
		}

		b, fin, err := decode(d)
		if err != nil {
			return fmt.Errorf("failed to decode block data: %v", err)
		}

		err = f(b, fin)
		if err != nil {
			return err
		}
	}

	return nil
}

//Write block info and replace any existing info
func (tx *BadgerTx) Write(b *Block, stk *Stakes) (err error) {
	buf := bytes.NewBuffer(nil)
	if err = gob.NewEncoder(buf).Encode(&struct {
		*Block
		*Stakes
	}{b, stk}); err != nil {
		return fmt.Errorf("failed to encode block data: %v", err)
	}

	err = tx.btx.Set(b.Hash().Bytes(), buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to set key data: %v", err)
	}

	return
}

func decode(d []byte) (b *Block, stk *Stakes, err error) {
	bb := &struct {
		*Block
		*Stakes
	}{}

	err = gob.NewDecoder(bytes.NewReader(d)).Decode(bb)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode block data: %v", err)
	}

	return bb.Block, bb.Stakes, nil
}

//Read block data from the store and any finalization info
func (tx *BadgerTx) Read(id ID) (b *Block, stk *Stakes, err error) {
	it, err := tx.btx.Get(id.Bytes())
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil, ErrBlockNotExist
		}

		return nil, nil, fmt.Errorf("failed to get key data: %v", err)
	}

	d, err := it.Value()
	if err != nil {
		return nil, nil, fmt.Errorf("faile to read block data: %v", err)
	}

	return decode(d)
}

//Discard any tx resources
func (tx *BadgerTx) Discard() { tx.btx.Discard() }

//Commit the transaction
func (tx *BadgerTx) Commit() (err error) { return tx.btx.Commit(nil) }

//TempBadgerStore will return a temporary store that will be fully cleaned up when
//the 'clean' func is called. The database is not closed prior to removal and it
//panics if any of the operations fails so this function is mostly used for testing
//purposes
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
