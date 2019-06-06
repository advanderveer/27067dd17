package simulator

import "errors"

var (
	//ErrBlockNotExist is returned when the block doesn't exist
	ErrBlockNotExist = errors.New("block doesn't exist")
)
