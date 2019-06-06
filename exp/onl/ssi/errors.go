package ssi

import "errors"

var (
	//ErrConflict is returned when the a conclicting transaction was detected
	ErrConflict = errors.New("conflict: other transactions committed concurrently and modified this transaction read data")
)
