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
		test.Equals(t, "2ea9ab9198", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Signature[0] = 0x01
		test.Equals(t, "932f773767", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Sender[0] = 0x01
		test.Equals(t, "26ce35972c", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Inputs = append(tr.Inputs, TrIn{})
		test.Equals(t, "ca22c15233", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Inputs[0].OutputIdx = 1
		test.Equals(t, "4846aee817", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Inputs[0].OutputTr[0] = 0x01
		test.Equals(t, "8c2270ea3a", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "5d8ce67cc5", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Outputs[0].Amount = 1
		test.Equals(t, "778f411320", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Outputs[0].Receiver[0] = 0x01
		test.Equals(t, "75d44a4119", fmt.Sprintf("%.5x", tr.Hash().Bytes()))

		//add extra inputs for testing consistency of the hashing
		tr.Inputs = append(tr.Inputs, TrIn{})
		test.Equals(t, "0f091da2d8", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
		tr.Outputs = append(tr.Outputs, TrOut{})
		test.Equals(t, "3909f0d0a9", fmt.Sprintf("%.5x", tr.Hash().Bytes()))
	}
}

func TestTransferSigning(t *testing.T) {

	//unsigned transaction
	tr0 := &Tr{}
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", tr0.Sender.Bytes()))
	test.Equals(t, "0000000000", fmt.Sprintf("%.5x", tr0.Signature[:]))

	id1 := NewIdentity([]byte{0x01})
	tr1 := id1.SignTransfer(&Tr{})
	test.Equals(t, "cecc1507dc", fmt.Sprintf("%.5x", tr1.Sender.Bytes()))
	test.Equals(t, "0e42d11968", fmt.Sprintf("%.5x", tr1.Signature[:]))

	//signing with a different identity should yield other values
	id2 := NewIdentity([]byte{0x02})
	tr2 := id2.SignTransfer(tr1)
	test.Equals(t, "6b79c57e6a", fmt.Sprintf("%.5x", tr2.Sender.Bytes()))
	test.Equals(t, "15650fa5ee", fmt.Sprintf("%.5x", tr2.Signature[:]))

	//resigning with the first identity should yield exactly the same
	id3 := NewIdentity([]byte{0x01})
	tr3 := id3.SignTransfer(tr1)
	test.Equals(t, tr1, tr3)
	test.Equals(t, tr1, tr3)
}

//a map that acts as a mock transfer reader
type trmap map[TrID]*Tr

func (trr trmap) ReadTr(id TrID) (tr *Tr, err error) {
	tr, ok := trr[id]
	if !ok {
		return nil, fmt.Errorf("not such transfer")
	}

	return
}

func TestEmptyTransfer(t *testing.T) {
	trr := make(trmap)

	ok, err := (&Tr{}).Verify(false, trr)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)

	tr1 := &Tr{}
	tr1.Inputs = append(tr1.Inputs, TrIn{})
	ok, err = tr1.Verify(false, trr)
	test.Equals(t, ErrTransferEmpty, err)
	test.Equals(t, false, ok)
}

func TestTransferVerification(t *testing.T) {
	trr := make(trmap)
	id1 := NewIdentity([]byte{0x01})
	id2 := NewIdentity([]byte{0x02})

	tr0 := id2.SignTransfer(&Tr{
		Outputs: []TrOut{
			{Amount: 100, Receiver: id1.SignPK()},
		},
	})

	tr1 := id1.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr0.Hash(), OutputIdx: 0},
		},
		Outputs: []TrOut{
			{Amount: 50, Receiver: id1.SignPK()},
			{Amount: 20, Receiver: id1.SignPK()},
			{Amount: 30, Receiver: id2.SignPK()},
		},
	})

	//fault transfer, tries to spend outputs that are not owned by sender
	tr2 := id2.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr1.Hash(), OutputIdx: 0},
			{OutputTr: tr1.Hash(), OutputIdx: 1},
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.SignPK()},
			{Amount: 35, Receiver: id2.SignPK()},
		},
	})

	//faulty transfer, not all inputs are spend
	tr3 := id1.SignTransfer(&Tr{
		Inputs: []TrIn{
			{OutputTr: tr1.Hash(), OutputIdx: 0},
			{OutputTr: tr1.Hash(), OutputIdx: 1},
		},
		Outputs: []TrOut{
			{Amount: 35, Receiver: id1.SignPK()},
		},
	})

	t.Run("test verfication", func(t *testing.T) {

		//verify coinbase 100 to id1
		test.OkEquals(t, true)(tr0.Verify(true, trr))

		//transfer 70 to self, and 30 to id2
		trr[tr0.Hash()] = tr0
		test.OkEquals(t, true)(tr1.Verify(false, trr))

		//faulty transfer should fail
		trr[tr1.Hash()] = tr1
		ok, err := tr2.Verify(false, trr)
		test.Equals(t, ErrTransferSenderNotFundsOwner, err)
		test.Equals(t, false, ok)

		ok, err = tr3.Verify(false, trr)
		test.Equals(t, ErrTransferOutputAmountInvalid, err)
		test.Equals(t, false, ok)

		t.Run("invalid signature", func(t *testing.T) {
			tr4 := &Tr{Inputs: []TrIn{{}}, Outputs: []TrOut{{}}}
			tr4 = id1.SignTransfer(tr4)
			tr4.Signature[0] = 0x01

			ok, err := tr4.Verify(false, trr)
			test.Equals(t, ErrTransferSignatureInvalid, err)
			test.Equals(t, false, ok)
		})
	})

}
