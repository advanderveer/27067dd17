package thr

import (
	"math"
	"math/rand"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestRandomUint64(t *testing.T) {
	test.Equals(t, uint64(0), Uint64n(2, rand.New(rand.NewSource(1))))
	test.Equals(t, uint64(1), Uint64n(2, rand.New(rand.NewSource(6))))
	test.Equals(t, uint64(5577006791947779410), Uint64n(math.MaxUint64, rand.New(rand.NewSource(1))))
}

func TestMaxUint64RandomDist(t *testing.T) {

	//distribute over 1 member
	d1 := Dist(1, math.MaxUint64, rand.New(rand.NewSource(1)))
	test.Equals(t, 1, len(d1))
	test.Equals(t, uint64(math.MaxUint64), d1[0])

	//distribute over 2 members
	d2 := Dist(2, math.MaxUint64, rand.New(rand.NewSource(1)))
	test.Equals(t, 2, len(d2))
	test.Equals(t, uint64(5416370213103200912), d2[0])
	test.Equals(t, uint64(13030373860606350703), d2[1])
	test.Equals(t, uint64(math.MaxUint64), d2[0]+d2[1])

	//distribute over many members
	d3 := Dist(1e6, math.MaxUint64, rand.New(rand.NewSource(1)))
	test.Equals(t, int(1e6), len(d3))

	//analyze distribution
	var total uint64
	var zeros []int
	for i, stake := range d3 {
		if stake == 0 {
			zeros = append(zeros, i)
		}
		total += stake
	}

	//sparse distriibution, some didn't get any stake
	test.Equals(t, 134957, len(zeros))

	//total should equal to the max
	test.Equals(t, uint64(math.MaxUint64), total)

	//average should be very close to MaxUint64/1m
	test.Equals(t, uint64(0), (uint64(math.MaxUint64)/uint64(1e6))-uint64(0x10c6f7a0b5ed))
}

func TestZeroDist(t *testing.T) {
	test.Equals(t, 0, len(Dist(0, math.MaxUint64, rand.New(rand.NewSource(1)))))

	d1 := Dist(2, 0, rand.New(rand.NewSource(1)))
	test.Equals(t, 2, len(d1))
	test.Equals(t, uint64(0), d1[0])
	test.Equals(t, uint64(0), d1[0])
}

func TestLowNrRandomDist(t *testing.T) {

	//the one member should get the one stake
	d1 := Dist(1, 1, rand.New(rand.NewSource(1)))
	test.Equals(t, 1, len(d1))
	test.Equals(t, uint64(1), d1[0])

	//not enough stake for both, should hand out to one
	d2 := Dist(2, 1, rand.New(rand.NewSource(1)))
	test.Equals(t, 2, len(d2))
	test.Equals(t, uint64(0), d2[0])
	test.Equals(t, uint64(1), d2[1])

	//spase distribution should still work
	d3 := Dist(1e5, 100, rand.New(rand.NewSource(1)))
	test.Equals(t, int(1e5), len(d3))
	var total uint64
	for _, stake := range d3 {
		total += stake
	}

	test.Equals(t, uint64(100), total)
}
