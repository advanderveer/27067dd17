package onl

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid block signature")
	ErrInvalidToken     = errors.New("invalid token")
)
