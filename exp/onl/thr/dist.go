package thr

import (
	"math"
	"math/rand"
)

// Uint64n returns, as a uint64, a pseudo-random number in [0,n).
// It is guaranteed more uniform than taking a Source value mod n
// for any n that is not a power of 2. Taken from the golang/exp repository:
// https://github.com/golang/exp/blob/14dda7b62fcdb381624aaca63b04df07203856d4/rand/rand.go
func Uint64n(n uint64, rnd *rand.Rand) uint64 {
	if n&(n-1) == 0 { // n is power of two, can mask
		if n == 0 {
			panic("invalid argument to Uint64n")
		}
		return rnd.Uint64() & (n - 1)
	}

	// If n does not divide v, to avoid bias we must not use
	// a v that is within maxUint64%n of the top of the range.
	v := rnd.Uint64()
	if v > math.MaxUint64-n { // Fast check.
		ceiling := math.MaxUint64 - math.MaxUint64%n
		for v >= ceiling {
			v = rnd.Uint64()
		}
	}

	return v % n
}

//Dist returns a random distribution of the total stake over the provided nr of
//members.
func Dist(n int, stake, part uint64, rnd *rand.Rand) (dist []uint64) {
	dist = make([]uint64, n)

	if n < 1 {
		return //zero length dist asked
	}

	if part > 1 {
		stake = stake / part //part can specify to only hand out some of the total stake
	} else if part == 0 {
		stake = 0 //part 0 means no stake used
	}

	step := stake / uint64(n)
	if step < 2 {
		step = 2 //the rand pick only ever returns non-zero if n > 1
	}

	for {
		pull := Uint64n(step, rnd)

		if pull >= stake {
			dist[rnd.Intn(len(dist))] += stake //remainder
			break
		}

		stake -= pull
		dist[rnd.Intn(len(dist))] += pull
	}

	return
}
