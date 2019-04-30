package onl

import "errors"

var (
	ErrInvalidSignature      = errors.New("invalid block signature")
	ErrInvalidToken          = errors.New("invalid token")
	ErrBlockExist            = errors.New("block exists")
	ErrBlockNotExist         = errors.New("block doesn't exist")
	ErrZeroRound             = errors.New("round is zero")
	ErrStableNotInChain      = errors.New("stable random source not in chain")
	ErrStateReconstruction   = errors.New("failed to reconstruct state")
	ErrApplyConflict         = errors.New("conflict during apply")
	ErrNoTokenPK             = errors.New("no token pk committed")
	ErrZeroRank              = errors.New("blocks has zero rank")
	ErrRoundNrNotAfterPrev   = errors.New("round number wasn't after the prev's round number")
	ErrTimestampNotAfterPrev = errors.New("timestamp didn't come after prev's timestamp")
	ErrNotWeighted           = errors.New("block's round is not weighted yet")
	ErrAppendConflict        = errors.New("concurrent append caused conflict")
)
