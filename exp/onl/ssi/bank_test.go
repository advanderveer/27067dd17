package ssi

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	test "github.com/advanderveer/go-test"
)

func TestBank(t *testing.T) {
	b := NewBank()

	n, err := b.CurrentBalance("alex")
	test.Equals(t, ErrAccountNotExist, err)
	test.Equals(t, int64(0), n)

	test.Ok(t, b.OpenAccount("alex", 100))
	test.Equals(t, ErrAccountExists, b.OpenAccount("alex", 100))

	test.OkEquals(t, int64(100))(b.CurrentBalance("alex"))

	test.Equals(t, ErrAccountNotExist, b.TransferFunds(40, "alex", "bob"))
	test.Ok(t, b.OpenAccount("bob", 0))
	test.Ok(t, b.TransferFunds(40, "alex", "bob"))
	test.OkEquals(t, int64(40))(b.CurrentBalance("bob"))

	test.Equals(t, ErrNotEnoughFunds, b.TransferFunds(61, "alex", "bob"))

	test.Ok(t, b.TransferFunds(40, "alex", "alex"))
	test.OkEquals(t, int64(60))(b.CurrentBalance("alex"))
}

func TestConcurrentBankFuzzing(t *testing.T) {
	b := NewBank()
	nAccounts := 10
	startFunds := 100
	maxTransfer := 25

	for i := 0; i < nAccounts; i++ {
		test.Ok(t, b.OpenAccount(strconv.Itoa(i), int64(startFunds)))
	}

	var wg sync.WaitGroup
	for i := 0; i < 50000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			amount := rand.Intn(maxTransfer)
			from := strconv.Itoa(rand.Intn(nAccounts))
			to := strconv.Itoa(rand.Intn(nAccounts))
			b.TransferFunds(int64(amount), from, to)

		}()
	}

	wg.Wait()

	var tot int64
	for i := 0; i < nAccounts; i++ {
		bal, err := b.CurrentBalance(strconv.Itoa(i))
		test.Ok(t, err)
		test.Assert(t, bal >= 0, "balance should be 0 or more")
		tot += bal
	}

	test.Equals(t, int64(nAccounts*startFunds), tot)
}
