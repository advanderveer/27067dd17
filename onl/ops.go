package onl

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/vrf/ed25519"
)

//OpID uniquely identifies an operation
type OpID [sha256.Size]byte

//Bytes returns the underlying bytes as a slice
func (id OpID) Bytes() []byte { return id[:] }

// Op is an operation that is encoded in blocks and totally ordered
// by the consensus protocol
type Op interface {
	Execute(state interface{}) (err error)
	Hash() OpID
}

// JoinOp will enable the signer to propose blocks and take part in the protocol
// and earning minting rewards. It deposits part of the identity's stake as
// collateral and it must commit to a PK for vrf token generation.
type JoinOp struct {
	//@TODO Nonce uint64 //increments on every op, must prevents replay attacks
	//@TODO figure out of the nonce is a kinda-lock that we don't want for the full namespace
	//but just for the account balance portion, because that is what is being manipulated
	//@TODO TreasuryPK PK                          //the identity that will receive the deposit

	Deposit   uint64                      //how much stake the identity is willing to bet
	TokenPK   PK                          //the VRF public key that the identity commits to
	Identity  PK                          //the identity that will sign new blocks
	Signature [ed25519.SignatureSize]byte //signed by the identity that wants to join
}

//Hash the operation
func (op *JoinOp) Hash() OpID {
	depb := make([]byte, 8)
	binary.BigEndian.PutUint64(depb, op.Deposit)

	return sha256.Sum256(bytes.Join([][]byte{
		depb,
		op.TokenPK[:],
		op.Identity[:],
	}, nil))
}

//Execute the operation
func (op *JoinOp) Execute(state interface{}) (err error) {
	//@TODO check basic syntax
	//@TODO check nonce with in the identities current state
	//@TODO check the signature, must be signed by the identity that deposits
	//@TODO check the balance, must be more or equal to the deposit amount
	//@TODO check that the treasuryPK exists

	return nil
}

//@TODO add operations below
// //LeaveOp will prevent the identity from being allowed to propose any blocks. In
// //contrary to the JoinOp it can be signed and broadcasted by everyone that believes
// //there is significant evidence that a member failed to provide stake often enough
// //to finalized blocks.
// type LeaveOp struct {
// 	//@TODO what fields?
// }
//
// //Execute the operation
// func (op *LeaveOp) Execute(state interface{}) (err error) {
// 	//@TODO what logic
// 	return nil
// }
//
// //Hash the operation
// func (op *LeaveOp) Hash() OpID {
// 	return sha256.Sum256(bytes.Join([][]byte{
// 		//@TODO add fields
// 	}, nil))
// }
//
// //TransferOp transfers currency amounts between identities
// type TransferOp struct {
// 	//@TODO what fields?
// }
//
// //Execute the operation
// func (op *TransferOp) Execute(state interface{}) (err error) {
// 	//@TODO what logic
// 	return nil
// }
//
// //Hash the operation
// func (op *TransferOp) Hash() OpID {
// 	return sha256.Sum256(bytes.Join([][]byte{
// 		//@TODO add fields
// 	}, nil))
// }
