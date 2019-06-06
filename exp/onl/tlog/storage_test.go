package tlog

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/advanderveer/go-test"
)

// TestTransparentLogRollback tries to figure out if it is easy to rollback
// the storage of a transparent log. That is, can we simply truncate the the
// append only hash file? And append now records?
func TestTransparentLogRollback(t *testing.T) {

	// file based hash reader
	hr := newFileHashStorage()

	//number of records to start
	n := int64(1000)

	//build large record set
	recs := make([][]byte, n)
	for i := int64(0); i < n; i++ {
		recs[i] = []byte(strconv.FormatInt(i, 10))

		hashes, err := StoredHashes(i, recs[i], hr)
		test.Ok(t, err)

		hr = hr.Append(hashes).(*fileHashStorage)
	}

	//should be able to calculate tree hash
	whole, err := TreeHash(n, hr)
	test.Ok(t, err)

	//end of the log
	rp999, err := ProveRecord(n, n-1, hr)
	test.Ok(t, err)
	test.Ok(t, CheckRecord(rp999, n, whole, n-1, RecordHash([]byte(fmt.Sprintf("%d", n-1)))))

	//beginning of the log
	rp0, err := ProveRecord(n, 0, hr)
	test.Ok(t, err)
	test.Ok(t, CheckRecord(rp0, n, whole, 0, RecordHash([]byte("0"))))

	//proof whole tree proof to itself
	tpAll, err := ProveTree(n, n, hr)
	test.Ok(t, err)
	test.Ok(t, CheckTree(tpAll, n, whole, n, whole))

	//creat proves of half trees
	for nn := n; nn > 1; nn = nn / 2 {
		half, err := TreeHash(nn, hr)
		test.Ok(t, err)

		tpHalf, err := ProveTree(n, nn, hr)
		test.Ok(t, err)

		test.Ok(t, CheckTree(tpHalf, n, whole, nn, half))

		//check halfway record proofs
		rp, err := ProveRecord(nn, nn-1, hr)
		test.Ok(t, err)
		test.Ok(t, CheckRecord(rp, nn, half, nn-1, RecordHash([]byte(fmt.Sprintf("%d", nn-1)))))
	}

	//creat proves of half trees with rollback
	for nn := n; nn > 1; nn = nn / 2 {
		hr.Rollback(nn)

		half, err := TreeHash(nn, hr)
		test.Ok(t, err)

		tpHalf, err := ProveTree(nn, nn, hr)
		test.Ok(t, err)

		// proof for itself should work
		test.Ok(t, CheckTree(tpHalf, nn, half, nn, half))

		//end of the log
		endp, err := ProveRecord(nn, nn-1, hr)
		test.Ok(t, err)
		test.Ok(t, CheckRecord(endp, nn, half, nn-1, RecordHash([]byte(fmt.Sprintf("%d", nn-1)))))

		//beginning of the log
		rp0, err := ProveRecord(nn, 0, hr)
		test.Ok(t, err)
		test.Ok(t, CheckRecord(rp0, nn, half, 0, RecordHash([]byte("0"))))
	}
}
