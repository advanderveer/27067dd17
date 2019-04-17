package onl

import "errors"

var (
	ErrInvalidSignature        = errors.New("invalid block signature")
	ErrInvalidToken            = errors.New("invalid token")
	ErrBlockExist              = errors.New("block exists")
	ErrBlockNotExist           = errors.New("block doesn't exist")
	ErrZeroRound               = errors.New("round is zero")
	ErrFinalizedPrevNotInChain = errors.New("finalized prev not in chain")
	ErrStateReconstruction     = errors.New("failed to reconstruct state")
	ErrApplyConflict           = errors.New("conflict during apply")
)
