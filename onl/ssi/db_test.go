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

func TestReadWriteConflict(t *testing.T) {

	db := NewDB()
	tx1 := db.NewTx()
	tx2 := db.NewTx()

	tx1.Get([]byte("alex"))
	tx1.Set([]byte("bob"), int64b(1))

	tx2.Get([]byte("bob")) //read from other tx's write
	tx2.Set([]byte("alex"), int64b(1))

	test.Ok(t, tx1.Commit())
	test.Equals(t, ErrConflict, tx2.Commit())
}
