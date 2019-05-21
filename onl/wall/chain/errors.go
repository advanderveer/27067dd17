package chain

import "errors"

var (
	ErrBlockNotExist     = errors.New("block doesn't exist")
	ErrBlockExists       = errors.New("block already exist")
	ErrDepositNotFound   = errors.New("couldnt find a deposit for the identity")
	ErrVoterAlreadyVoted = errors.New("voter already voted in the round")
)
