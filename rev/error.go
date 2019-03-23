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
)
