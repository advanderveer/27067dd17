package wall

import "errors"

var (
	ErrTransferEmpty               = errors.New("transfer had no inputs or outputs")
	ErrTransferIDInvalid           = errors.New("transfer id was invalid")
	ErrTransferSenderNotFundsOwner = errors.New("transfer sender is not owner of all the referenced funds")
	ErrTransferOutputAmountInvalid = errors.New("the output amount doesn't equal the amount referenced as input")
	ErrTransferTimeLockedOutput    = errors.New("the transfer's input tried to spend a time locked output too early")
	ErrTansferDoubleSpends         = errors.New("the transfer tries to use an output that was already consumed in another transfer")
)
