package chain

import "errors"

var (
	ErrBlockNotExist   = errors.New("block doesn't exist")
	ErrDepositNotFound = errors.New("couldnt find a deposit for the identity")
)
