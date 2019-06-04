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

	//we need to copy the data else changing the array outside
	//of the tx changes it in the snapshot
	kd := make([]byte, len(k))
	copy(kd, k)
	vd := make([]byte, len(v))
	copy(vd, v)

	tx.snapshot.Insert(kd, vd)
}

//Get the value 'v' at key 'k'
func (tx *Tx) Get(k []byte) (v []byte) {
	tx.data.ReadRows.Add(k)
	vraw, ok := tx.snapshot.Get(k)
	if !ok {
		return nil
	}

	//make sure to copy
	v = make([]byte, len(vraw.([]byte)))
	copy(v, vraw.([]byte))
	return
}

//Data returns the underlying data, suitable for transport
func (tx *Tx) Data() *TxData { return tx.data }

//Commit the transaction
func (tx *Tx) Commit() error {
	return tx.c.Commit(tx.Data(), false)
}
