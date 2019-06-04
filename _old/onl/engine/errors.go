package engine

import "errors"

var (
	ErrAlreadyInPool         = errors.New("write is already in pool")
	ErrInvalidWriteSignature = errors.New("write signature is invalid")
)
