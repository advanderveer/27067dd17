package slot

import "fmt"

var (
	//ErrPrevNotExist is returned if it was expected that a prev ref block would exist
	ErrPrevNotExist = fmt.Errorf("referenced 'prev' block doesn't exist")

	//ErrWrongRound is returned when a different round was expected
	ErrWrongRound = fmt.Errorf("unexpected round provided")

	//ErrPrevWrongRound is returned when a different round on the prev block was expected
	ErrPrevWrongRound = fmt.Errorf("unexpected prev round provided")

	//ErrInvalidVRF is returned when an invalid vrf construction was seen
	ErrInvalidVRF = fmt.Errorf("invalid VRF encountered")
)
