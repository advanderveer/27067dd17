package thr

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/advanderveer/go-test"
	"github.com/cockroachdb/apd"
)

func TestFiltering(t *testing.T) {
	c := apd.BaseContext.WithPrecision(5)
	n := 10
	f := apd.New(999, -3)
	T := uint64(2000)
	r := rand.New(rand.NewSource(1))
	l := 32

	dist := Dist(n, T, r)
	passed := Filter(c, dist, l, f, r)
	test.Equals(t, 5, len(passed))
}

func TestMessageBound(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	c := apd.BaseContext.WithPrecision(5) //decimal calculation precision
	r := rand.New(rand.NewSource(3))      //seeded random source
	l := 32                               //nr of random bytes
	k := 10                               //nr of rounds per case

	for _, tc := range []struct {
		f *apd.Decimal //chance of passing threshold if one would own all the stake
		T uint64       //total stake in the system
		n int          //spread over this many members
	}{

		//the Ouroboras/Cdoe threshold threshold function makes it so that it doesn't matter if the
		//total stake is spread out over 2 members or over 2000. We can expect the
		//same absolute nr of identities per round that submit a block:
		{f: apd.New(999, -3), n: 2, T: 2000},    // 1.7000
		{f: apd.New(999, -3), n: 20, T: 2000},   // 5.4000
		{f: apd.New(999, -3), n: 200, T: 2000},  // 7.3000
		{f: apd.New(999, -3), n: 2000, T: 2000}, // 6.5000

		//the same holds true if the stake increases but the nr of members stays equal:
		{f: apd.New(999, -3), n: 20, T: 20},             // 5.4000
		{f: apd.New(999, -3), n: 20, T: 200},            // 5.9000
		{f: apd.New(999, -3), n: 20, T: 2000},           // 5.3000
		{f: apd.New(999, -3), n: 20, T: 20000},          // 5.0000
		{f: apd.New(999, -3), n: 20, T: math.MaxUint64}, // 4.9000

		//and it still gives sensible results if the total stake is very large
		{f: apd.New(999, -3), n: 20, T: math.MaxUint64},    // 5.9000
		{f: apd.New(999, -3), n: 200, T: math.MaxUint64},   // 5.8000
		{f: apd.New(999, -3), n: 2000, T: math.MaxUint64},  // 7.0000
		{f: apd.New(999, -3), n: 20000, T: math.MaxUint64}, // 7.8000
	} {

		t.Run(fmt.Sprintf("f: %s\t n: %d\t T: %d", tc.f, tc.n, tc.T), func(t *testing.T) {
			var total int
			for i := 0; i < k; i++ {
				dist := Dist(tc.n, tc.T, r)
				passed := Filter(c, dist, l, tc.f, r)
				t.Logf("\t%d", len(passed))
				total += len(passed)
			}

			t.Logf("average nr of members passed threshold per round: %.4f", float64(total)/float64(k))
		})
	}

}
