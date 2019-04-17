package ssi

import (
	iradix "github.com/hashicorp/go-immutable-radix"
)

//DB creates the database
type DB struct {
	txReqs     chan *txReq     //transaction starts
	commitReqs chan *commitReq //transaction commits

	oracle *Oracle
	store  *iradix.Tree
}

//NewDB sets up the database
func NewDB() (db *DB) {
	db = &DB{
		txReqs:     make(chan *txReq),
		commitReqs: make(chan *commitReq),

		oracle: NewOracle(),
		store:  iradix.New(),
	}

	go func() {
		for {

			select {
			case req := <-db.txReqs:
				tx := &Tx{
					c:        db,
					snapshot: db.store.Txn(), //fetch a poin-in-time copy
					// commitC:  db.commitReqs,
					data: &TxData{
						TimeStart: db.oracle.Curr(), //pick up a read time stamp
						ReadRows:  make(KeySet),
						WriteRows: make(KeyChangeSet),
					},
				}

				req.tx <- tx
			case req := <-db.commitReqs:

				//commit to the oracle for a timestamp
				tc := db.oracle.Commit(req.rr, req.rw.KeySet(), req.ts)
				if tc == 0 {
					req.tc <- tc //no commit time, must be a conflict
					break
				}

				//no conflict, go ahead and write changes to the store
				for _, change := range req.rw {
					db.store, _, _ = db.store.Insert(change.K, change.V) //write actual changes for new reads
				}

				req.tc <- tc
			}
		}
	}()

	return
}

//NewTx creates an new transaction
func (db *DB) NewTx() *Tx {
	req := &txReq{tx: make(chan *Tx)}
	db.txReqs <- req
	return <-req.tx
}

//Commit a transaction with just it's data portion.
func (db *DB) Commit(txd *TxData) (err error) {
	if len(txd.WriteRows) < 1 {
		return nil //nothing to commit
	}

	req := &commitReq{
		tc: make(chan uint64),
		rr: txd.ReadRows,
		rw: txd.WriteRows,
		ts: txd.TimeStart,
	}

	db.commitReqs <- req
	txd.TimeCommit = <-req.tc
	if txd.TimeCommit == 0 {
		return ErrConflict
	}

	return
}

//tx req is send when a user requests a new transaction
type txReq struct {
	tx chan *Tx
}

//commitReq is send when a user wants to commit a transaction
type commitReq struct {
	rw KeyChangeSet
	rr KeySet

	ts uint64
	tc chan uint64
}
