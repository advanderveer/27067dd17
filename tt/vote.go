package tt

//Vote are small fixed-sized messages that saturate the network with hints to the
//longest chain
type Vote struct {
	Token []byte
	Proof []byte
	PK    []byte

	Tip ID
}
