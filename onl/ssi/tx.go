package ssi

import (
	iradix "github.com/hashicorp/go-immutable-radix"
)

//Committer is the entity that can commit using just the transportable tx data
type Committer interface {
	Commit(txd *TxData, dry bool) (err error)
}

//TxData holds the transportable portion of the transaction
type TxData struct {
	TimeStart  uint64
	TimeCommit uint64
	ReadRows   KeySet
	WriteRows  KeyChangeSet
}

//Tx is a transaction
type Tx struct {
	c        Committer
	snapshot *iradix.Txn
	data     *TxData
}

//Set key 'k' to value 'v'
func (tx *Tx) Set(k, v []byte) {
	tx.data.WriteRows.Add(k, v)
	tx.snapshot.Insert([]byte(k), v)
}

//Get the value 'v' at key 'k'
func (tx *Tx) Get(k []byte) []byte {
	tx.data.ReadRows.Add(k)
	vraw, ok := tx.snapshot.Get(k)
	if !ok {
		return nil
	}

	return vraw.([]byte)
}

//Data returns the underlying data, suitable for transport
func (tx *Tx) Data() *TxData { return tx.data }

//Commit the transaction
func (tx *Tx) Commit() error {
	return tx.c.Commit(tx.Data(), false)
}
