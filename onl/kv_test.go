package onl_test

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/go-test"
)

func TestKVOperations(t *testing.T) {
	idn1 := onl.NewIdentity([]byte{0x01})
	idn2 := onl.NewIdentity([]byte{0x02})
	idn3 := onl.NewIdentity([]byte{0x03})

	st1, _ := onl.NewState(nil)
	test.Ok(t, st1.Apply(st1.Update(func(kv *onl.KV) {

		//try to read non-existing account
		test.Equals(t, uint64(0), kv.AccountBalance(idn1.PK()))

		//try to deposit on non-existing account
		kv.DepositStake(idn1.PK(), 9, []byte{0x01})
		stake, tpk := kv.ReadStake(idn1.PK())
		test.Equals(t, uint64(0), stake)
		test.Equals(t, []byte(nil), tpk)

		//write coinbase amount
		kv.CoinbaseTransfer(idn1.PK(), 90)
		kv.CoinbaseTransfer(idn1.PK(), 10)

		//try to read it right away
		test.Equals(t, uint64(100), kv.AccountBalance(idn1.PK()))

		//try to deposit more then balance has
		kv.DepositStake(idn1.PK(), 999, []byte{0x01})
		stake, tpk = kv.ReadStake(idn1.PK())
		test.Equals(t, uint64(0), stake)
		test.Equals(t, []byte(nil), tpk)

		//deposit currency as stake
		kv.DepositStake(idn1.PK(), 9, []byte{0x01})
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK())

		//read the deposit
		stake, tpk = kv.ReadStake(idn1.PK())
		test.Equals(t, uint64(10), stake)
		test.Equals(t, idn1.TokenPK(), tpk)

		//read the new account balance
		test.Equals(t, uint64(90), kv.AccountBalance(idn1.PK()))

		//transfer to some other account
		kv.TransferCurrency(idn1.PK(), idn2.PK(), 50)

		//read balance of receiving account
		test.Equals(t, uint64(40), kv.AccountBalance(idn1.PK()))
		test.Equals(t, uint64(50), kv.AccountBalance(idn2.PK()))

		//tranfer while send doesn't have enough balance shouldn't change anything
		kv.TransferCurrency(idn1.PK(), idn2.PK(), 150)
		test.Equals(t, uint64(50), kv.AccountBalance(idn2.PK()))

		//transferring from non-existing shouldn't change anything
		kv.TransferCurrency(idn3.PK(), idn2.PK(), 1)
		test.Equals(t, uint64(50), kv.AccountBalance(idn2.PK()))

		//tranfer to itself shouldn't do anthing
		kv.TransferCurrency(idn2.PK(), idn2.PK(), 150)
		test.Equals(t, uint64(50), kv.AccountBalance(idn2.PK()))

	}), false))
}

func TestKVOperationsFuzzing(t *testing.T) {
	nIdentities := 10
	startBalance := uint64(100)
	nOps := 1000
	maxTransfer := 25
	depositFreq := 100

	//create identities
	idns := make([]*onl.Identity, nIdentities)
	for i := 0; i < nIdentities; i++ {
		idb := make([]byte, 8)
		binary.BigEndian.PutUint64(idb, uint64(i))
		idns[i] = onl.NewIdentity(idb)
	}

	//setup blank state
	state, err := onl.NewState(nil)
	test.Ok(t, err)

	//initialize with coinbase transfers
	for _, idn := range idns {
		test.Ok(t, state.Apply(state.Update(func(kv *onl.KV) {
			kv.CoinbaseTransfer(idn.PK(), startBalance)
		}), false))
	}

	//a slice of writes created concurrently on a single state
	writes := make(chan *onl.Write, nOps)
	var wg sync.WaitGroup
	for i := 0; i < nOps; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			//perform random ops
			writes <- state.Update(func(kv *onl.KV) {
				amount := rand.Intn(maxTransfer)
				from := idns[rand.Intn(nIdentities)]

				if i%depositFreq == 0 {
					kv.DepositStake(from.PK(), uint64(amount), nil)
				} else {
					to := idns[rand.Intn(nIdentities)]
					kv.TransferCurrency(from.PK(), to.PK(), uint64(amount))
				}
			})

		}(i)
	}

	wg.Wait()

	//applying them shouldn't invalidate the system
	var i int
	for w := range writes {
		err := state.Apply(w, false)
		if err != nil && err != onl.ErrApplyConflict {
			t.Fatalf("unexpected apply error: %v", err)
		}

		i++
		if i >= nOps {
			break
		}
	}

	//check the total amount of currency in the system
	var total uint64
	for _, idn := range idns {
		state.Read(func(kv *onl.KV) {
			bal := kv.AccountBalance(idn.PK())
			stake, _ := kv.ReadStake(idn.PK())

			total += stake
			total += bal
		})
	}

	//should be the total capital put in
	test.Equals(t, uint64(nIdentities)*startBalance, total)
}
