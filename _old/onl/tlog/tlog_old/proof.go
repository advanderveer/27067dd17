package tlog

import "fmt"

type Siblings struct{}

type Proof2 []*Siblings

type h struct {
	L uint64
	K uint64
}

func (h h) String() string {
	return fmt.Sprintf("h(L=%d, K=%d)", h.L, h.K)
}

type level struct {
	Node  h
	Left  h
	Right h
}

func (l level) String() string {
	return fmt.Sprintf("level(h=%s, left=%s, right=%s)", l.Node, l.Left, l.Right)
}

func NewProof(i, N uint64) (p *Proof2) {

	//decompose into perfect balanced binary tries
	decomp := pow2decompose(N)

	//for each decomp iets peak as an absolute positioned hash
	var peaks []h
	var maxlvl uint64
	for _, nn := range decomp {
		hpeak := h{L: uint64(log2(nn)), K: N}

		//find the absolute K for each peak
		for L := uint64(0); L < hpeak.L; L++ {
			hpeak.K = hpeak.K / 2
		}

		fmt.Printf("peak n=%d: %s\n", nn, hpeak)
		peaks = append(peaks, hpeak)
		maxlvl = hpeak.L + 1
	}

	// find the necessary siblings
	width := N
	for L := uint64(0); L < maxlvl; L++ {
		l := level{Node: h{L: L + 1, K: i / 2}}

		//which sibling do we read for this level
		if i%2 == 0 {

			l.Left = h{L: L, K: i}
			l.Right = h{L: L, K: i + 1}

			// sibling = i + 1 //even, get right siblings

		} else {
			// sibling = i - 1 //odd, get left sibling

			l.Left = h{L: L, K: i - 1}
			l.Right = h{L: L, K: i}

		}

		off := l.Right.K >= width
		fmt.Printf("%s, off:%v\n", l, off)
		if off {
			l.Right = peaks[0]
			peaks = peaks[1:]
		}

		//at the next level, i is half of the current i
		i = i / 2

		// every level cuts the nr nodes in half
		width = width / 2
	}

	return
}

func Decompose(i, N uint64) {

	//decompose into perfect balanced binary tries
	decomp := pow2decompose(N)

	//for each decomp iets peak as an absolute positioned hash
	var peaks []h
	var maxlvl uint64
	for _, nn := range decomp {
		hpeak := h{L: uint64(log2(nn)), K: N}

		//find the absolute K for each peak
		for L := uint64(0); L < hpeak.L; L++ {
			hpeak.K = hpeak.K / 2
		}

		fmt.Printf("peak n=%d: %s\n", nn, hpeak)
		peaks = append(peaks, hpeak)
		maxlvl = hpeak.L + 1
	}

	var K uint64
	for L := maxlvl; L > 0; L-- {

		K = K * 2

		fmt.Println(L, K)
	}

}
