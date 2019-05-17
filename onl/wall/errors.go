package wall

import "errors"

var (
	ErrTransferEmpty               = errors.New("transfer had no inputs or outputs")
	ErrTransferSignatureInvalid    = errors.New("transfer signature invalid")
	ErrTransferSenderNotFundsOwner = errors.New("transfer sender is not owner of all the referenced funds")
	ErrTransferOutputAmountInvalid = errors.New("the output amount doesn't equal the amount referenced as input")
)
