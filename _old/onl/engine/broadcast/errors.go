package broadcast

import "errors"

var (
	ErrClosed      = errors.New("closed broadcast")
	ErrPeerRefused = errors.New("peer refused connection")
)
