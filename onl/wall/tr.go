package wall

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/vrf"
)

// TrID is the transfer ID
type TrID [vrf.Size]byte

// OID uniquely references a transfer output
type OID [vrf.Size + 8]byte

// Tr returns the transaction this output is held in
func (r OID) Tr() (id TrID) {
	copy(id[:], r[:32])
	return
}

// Idx returns the output index the output is at
func (r OID) Idx() (i uint64) {
	return binary.BigEndian.Uint64(r[32:])
}

//Ref returns a new output reference
func Ref(trid TrID, i uint64) (ref OID) {
	copy(ref[:], trid[:])
	binary.BigEndian.PutUint64(ref[32:], i)
	return
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
	Inputs  []OID   //references to outputs the sender owns
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
		fields = append(fields, in[:])
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
// against. It take its main inspiration from the rules as set forth by Bitcoin
// as specified here: https://en.bitcoin.it/wiki/Protocol_rules#.22tx.22_messages
func (tr *Tr) Verify(coinbase bool, round uint64, utro *UTRO) (ok bool, err error) {
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
		out, ok := utro.Get(in)
		if !ok {
			return false, ErrTransferUsesUnspendableOutput
		}

		//check that the sender is the owner of the used funds
		if out.Receiver != tr.Sender {
			return false, ErrTransferSenderNotFundsOwner
		}

		//check if the round in which this input gets encoded is
		//larger then the minimum round the output unlocks
		if round <= out.UnlocksAfter {
			return false, ErrTransferTimeLockedOutput
		}

		inTotal += out.Amount
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
