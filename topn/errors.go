package topn

import "errors"

var (
	ErrFutureRound = errors.New("round is in the future")

	ErrZeroRound = errors.New("block for round 0")

	ErrBlockExist = errors.New("block already exists")

	ErrDoublePropose = errors.New("identity already proposed")

	ErrBlockNotExist = errors.New("block doesn't exist")

	ErrPrevNotExist = errors.New("prev block doesn't exist")

	ErrNonIncreasingRound = errors.New("round not an increment from previous round")

	ErrInvalidToken = errors.New("invalid token")

	ErrZeroRank = errors.New("blocks has zero rank")

	ErrRankTooLow = errors.New("block ranks too low")
)
