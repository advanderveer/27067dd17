package engine_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/27067dd17/onl/engine/clock"
	"github.com/advanderveer/go-test"
)

func drawPNG(t *testing.T, e *engine.Engine, name string) {
	f, err := os.Create(name)
	test.Ok(t, err)
	defer f.Close()

	buf := bytes.NewBuffer(nil)
	test.Ok(t, e.Draw(buf))

	cmd := exec.Command("dot", "-Tpng")
	cmd.Stdin = buf
	cmd.Stdout = f
	test.Ok(t, cmd.Run())
}

func testEngine(t *testing.T, osc *clock.MemOscillator, idn *onl.Identity, genf ...func(kv *onl.KV)) (bc *broadcast.Mem, e *engine.Engine, clean func()) {
	store, cleanstore := onl.TempBadgerStore()

	chain, _, err := onl.NewChain(store, 0, genf...)
	test.Ok(t, err)

	bc = broadcast.NewMem(100)

	e = engine.New(os.Stderr, bc, osc.Clock(), idn, chain)
	return bc, e, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		test.Ok(t, e.Shutdown(ctx))
		cleanstore()
	}
}

func TestEngineEmptyRounds(t *testing.T) {
	idn := onl.NewIdentity([]byte{0x01})
	osc := clock.NewMemOscillator()
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
	osc := clock.NewMemOscillator()
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
	osc := clock.NewMemOscillator()

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

func TestEngine3WriteConsensusStepByStep(t *testing.T) {
	nWrites := uint64(15) //doesn't fit into one block
	nRounds := 3

	//global memory oscillator
	osc := clock.NewMemOscillator()

	//setup the writing identities
	idn1 := onl.NewIdentity([]byte{0x01})
	idn1.SetName("e1")
	idn2 := onl.NewIdentity([]byte{0x02})
	idn2.SetName("e2")
	idn3 := onl.NewIdentity([]byte{0x03})
	idn3.SetName("e3")

	//create coinbases and deposits
	genf := func(kv *onl.KV) {
		kv.CoinbaseTransfer(idn1.PK(), 1)
		kv.DepositStake(idn1.PK(), 1, idn1.TokenPK())
		kv.CoinbaseTransfer(idn2.PK(), 1)
		kv.DepositStake(idn2.PK(), 1, idn2.TokenPK())
		kv.CoinbaseTransfer(idn3.PK(), 1)
		kv.DepositStake(idn3.PK(), 1, idn3.TokenPK())
	}

	//create engines with genesis state, starting in round 1
	bc1, e1, clean1 := testEngine(t, osc, idn1, genf)
	bc2, e2, clean2 := testEngine(t, osc, idn2, genf)
	bc3, e3, clean3 := testEngine(t, osc, idn3, genf)

	//create a ring topology bc1->bc2->bc3->bc1
	bc1.To(bc2)
	bc2.To(bc3)
	bc3.To(bc1)

	//write to mempool one-time
	for j := uint64(0); j < nWrites; j++ {
		kb := make([]byte, 8)
		binary.LittleEndian.PutUint64(kb, j)

		test.Ok(t, e1.Update(context.Background(), func(kv *onl.KV) { kv.Set(kb, append([]byte{0x01}, kv.Get(kb)...)) }))
		test.Ok(t, e2.Update(context.Background(), func(kv *onl.KV) { kv.Set(kb, append([]byte{0x02}, kv.Get(kb)...)) }))
		test.Ok(t, e3.Update(context.Background(), func(kv *onl.KV) { kv.Set(kb, append([]byte{0x03}, kv.Get(kb)...)) }))
	}

	//wait for writes to have spread to all members
	time.Sleep(time.Millisecond * 500)

	//fire rounds
	for i := 0; i < nRounds; i++ {
		osc.Fire()
		time.Sleep(time.Millisecond * 500)
	}

	//more settle down time?
	time.Sleep(time.Millisecond * 500)

	//wrap it all up
	clean1()
	clean2()
	clean3()

	//draw
	drawPNG(t, e1, "e1.png")
	drawPNG(t, e2, "e2.png")
	drawPNG(t, e3, "e3.png")

	//should all have the same tip
	test.Equals(t, e1.Tip(), e2.Tip())
	test.Equals(t, e1.Tip(), e3.Tip())

	//check if replication was successfull
	results := make([][3]byte, nWrites)
	for j := uint64(0); j < nWrites; j++ {
		kb := make([]byte, 8)
		binary.LittleEndian.PutUint64(kb, j)

		test.Ok(t, e1.View(func(kv *onl.KV) { results[j][0] = kv.Get(kb)[0] }))
		test.Ok(t, e2.View(func(kv *onl.KV) { results[j][1] = kv.Get(kb)[0] }))
		test.Ok(t, e3.View(func(kv *onl.KV) { results[j][2] = kv.Get(kb)[0] }))
	}

	//print and test results
	for i, r := range results {
		if r[0] != r[1] || r[2] != r[0] {
			t.Errorf("didn't reach consensus on write %d: %v", i, r)
		}
	}
}
