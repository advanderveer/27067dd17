package bchain

import "errors"

var (
	ErrBlockTicketInvalid    = errors.New("block ticket is invalid")
	ErrBlockSignatureInvalid = errors.New("block signature is invalid")
	ErrBlockRoundInvalid     = errors.New("block round is invalid")
	ErrBlockNotExist         = errors.New("block doesn't exist")
	ErrBlockExist            = errors.New("block already exists")

	ErrTipNotExist = errors.New("tip doesn't exist")
)
