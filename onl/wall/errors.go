package wall

import "errors"

var (
	ErrTransferEmpty                 = errors.New("transfer had no inputs or outputs")
	ErrTransferIDInvalid             = errors.New("transfer id was invalid")
	ErrTransferSenderNotFundsOwner   = errors.New("transfer sender is not owner of all the referenced funds")
	ErrTransferOutputAmountInvalid   = errors.New("the output amount doesn't equal the amount referenced as input")
	ErrTransferTimeLockedOutput      = errors.New("the transfer's input tried to spend a time locked output too early")
	ErrTransferUsesUnspendableOutput = errors.New("the transfer tries to use an output that was already consumed in another transfer")

	ErrBlockIDInvalid             = errors.New("block id was invalid")
	ErrBlockVoteSignatureInvalid  = errors.New("the block's vote signature was invalid")
	ErrBlockTicketInvalid         = errors.New("the block ticket is invalid")
	ErrWitnessSignatureInvalid    = errors.New("witness signature is invalid")
	ErrBlockTimstampInPast        = errors.New("block's timestamp is in the past")
	ErrBlockRoundInPast           = errors.New("block's round is in the past")
	ErrBlockDepositNotSpendable   = errors.New("blocks's stake deposit is not spendable")
	ErrBlockVoterDoesntOwnDeposit = errors.New("block's stake deposit is not owned by voter")
	ErrBlockDepositNotLocked      = errors.New("blocks stake deposit was no longer locked")
	ErrBlockDepositLockedTooLong  = errors.New("block's stake deposit is locked for too long")
)
