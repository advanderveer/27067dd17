package rev

import "errors"

var (
	//ErrProposalTokenInvalid is returned when we expected a proposal token to be valid
	ErrProposalTokenInvalid = errors.New("proposal token is invalid")

	//ErrProposalHasNoBlock is returend when it was expected that a proposal has a block
	ErrProposalHasNoBlock = errors.New("proposal didn't encode a block")

	//ErrProposalHasNoWitness is returned when the proposal was expected to have a witness
	ErrProposalHasNoWitness = errors.New("proposal didn't have witness")

	//ErrProposalWitnessUnknown is returned when a witness is not known locally
	ErrProposalWitnessUnknown = errors.New("proposal witness unkown")

	//ErrNotEnoughWitness is returned when a proposal did not provide enough witness
	ErrNotEnoughWitness = errors.New("nog enough witnesses in proposal")

	//ErrPrevProposalNotFound is returned when we expected a proposal for the prev block
	ErrPrevProposalNotFound = errors.New("no proposal that contained the prev block")

	//ErrPrevProposalNotTopWitness is returned when we expected that the proposals top witness holds the prev block
	ErrPrevProposalNotTopWitness = errors.New("the proposal that holds the prev block is not the top witness")
)
