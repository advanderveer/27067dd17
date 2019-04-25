package engine_test

import (
	"context"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/go-test"
)

type testClock uint64

func (c testClock) ReadUs() uint64 { return uint64(c) }

func testEngine(t *testing.T, osc *engine.MemOscillator, idn *onl.Identity, genf ...func(kv *onl.KV)) (bc *engine.MemBroadcast, e *engine.Engine, clean func()) {
	store, cleanstore := onl.TempBadgerStore()

	chain, _, err := onl.NewChain(store, genf...)
	test.Ok(t, err)

	bc = engine.NewMemBroadcast(100)

	e = engine.New(os.Stderr, bc, osc.Pulse(), idn, chain)
	return bc, e, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		test.Ok(t, e.Shutdown(ctx))
		cleanstore()
	}
}

func TestEngineEmptyRounds(t *testing.T) {
	idn := onl.NewIdentity([]byte{0x01})
	osc := engine.NewMemOscillator()
	_, e1, clean1 := testEngine(t, osc, idn)
	for i := 0; i < 100; i++ {
		osc.Fire()
	}

	clean1() //read any remaining pulses before closing down
	test.Equals(t, uint64(101), e1.Round())
}

// Test that an engine can progress through key value writes on its own.
func TestEngineWritingByItself(t *testing.T) {
	nWrites := uint64(50)

	idn := onl.NewIdentity([]byte{0x01})
	osc := engine.NewMemOscillator()
	_, e1, clean1 := testEngine(t, osc, idn, func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn.PK(), 1)
		kv.DepositStake(idn.PK(), 1, idn.TokenPK())
	})

	//write to the engine async
	go func() {
		ctx := context.Background()
		for j := uint64(0); j < nWrites; j++ {
			time.Sleep(time.Millisecond)
			test.Ok(t, e1.Update(ctx, func(kv *onl.KV) {
				kb := make([]byte, 8)
				binary.LittleEndian.PutUint64(kb, j)

				kv.Set(kb, []byte{0x01})
			}))
		}
	}()

	//fire some rounds
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 3)
		osc.Fire()
	}

	//wait for rounds to be wrapped up
	clean1()

	//should now read all 50 puts on itself
	test.Ok(t, e1.View(func(kv *onl.KV) {
		for j := uint64(0); j < nWrites; j++ {
			kb := make([]byte, 8)
			binary.LittleEndian.PutUint64(kb, j)

			//@TODO (#2) this fails onces in a full moont
			test.Equals(t, []byte{0x01}, kv.Get(kb))

		}
	}))

	//@TODO assert memory pool size to be very empty
	//@TODO assert out-of-order to be empty
}

// Test that one n engine writes and other engines replicate without writing themselves
func TestEngineWriterReplication(t *testing.T) {

	//testing variables
	nWrites := uint64(20)

	//global memory oscillator
	osc := engine.NewMemOscillator()

	//one identity will be writing blocks
	idn1 := onl.NewIdentity([]byte{0x01})
	genf := func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK())
	}

	idn3 := onl.NewIdentity([]byte{0x03})
	idn2 := onl.NewIdentity([]byte{0x02})

	//create engines with genesis status
	bc1, e1, clean1 := testEngine(t, osc, idn1, genf)
	bc2, e2, clean2 := testEngine(t, osc, idn2, genf)
	bc3, e3, clean3 := testEngine(t, osc, idn3, genf)

	//create a ring topology bc1->bc2->bc3->bc1
	bc1.To(bc2)
	bc2.To(bc3)
	bc3.To(bc1)

	//start writing thread
	go func() {
		ctx := context.Background()
		for j := uint64(0); j < nWrites; j++ {
			time.Sleep(time.Millisecond)
			test.Ok(t, e1.Update(ctx, func(kv *onl.KV) {
				kb := make([]byte, 8)
				binary.LittleEndian.PutUint64(kb, j)

				kv.Set(kb, []byte{0x01})
			}))
		}
	}()

	//fire some rounds
	for i := 0; i < 50; i++ {
		time.Sleep(time.Millisecond * 3)
		osc.Fire()
	}

	//wrap it all up
	clean1()
	clean2()
	clean3()

	//check if repication was successfull
	test.Ok(t, e2.View(func(kv *onl.KV) {
		for j := uint64(0); j < nWrites; j++ {
			kb := make([]byte, 8)
			binary.LittleEndian.PutUint64(kb, j)
			test.Equals(t, []byte{0x01}, kv.Get(kb))
		}
	}))

	test.Ok(t, e3.View(func(kv *onl.KV) {
		for j := uint64(0); j < nWrites; j++ {
			kb := make([]byte, 8)
			binary.LittleEndian.PutUint64(kb, j)
			test.Equals(t, []byte{0x01}, kv.Get(kb))
		}
	}))

	//@TODO asset if .LastError() is empty
}

//Tests whether 3 engines writing come to consensus
func TestEngine3WriterConsensus(t *testing.T) {

	//testing variables
	nWrites := uint64(20)
	nRounds := 30
	roundEvery := time.Millisecond * 3
	writeEvery := time.Millisecond

	//global memory oscillator
	osc := engine.NewMemOscillator()

	//setup the writing identities
	idn1 := onl.NewIdentity([]byte{0x01})
	idn2 := onl.NewIdentity([]byte{0x02})
	idn3 := onl.NewIdentity([]byte{0x03})

	//create coinbases and deposits
	genf := func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK())
		kv.CoinbaseTransfer(idn2.PK(), 1)
		kv.DepositStake(idn2.PK(), 1, idn2.TokenPK())
		kv.CoinbaseTransfer(idn3.PK(), 1)
		kv.DepositStake(idn3.PK(), 1, idn3.TokenPK())
	}

	//create engines with genesis state
	bc1, e1, clean1 := testEngine(t, osc, idn1, genf)
	bc2, e2, clean2 := testEngine(t, osc, idn2, genf)
	bc3, e3, clean3 := testEngine(t, osc, idn3, genf)

	//create a ring topology bc1->bc2->bc3->bc1
	bc1.To(bc2)
	bc2.To(bc3)
	bc3.To(bc1)

	//start writing thread, high contention
	go func() {
		ctx := context.Background()
		for j := uint64(0); j < nWrites; j++ {
			time.Sleep(writeEvery)

			kb := make([]byte, 8)
			binary.LittleEndian.PutUint64(kb, j)

			test.Ok(t, e1.Update(ctx, func(kv *onl.KV) { kv.Set(kb, []byte{0x01}) }))
			test.Ok(t, e2.Update(ctx, func(kv *onl.KV) { kv.Set(kb, []byte{0x02}) }))
			test.Ok(t, e3.Update(ctx, func(kv *onl.KV) { kv.Set(kb, []byte{0x03}) }))
		}
	}()

	//fire some rounds
	for i := 0; i < nRounds; i++ {
		time.Sleep(roundEvery)
		osc.Fire()
	}

	//wrap it all up
	clean1()
	clean2()
	clean3()

	//check if repication was successfull
	results := make([][3]byte, nWrites)
	for j := uint64(0); j < nWrites; j++ {
		kb := make([]byte, 8)
		binary.LittleEndian.PutUint64(kb, j)

		test.Ok(t, e1.View(func(kv *onl.KV) { results[j][0] = kv.Get(kb)[0] }))
		test.Ok(t, e2.View(func(kv *onl.KV) { results[j][1] = kv.Get(kb)[0] }))
		test.Ok(t, e3.View(func(kv *onl.KV) { results[j][2] = kv.Get(kb)[0] }))
	}

	//print and test results results
	_ = results
	// for i, r := range results {
	// 	fmt.Println(i, r)
	// 	if r[0] != r[1] || r[2] != r[0] {
	// 		t.Errorf("didn't reach consensus on write %d: %v", i, r)
	// 	}
	// }

	//@TODO we expect all kv stores to be the same
	//@TODO we expect far less writes per block after some rounds as the should
	//      all of them should conflict

}
