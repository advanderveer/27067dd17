package tt

import (
	"bytes"
	"crypto/sha256"
	"math"
	"math/big"
	"sort"
)

//ID uniquely identifies a block
type ID [sha256.Size]byte

var (
	//NilID is an empty ID
	NilID = ID{}
)

//Block contains the data and can be submitted by the fist member to solve the
//vote puzzle and encodes the data the protocol reach consensus over
type Block struct {
	Prev  ID
	Data  []byte
	Votes []Vote
}

//B creates a new block
func B(prev ID, d []byte) *Block {
	return &Block{prev, d, nil}
}

//Hash the content of the block into a unique identifier
func (b *Block) Hash() (id ID) {
	var vhashs [][]byte
	for _, v := range b.Votes {
		vid := v.Hash()
		vhashs = append(vhashs, vid[:])
	}

	return ID(sha256.Sum256(bytes.Join([][]byte{
		b.Prev[:],
		b.Data,
		bytes.Join(vhashs, nil),
	}, nil)))
}

//Mine will attempt al different set combinations of the provided votes in order
//to find a block hash that satisfies the provided difficulty requirement
func (b *Block) Mine(target *big.Int, votes map[VID]Vote) (ok bool) {
	var vids []VID
	for vid := range votes {
		vids = append(vids, vid) //serialize into a slice
	}

	//sort by byte order
	sort.Slice(vids, func(i, j int) bool {
		return bytes.Compare(vids[i][:], vids[j][:]) < 0
	})

	//loop over all variations
	for _, comb := range PowerSet(vids...) {
		if comb == nil {
			continue //the empty variation is skipped
		}

		b.Votes = make([]Vote, len(comb))
		for i, id := range comb {
			b.Votes[i] = votes[id]
		}

		//hash and check if we're past the required threshold with these sets of
		//votes
		bid := b.Hash()
		score := big.NewInt(0).SetBytes(bid[:])
		if score.Cmp(target) < 0 {
			return true
		}
	}

	return
}

func (b *Block) Verify() {

	//no double votes (same hash)
	//miner can not be the same person as voter (different PK)
	//only distict voters (not same pk)
	//check vote syntax
	//passes the diffuclty check

}

//PowerSet of a slice of vote ids returns all possibe set combinations
func PowerSet(original ...VID) [][]VID {
	powerSetSize := int(math.Pow(2, float64(len(original))))
	result := make([][]VID, 0, powerSetSize)

	var index int
	for index < powerSetSize {
		var subSet []VID

		for j, elem := range original {
			if index&(1<<uint(j)) > 0 {
				subSet = append(subSet, elem)
			}
		}
		result = append(result, subSet)
		index++
	}

	return result
}
