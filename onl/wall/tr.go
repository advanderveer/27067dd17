package wall

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

// TrID is the transfer ID
type TrID [32]byte

// Bytes returns the id as a byte slice
func (id TrID) Bytes() []byte { return id[:] }

// TrIn describes the inputs to a transfer
type TrIn struct {
	OutputTr  TrID   //transfer that holds the output we consume
	OutputIdx uint64 //the index of the output in the transfer
}

// TrOut describes the outputs to a transfer
type TrOut struct {
	Amount   uint64 //amount we're transferring
	Receiver SignPK //the receiver of this amount
}

// Tr is a (currency) transfer, the same as a bitcoin transaction but this
// term is so overloaded in our case that it is given a different name
type Tr struct {
	Inputs  []TrIn  //references to outputs the sender owns
	Outputs []TrOut //amounts send to receivers

	// Sender signs off on the fact that this transfer consumes the referenced
	// output and the mising of the amount into the outputs for the receivers.
	Sender    SignPK
	Signature [64]byte
}

// Hash the transfer
func (tr *Tr) Hash() (id TrID) {
	var fields [][]byte

	for _, in := range tr.Inputs {
		fields = append(fields, in.OutputTr[:])
		idx := make([]byte, 8)
		binary.BigEndian.PutUint64(idx, in.OutputIdx)
		fields = append(fields, idx)
	}

	for _, out := range tr.Outputs {
		fields = append(fields, out.Receiver[:])
		amount := make([]byte, 8)
		binary.BigEndian.PutUint64(amount, out.Amount)
		fields = append(fields, amount)
	}

	fields = append(fields, tr.Sender[:])
	fields = append(fields, tr.Signature[:])

	return TrID(sha256.Sum256(bytes.Join(fields, nil)))
}

//TrReader provides transfer reading
type TrReader interface {
	ReadTr(id TrID) (tr *Tr, err error)
}

// Verify a transfer
func (tr *Tr) Verify(coinbase bool, trr TrReader) (ok bool, err error) {
	if len(tr.Outputs) < 1 {
		return false, ErrTransferEmpty
	}

	if !coinbase && len(tr.Inputs) < 1 {
		return false, ErrTransferEmpty
	}

	//the signature was made from hasing without the signature
	sig := tr.Signature
	tr.Signature = [64]byte{}

	//verify the signature
	pk := [32]byte(tr.Sender)
	ok = ed25519.Verify(&pk, tr.Hash().Bytes(), &sig)
	if !ok {
		return false, ErrTransferSignatureInvalid
	}

	//reset the signature
	tr.Signature = sig

	//the sum of the outputs must be equal to the sum of the inputs
	var inTotal uint64
	for _, in := range tr.Inputs {

		refTr, err := trr.ReadTr(in.OutputTr)
		if err != nil {
			return false, fmt.Errorf("failed to read the referenced transfer: %v", err)
		}

		idx := int(in.OutputIdx)
		if len(refTr.Outputs) < (idx + 1) {
			return false, fmt.Errorf("referenced transfer didn't have enough outputs")
		}

		if refTr.Outputs[idx].Receiver != tr.Sender {
			return false, ErrTransferSenderNotFundsOwner
		}

		inTotal += refTr.Outputs[idx].Amount
	}

	var outTotal uint64
	for _, out := range tr.Outputs {
		outTotal += out.Amount
	}

	if !coinbase && outTotal != inTotal {
		return false, ErrTransferOutputAmountInvalid
	}

	return
}
