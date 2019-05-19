package wall

import (
	"fmt"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestTransferHashing(t *testing.T) {
	for i := 0; i < 10; i++ {

		//add each field and check that the hash changes
		tr := Tr{}
		test.Equals(t, "b393978842", fmt.Sprintf("%.5x", tr.Hash()))
		tr.ID[0] = 0x01
		test.Equals(t, "07810c11b9", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Proof[0] = 0x01
		test.Equals(t, "f769f5a2c6", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Sender[0] = 0x01
		test.Equals(t, "712974f9b3", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Inputs = append(tr.Inputs, TrIn{})
		test.Equals(t, "4d09841576", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Inputs[0].OutputIdx = 1
		test.Equals(t, "45f65f3253", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Inputs[0].OutputTr[0] = 0x01
		test.Equals(t, "b0d831bef1", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "cd90559b8f", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs[0].Amount = 1
		test.Equals(t, "11d1108b25", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs[0].Receiver[0] = 0x01
		test.Equals(t, "b1b2e0aecb", fmt.Sprintf("%.5x", tr.Hash()))

		//add extra inputs for testing consistency of the hashing
		tr.Inputs = append(tr.Inputs, TrIn{})
		test.Equals(t, "a5f4410a6e", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "b8ddca1046", fmt.Sprintf("%.5x", tr.Hash()))
	}
}

func TestTransferSigning(t *testing.T) {

	//unsigned transaction
	tr0 := &Tr{}
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", tr0.Sender.Bytes()))
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", tr0.ID[:]))
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", tr0.Proof[:]))

	id1 := NewIdentity([]byte{0x01})
	tr1 := id1.SignTransfer(&Tr{})
	test.Equals(t, "4762ad6415", fmt.Sprintf("%.5x", tr1.Sender.Bytes()))
	test.Equals(t, "30d0f4340f", fmt.Sprintf("%.5x", tr1.ID[:]))
	test.Equals(t, "043d044565", fmt.Sprintf("%.5x", tr1.Proof[:]))

	//signing with a different identity should yield other values
	id2 := NewIdentity([]byte{0x02})
	tr2 := id2.SignTransfer(tr1)
	test.Equals(t, "3a5a0c2134", fmt.Sprintf("%.5x", tr2.Sender.Bytes()))
	test.Equals(t, "f1aae682c9", fmt.Sprintf("%.5x", tr2.ID[:]))
	test.Equals(t, "42a7f18888", fmt.Sprintf("%.5x", tr2.Proof[:]))

	//resigning with the first identity should yield exactly the same
	id3 := NewIdentity([]byte{0x01})
	tr3 := id3.SignTransfer(tr1)
	test.Equals(t, tr1, tr3)
	test.Equals(t, tr1, tr3)
}

//a map that acts as a mock transfer reader
type state struct {
	blocks map[TrID]*Tr
	round  uint64
	spent  map[TrID][]uint64
}

func newState() *state {
	return &state{blocks: make(map[TrID]*Tr), spent: make(map[TrID][]uint64)}
}

func (st *state) CurrRound() uint64 {
	return st.round
}

func (st *state) HasBeenSpend(outtr TrID, outi uint64) (ok bool) {
	out, ok := st.spent[outtr]
	if !ok {
		return false
	}

	for _, i := range out {
		if i == outi {
			return true
		}
	}

	return false
}

func (st *state) ReadTr(id TrID) (tr *Tr, err error) {
	tr, ok := st.blocks[id]
	if !ok {
		return nil, fmt.Errorf("not such transfer")
	}

	return
}

func TestEmptyTransfer(t *testing.T) {
	s1 := newState()

	ok, err := (&Tr{}).Verify(false, s1)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)

	tr1 := &Tr{Inputs: []TrIn{{}}}
	ok, err = tr1.Verify(false, s1)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)

	tr2 := &Tr{Outputs: []TrOut{{}}}
	ok, err = tr2.Verify(false, s1)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)
}

func TestTransferVerification(t *testing.T) {
	st1 := newState()
	st1.round = 1

	id1 := NewIdentity([]byte{0x01})
	id2 := NewIdentity([]byte{0x02})

	tr0 := id2.SignTransfer(&Tr{
		Outputs: []TrOut{
			{Amount: 100, Receiver: id1.PublicKey()},
		},
	})

	tr1 := id1.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr0.ID, OutputIdx: 0},
		},
		Outputs: []TrOut{
			{Amount: 50, Receiver: id1.PublicKey()},
			{Amount: 20, Receiver: id1.PublicKey()},
			{Amount: 30, Receiver: id2.PublicKey()},
		},
	})

	//fault transfer: tries to spend outputs that are not owned by sender
	tr2 := id2.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr1.ID, OutputIdx: 0},
			{OutputTr: tr1.ID, OutputIdx: 1},
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.PublicKey()},
			{Amount: 35, Receiver: id2.PublicKey()},
		},
	})

	//faulty transfer: not all inputs are spend
	tr3 := id1.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr1.ID, OutputIdx: 0},
			{OutputTr: tr1.ID, OutputIdx: 1},
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.PublicKey()},
		},
	})

	//a transfer to ones-self that an output that can only be consumed
	//with a transfer at a certain height.
	trlock := id2.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr1.ID, OutputIdx: 2},
		},
		Outputs: []TrOut{
			{Amount: 30, Receiver: id2.PublicKey(), UnlocksAfter: 400},
		},
	})

	trunlock := id2.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: trlock.ID, OutputIdx: 0},
		},
		Outputs: []TrOut{
			{Amount: 30, Receiver: id2.PublicKey()},
		},
	})

	t.Run("test verfication", func(t *testing.T) {

		//verify coinbase 100 to id1
		test.OkEquals(t, true)(tr0.Verify(true, st1))

		//transfer 70 to self, and 30 to id2
		st1.blocks[tr0.ID] = tr0
		test.OkEquals(t, true)(tr1.Verify(false, st1))

		//faulty transfer should fail
		st1.blocks[tr1.ID] = tr1
		ok, err := tr2.Verify(false, st1)
		test.Equals(t, ErrTransferSenderNotFundsOwner, err)
		test.Equals(t, false, ok)

		ok, err = tr3.Verify(false, st1)
		test.Equals(t, ErrTransferOutputAmountInvalid, err)
		test.Equals(t, false, ok)

		t.Run("invalid signature", func(t *testing.T) {
			tr4 := &Tr{Inputs: []TrIn{{}}, Outputs: []TrOut{{}}}
			tr4 = id1.SignTransfer(tr4)
			tr4.ID[0] = 0x01

			ok, err := tr4.Verify(false, st1)
			test.Equals(t, ErrTransferIDInvalid, err)
			test.Equals(t, false, ok)
		})

		t.Run("invalid proof", func(t *testing.T) {
			tr5 := &Tr{Inputs: []TrIn{{}}, Outputs: []TrOut{{}}}
			tr5 = id1.SignTransfer(tr5)
			tr5.Proof[0] = 0x01

			ok, err := tr5.Verify(false, st1)
			test.Equals(t, ErrTransferIDInvalid, err)
			test.Equals(t, false, ok)
		})
	})

	t.Run("time locking verify", func(t *testing.T) {
		test.OkEquals(t, true)(trlock.Verify(false, st1))
		st1.blocks[trlock.ID] = trlock

		//an output with a time lock cannot be spend until the chain is at least
		//at some round in the future.
		ok, err := trunlock.Verify(false, st1)
		test.Equals(t, ErrTransferTimeLockedOutput, err)
		test.Equals(t, false, ok)

		//should unlock fine at a later round
		st1.round = 401
		test.OkEquals(t, true)(trunlock.Verify(false, st1))
	})

	t.Run("test double spending verify", func(t *testing.T) {
		test.OkEquals(t, true)(tr1.Verify(false, st1))

		st1.spent[tr0.ID] = append(st1.spent[tr0.ID], 0)

		ok, err := tr1.Verify(false, st1)
		test.Equals(t, ErrTansferDoubleSpends, err)
		test.Equals(t, false, ok)
	})

}
