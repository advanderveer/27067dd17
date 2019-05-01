package onl

//Stakes describes the stake distribution as observed by a member
type Stakes struct {

	//Sum of all stake that is deposited in the ancestory of this block plus what
	//is deposited in this block
	Sum uint64

	//Votes hold the current voting state of a block
	Votes map[PK]uint64
}

//NewStakes initializes a stakes struct
func NewStakes(sum uint64) *Stakes {
	return &Stakes{Sum: sum, Votes: make(map[PK]uint64)}
}

//Finalization returns a measure of finalization with a maximum of 1.0 where
//everyone unanimously voted indirectly on this block.
func (stk *Stakes) Finalization() (f float64) {
	var tot float64
	for _, stake := range stk.Votes {
		tot += float64(stake)
	}

	if stk.Sum == 0 {
		panic("encountered block with a 0 deposit sum")
	}

	return tot / float64(stk.Sum)
}
