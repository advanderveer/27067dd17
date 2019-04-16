package onl

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid block signature")
	ErrInvalidToken     = errors.New("invalid token")
	ErrBlockExist       = errors.New("block exists")
	ErrBlockNotExist    = errors.New("block doesn't exist")
	ErrZeroRound        = errors.New("round is zero")
)
