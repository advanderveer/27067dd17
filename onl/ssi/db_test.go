package ssi

import (
	"testing"

	test "github.com/advanderveer/go-test"
)

func TestBasicTimestamping(t *testing.T) {
	db := NewDB()
	tx := db.NewTx()

	test.Equals(t, uint64(1), tx.data.TimeStart)
	test.Equals(t, uint64(0), tx.data.TimeCommit)

	test.Ok(t, tx.Commit()) //commit should provide the tx with a commit timestamp
	test.Equals(t, uint64(1), tx.data.TimeStart)
	test.Equals(t, uint64(0), tx.data.TimeCommit) //read-only transaction
}

func TestBasicDataStorage(t *testing.T) {
	db := NewDB()
	tx := db.NewTx()

	tx.Set([]byte("alex"), int64b(100))
	c, ok := tx.data.WriteRows[keyHash([]byte("alex"))]
	test.Equals(t, true, ok)
	test.Equals(t, []byte("alex"), c.K)
	test.Equals(t, int64b(100), c.V)

	test.Equals(t, []byte(nil), tx.Get([]byte("bob")))

	_, ok = tx.data.ReadRows[keyHash([]byte("alex"))]
	test.Equals(t, false, ok)

	_, ok = tx.data.ReadRows[keyHash([]byte("bob"))]
	test.Equals(t, true, ok)
	test.Equals(t, int64b(100), tx.Get([]byte("alex")))

	test.Ok(t, tx.Commit())
	test.Equals(t, uint64(2), tx.data.TimeCommit)

	tx2 := db.NewTx()
	test.Equals(t, int64b(100), tx2.Get([]byte("alex")))
	test.Equals(t, []byte(nil), tx2.Get([]byte("bob")))
}

func TestDryRun(t *testing.T) {
	db := NewDB()
	tx := db.NewTx()

	tx.Set([]byte("alex"), []byte{0x01})
	test.Ok(t, db.Commit(tx.Data(), true))
}

func TestReadWriteConflict(t *testing.T) {

	db := NewDB()
	tx1 := db.NewTx()
	tx2 := db.NewTx()

	tx1.Get([]byte("alex"))
	tx1.Set([]byte("bob"), int64b(1))

	tx2.Get([]byte("bob")) //read from other tx's write
	tx2.Set([]byte("alex"), int64b(1))

	test.Ok(t, db.Commit(tx1.Data(), true)) //dry run, should not commit data
	tx3 := db.NewTx()
	test.Equals(t, []byte(nil), tx3.Get([]byte("bob")))

	test.Ok(t, db.Commit(tx1.Data(), false)) //no dry run, should commit data for reading
	tx4 := db.NewTx()
	test.Equals(t, int64b(1), tx4.Get([]byte("bob"))) //should have been committed

	//tx2 should not commit with dry run
	test.Equals(t, ErrConflict, db.Commit(tx2.Data(), true))
}
