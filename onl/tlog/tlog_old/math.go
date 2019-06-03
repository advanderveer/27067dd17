package tlog

var log2t = [256]int{
	-1, 0, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, // 0

	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // 1

	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, // 2
	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, // 3

	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, // 4
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, // 5
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, // 6
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, // 7

	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 8
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 9
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 10
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 11
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 12
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 13
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 14
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, // 15
}

// log2u uses a lookup table to quickly calculate the base2 log of an uint64
// @credits https://github.com/cznic/mathutil/blob/eba54fb065b72d838b4b3481c6056998ead55e74/mathutil.go
func log2(n uint64) int {
	if b := n >> 56; b != 0 {
		return log2t[b] + 56
	}

	if b := n >> 48; b != 0 {
		return log2t[b] + 48
	}

	if b := n >> 40; b != 0 {
		return log2t[b] + 40
	}

	if b := n >> 32; b != 0 {
		return log2t[b] + 32
	}

	if b := n >> 24; b != 0 {
		return log2t[b] + 24
	}

	if b := n >> 16; b != 0 {
		return log2t[b] + 16
	}

	if b := n >> 8; b != 0 {
		return log2t[b] + 8
	}

	return log2t[n]
}

// decompose a number into components that are a power of 2
func pow2decompose(x uint64) (comps []uint64) {
	for i := uint64(1); i <= x; {
		if i&x != 0 {
			comps = append(comps, i)
		}

		i = i << 1
	}

	return
}
