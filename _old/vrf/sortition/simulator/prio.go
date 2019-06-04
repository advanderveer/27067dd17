package simulator

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"

	"github.com/advanderveer/27067dd17/vrf"
)

// DrawPriority a pseudo-random lottery that is more likely to draw a higher priority
// if more weight is provided. Besides the priority it returns the weight index that
// resulted in the highest priority and a proof that can be used to verify that we
// indeed drew randomly.
func DrawPriority(sk *[vrf.SecretKeySize]byte, seed []byte, weight uint64) (prio *big.Int, widx uint64, ticket, proof []byte) {

	//the vrf provides us with random bytes that can be verified by others it
	//gives everyone its own space to draw from. Else everyone would draw the
	//same value on widx 0.
	ticket, proof = vrf.Prove(seed, sk)

	prio = big.NewInt(0)
	curr := new(big.Int)

	//we are allowed to draw for every index into our weight, the more weight we
	//have the more we're allowed to draw.
	for i := uint64(0); i < weight; i++ {
		DeterminePriority(ticket, i, curr)

		//if our new priority is higher then the old one, we consider it better
		if curr.Cmp(prio) > 0 {
			prio.Set(curr)
			widx = i
		}
	}

	return
}

// DeterminePriority sets the provided priority to the number determinisitally
// determined by the ticket and weight index.
func DeterminePriority(ticket []byte, widx uint64, prio *big.Int) {
	idxb := make([]byte, 8)
	binary.BigEndian.PutUint64(idxb, widx)

	//@TODO we not need a cryptographic hash, we just need something with a
	//a uniform distribution. Faster is better, and it should not be worthwile
	//to create specialized hardware to speed this up.
	h := sha256.Sum256(bytes.Join([][]byte{ticket, idxb}, nil))
	prio.SetBytes(h[:])
	return
}

// VerifyPriority will verify if the priority is based on a random draw and that
// the ticket and widx indeed result in the provided priority number
func VerifyPriority(pk, seed, ticket, proof []byte, widx uint64, prio *big.Int) (ok bool) {
	if !vrf.Verify(pk, seed, ticket, proof) {
		return //invalid ticket
	}

	realPrio := new(big.Int)
	DeterminePriority(ticket, widx, realPrio)

	if realPrio.Cmp(prio) != 0 {
		return false //given ticket and widx doesn't result into the given priority
	}

	return true
}
