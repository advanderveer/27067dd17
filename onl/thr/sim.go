package thr

import (
	"math/rand"

	"github.com/cockroachdb/apd"
)

// Filter will return a sub set of dist that passes
func Filter(c *apd.Context, dist []uint64, tlen int, f *apd.Decimal, rnd *rand.Rand) (passed []int) {
	var total uint64
	for _, stake := range dist {
		total += stake
	}

	var ok bool
	for i, stake := range dist {

		//roll a random token, lower is better
		t := make([]byte, tlen)
		_, err := rnd.Read(t)
		if err != nil {
			panic("thr/filter: " + err.Error())
		}

		//check if the roll is low enought
		_, _, ok = Thr(c, f, stake, total, t)
		if ok {
			passed = append(passed, i)
		}
	}

	return
}
