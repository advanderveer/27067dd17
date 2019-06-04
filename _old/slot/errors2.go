package slot

import (
	"fmt"
)

var (
	//ErrInvalidBlockNilPrev is returnd during validation to indicate the prev is empty
	ErrInvalidBlockNilPrev = fmt.Errorf("block is invalid, prev reference is empty")

	ErrInvalidProposalTokenLen = fmt.Errorf("proposal token length is invalid")

	ErrInvalidProposalProofLen = fmt.Errorf("proposal proof length is invalid")

	ErrInvalidProposalPKLen = fmt.Errorf("proposal public key length is invalid")

	ErrInvalidProposalNoBlock = fmt.Errorf("proposal has not block")

	ErrInvalidVoteTokenLen = fmt.Errorf("vote token length is invalid")

	ErrInvalidVoteProofLen = fmt.Errorf("vote proof length is invalid")

	ErrInvalidVotePKLen = fmt.Errorf("vote public key length is invalid")

	ErrInvalidVoteNoProposal = fmt.Errorf("vote has not proposal")
)
