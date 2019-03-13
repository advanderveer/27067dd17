package slot

import (
	"fmt"
	"strings"
)

var (
	//ErrPrevNotExist is returned if it was expected that a prev ref block would exist
	ErrPrevNotExist = fmt.Errorf("referenced 'prev' block doesn't exist")

	//ErrWrongRound is returned when a different round was expected
	ErrWrongRound = fmt.Errorf("unexpected round provided")

	//ErrPrevWrongRound is returned when a different round on the prev block was expected
	ErrPrevWrongRound = fmt.Errorf("unexpected prev round provided")

	//ErrProposeProof is returned when an invalid vrf construction was seen
	ErrProposeProof = fmt.Errorf("invalid proposer proof")

	//ErrNotarizeProof is returned when an invalid vrf construction was seen
	ErrNotarizeProof = fmt.Errorf("invalid notarizer proof")

	//ErrBroadcastClosed is returned when the broadcast shut down
	ErrBroadcastClosed = fmt.Errorf("broadcast closed down")

	//ErrUnknownMessage is returned by the engine when it ran into an unkown message
	ErrUnknownMessage = fmt.Errorf("read unkown message from broadcast")
)

//MsgError is returned when an error occurded due to a certain message
type MsgError struct {
	N uint64
	T MsgType
	E error
	M string
}

func (e MsgError) Error() string {
	return fmt.Sprintf("failed to %s on n=%d (type: %d): %v", e.M, e.N, e.T, e.E)
}

// ResolveErr is returend when we failed to handle after an out-of-order resolve
type ResolveErr []error

func (e ResolveErr) Error() string {
	var str []string
	for _, err := range e {
		str = append(str, err.Error())
	}

	return fmt.Sprintf("failed to resolve out-of-order messages: %v", strings.Join(str, ", "))
}
