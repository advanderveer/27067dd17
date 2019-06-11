package tlog

import (
	"math"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestPow2Decompose(t *testing.T) {
	test.Equals(t, []uint64{1, 4, 8}, pow2decompose(13))
	test.Equals(t, []uint64{1, 4}, pow2decompose(5))
	test.Equals(t, []uint64{2}, pow2decompose(2))
	test.Equals(t, []uint64{4}, pow2decompose(4))

	t.Run("decompose 0", func(t *testing.T) {
		test.Equals(t, []uint64(nil), pow2decompose(0))
	})

	t.Run("decompose max size", func(t *testing.T) {
		test.Equals(t, []uint64{0x1,
			0x2,
			0x4,
			0x8,
			0x10,
			0x20,
			0x40,
			0x80,
			0x100,
			0x200,
			0x400,
			0x800,
			0x1000,
			0x2000,
			0x4000,
			0x8000,
			0x10000,
			0x20000,
			0x40000,
			0x80000,
			0x100000,
			0x200000,
			0x400000,
			0x800000,
			0x1000000,
			0x2000000,
			0x4000000,
			0x8000000,
			0x10000000,
			0x20000000,
			0x40000000,
			0x80000000,
			0x100000000,
			0x200000000,
			0x400000000,
			0x800000000,
			0x1000000000,
			0x2000000000,
			0x4000000000,
			0x8000000000,
			0x10000000000,
			0x20000000000,
			0x40000000000,
			0x80000000000,
			0x100000000000,
			0x200000000000,
			0x400000000000,
			0x800000000000,
			0x1000000000000,
			0x2000000000000,
			0x4000000000000,
			0x8000000000000,
			0x10000000000000,
			0x20000000000000,
			0x40000000000000,
			0x80000000000000,
			0x100000000000000,
			0x200000000000000,
			0x400000000000000,
			0x800000000000000,
			0x1000000000000000,
			0x2000000000000000,
			0x4000000000000000},
			pow2decompose(math.MaxInt64))
	})
}

func TestLog2(t *testing.T) {
	var i int
	for _, n := range pow2decompose(math.MaxInt64) {
		ii := log2(n)
		if ii == 0 {
			continue
		}
		test.Equals(t, 1, ii-i)
		i = ii
	}

	test.Equals(t, i, 62)
}