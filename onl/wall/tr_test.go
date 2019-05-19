package wall

import (
	"fmt"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestTransferRef(t *testing.T) {
	tr1 := NewIdentity([]byte{0x02}).SignTransfer(&Tr{})
	outid1 := Ref(tr1.ID, 1)

	test.Equals(t, tr1.ID, outid1.Tr())
	test.Equals(t, uint64(1), outid1.Idx())
}

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
		tr.Inputs = append(tr.Inputs, OID{})
		test.Equals(t, "4d09841576", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Inputs[0][0] = 0x01
		test.Equals(t, "f87e962f7f", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "f61edfa9e6", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs[0].Amount = 1
		test.Equals(t, "1db8c51c42", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs[0].Receiver[0] = 0x01
		test.Equals(t, "90152fa697", fmt.Sprintf("%.5x", tr.Hash()))

		//add extra inputs for testing consistency of the hashing
		tr.Inputs = append(tr.Inputs, OID{})
		test.Equals(t, "7dc7a16a16", fmt.Sprintf("%.5x", tr.Hash()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "a049d3d0cc", fmt.Sprintf("%.5x", tr.Hash()))
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

func TestEmptyTransfer(t *testing.T) {
	utro := NewUTRO()

	ok, err := (&Tr{}).Verify(false, 0, utro)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)

	tr1 := &Tr{Inputs: []OID{{}}}
	ok, err = tr1.Verify(false, 0, utro)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)

	tr2 := &Tr{Outputs: []TrOut{{}}}
	ok, err = tr2.Verify(false, 0, utro)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)
}

func TestTransferVerification(t *testing.T) {
	utro := NewUTRO()

	id1 := NewIdentity([]byte{0x01})
	id2 := NewIdentity([]byte{0x02})

	tr0 := id2.SignTransfer(&Tr{
		Outputs: []TrOut{
			{Amount: 100, Receiver: id1.PublicKey()},
		},
	})

	tr1 := id1.SignTransfer(&Tr{
		Inputs: []OID{
			// {OutputTr: tr0.ID, OutputIdx: 0},
			Ref(tr0.ID, 0),
		},
		Outputs: []TrOut{
			{Amount: 50, Receiver: id1.PublicKey()},
			{Amount: 20, Receiver: id1.PublicKey()},
			{Amount: 30, Receiver: id2.PublicKey()},
		},
	})

	//fault transfer: tries to spend outputs that are not owned by sender
	tr2 := id2.SignTransfer(&Tr{
		Inputs: []OID{
			Ref(tr1.ID, 0),
			Ref(tr0.ID, 1),
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.PublicKey()},
			{Amount: 35, Receiver: id2.PublicKey()},
		},
	})

	//faulty transfer: not all inputs are spend
	tr3 := id1.SignTransfer(&Tr{
		Inputs: []OID{
			Ref(tr1.ID, 0),
			Ref(tr1.ID, 1),
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.PublicKey()},
		},
	})

	//a transfer to ones-self that an output that can only be consumed
	//with a transfer at a certain height.
	trlock := id2.SignTransfer(&Tr{
		Inputs: []OID{
			Ref(tr1.ID, 2),
		},
		Outputs: []TrOut{
			{Amount: 30, Receiver: id2.PublicKey(), UnlocksAfter: 400},
		},
	})

	trunlock := id2.SignTransfer(&Tr{
		Inputs: []OID{
			Ref(trlock.ID, 0),
		},
		Outputs: []TrOut{
			{Amount: 30, Receiver: id2.PublicKey()},
		},
	})

	t.Run("test verfication", func(t *testing.T) {

		//verify coinbase 100 to id1
		test.OkEquals(t, true)(tr0.Verify(true, 1, utro))

		//transfer 70 to self, and 30 to id2
		utro.Put(Ref(tr0.ID, 0), tr0.Outputs[0])
		test.OkEquals(t, true)(tr1.Verify(false, 1, utro))

		//faulty transfer should fail
		utro.Put(Ref(tr1.ID, 0), tr1.Outputs[0])
		utro.Put(Ref(tr1.ID, 1), tr1.Outputs[1])
		utro.Put(Ref(tr1.ID, 2), tr1.Outputs[2])

		ok, err := tr2.Verify(false, 1, utro)
		test.Equals(t, ErrTransferSenderNotFundsOwner, err)
		test.Equals(t, false, ok)

		ok, err = tr3.Verify(false, 1, utro)
		test.Equals(t, ErrTransferOutputAmountInvalid, err)
		test.Equals(t, false, ok)

		t.Run("invalid signature", func(t *testing.T) {
			tr4 := &Tr{Inputs: []OID{{}}, Outputs: []TrOut{{}}}
			tr4 = id1.SignTransfer(tr4)
			tr4.ID[0] = 0x01

			ok, err := tr4.Verify(false, 1, utro)
			test.Equals(t, ErrTransferIDInvalid, err)
			test.Equals(t, false, ok)
		})

		t.Run("invalid proof", func(t *testing.T) {
			tr5 := &Tr{Inputs: []OID{{}}, Outputs: []TrOut{{}}}
			tr5 = id1.SignTransfer(tr5)
			tr5.Proof[0] = 0x01

			ok, err := tr5.Verify(false, 1, utro)
			test.Equals(t, ErrTransferIDInvalid, err)
			test.Equals(t, false, ok)
		})
	})

	t.Run("time locking verify", func(t *testing.T) {
		test.OkEquals(t, true)(trlock.Verify(false, 1, utro))

		utro.Put(Ref(trlock.ID, 0), trlock.Outputs[0])

		//an output with a time lock cannot be spend until the chain is at least
		//at some round in the future.
		ok, err := trunlock.Verify(false, 1, utro)
		test.Equals(t, ErrTransferTimeLockedOutput, err)
		test.Equals(t, false, ok)

		//should unlock fine at a later round
		test.OkEquals(t, true)(trunlock.Verify(false, 401, utro))
	})

	t.Run("test double spending verify", func(t *testing.T) {
		test.OkEquals(t, true)(tr1.Verify(false, 1, utro))

		utro.Del(Ref(tr0.ID, 0))

		ok, err := tr1.Verify(false, 1, utro)
		test.Equals(t, ErrTransferUsesUnspendableOutput, err)
		test.Equals(t, false, ok)
	})

}
