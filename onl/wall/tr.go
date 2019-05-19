package wall

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/advanderveer/27067dd17/vrf"
)

// TrID is the transfer ID
type TrID [vrf.Size]byte

// TrIn describes the inputs to a transfer
type TrIn struct {
	OutputTr  TrID   //transfer that holds the output we consume
	OutputIdx uint64 //the index of the output in the transfer
}

// TrOut describes the outputs to a transfer
type TrOut struct {
	Amount       uint64 //amount we're transferring
	Receiver     PK     //the receiver of this amount
	UnlocksAfter uint64 //the round after which the funds in this output unlock
}

// Tr is a (currency) transfer, the same as a bitcoin transaction but this
// term is so overloaded in our case that it is given a different name
type Tr struct {
	Inputs  []TrIn  //references to outputs the sender owns
	Outputs []TrOut //amounts send to receivers

	// Sender signs off on the fact that this transfer consumes the referenced
	// output and the mising of the amount into the outputs for the receivers.
	// The ID is verfiably unique and based on the transfer content so acts as
	// both a signature and an ID
	Sender PK
	Proof  [vrf.ProofSize]byte
	ID     TrID
}

// Hash the transfer
func (tr *Tr) Hash() (h [32]byte) {
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
	fields = append(fields, tr.ID[:])
	fields = append(fields, tr.Proof[:])

	return sha256.Sum256(bytes.Join(fields, nil))
}

// Verify a transfer considering the current state of the tip it will be minted
// against. It take sits main inspiration from the rules as set forth by Bitcoin
// as specified here: https://en.bitcoin.it/wiki/Protocol_rules#.22tx.22_messages
func (tr *Tr) Verify(coinbase bool, s State) (ok bool, err error) {
	if len(tr.Outputs) < 1 {
		return false, ErrTransferEmpty
	}

	if !coinbase && len(tr.Inputs) < 1 {
		return false, ErrTransferEmpty
	}

	//empty the elements that represent the signature
	sig := tr.ID
	proof := tr.Proof
	tr.ID = [vrf.Size]byte{}
	tr.Proof = [vrf.ProofSize]byte{}

	//verify the signature
	h := tr.Hash()
	ok = vrf.Verify(tr.Sender[:], h[:], sig[:], proof[:])
	if !ok {
		return false, ErrTransferIDInvalid
	}

	//reset the signature elements
	tr.ID = sig
	tr.Proof = proof

	//the sum of the outputs must be equal to the sum of the inputs
	//@TODO prevent total summing to wrap around max uint64
	var inTotal uint64
	for _, in := range tr.Inputs {

		//read the transfer that is supposed to hold the funds for this input
		refTr, err := s.ReadTr(in.OutputTr)
		if err != nil {
			return false, fmt.Errorf("failed to read the referenced transfer: %v", err)
		}

		//read the specific output
		idx := int(in.OutputIdx)
		if len(refTr.Outputs) < (idx + 1) {
			return false, fmt.Errorf("referenced transfer didn't have enough outputs")
		}

		//check that the sender is the owner of the used funds
		if refTr.Outputs[idx].Receiver != tr.Sender {
			return false, ErrTransferSenderNotFundsOwner
		}

		//check if the output hasn't been spend already
		if s.HasBeenSpend(in.OutputTr, in.OutputIdx) {
			return false, ErrTansferDoubleSpends
		}

		//check if the round in which this input gets encoded is
		//larger then the minimum round the output unlocks
		if s.CurrRound() <= refTr.Outputs[idx].UnlocksAfter {
			return false, ErrTransferTimeLockedOutput
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
