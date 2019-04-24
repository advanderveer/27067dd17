package engine

//Pulse provides an interface for the system on a synchronized pulse
type Pulse interface {
	Next() (err error)
	Close() (err error)
}
