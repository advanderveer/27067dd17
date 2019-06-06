package slot

import "encoding/binary"

//Seed function calculates the VRF seed of a new block
func Seed(prevb *Block, round uint64) (seed []byte) {
	seed = make([]byte, len(prevb.Ticket))
	copy(seed[:], prevb.Ticket[:])
	seed = append(make([]byte, 8), seed...)
	binary.LittleEndian.PutUint64(seed, round)
	return
}
