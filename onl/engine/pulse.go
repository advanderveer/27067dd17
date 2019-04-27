package engine

//Clock provides an interface for the system to a numbered timestamped pulse
type Clock interface {
	Round() (round uint64)
	Next() (round, ts uint64, err error)
	Close() (err error)
}
