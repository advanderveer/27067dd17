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

	dist := Dist(n, T, 1, r)
	passed := Filter(c, dist, T, l, f, r)
	test.Equals(t, 5, len(passed))
}

func TestMessageBound(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	c := apd.BaseContext.WithPrecision(5) //decimal calculation precision
	l := 32                               //nr of random bytes
	k := 10                               //nr of rounds per case

	for _, tc := range []struct {
		f   *apd.Decimal // chance of passing threshold if one would own all the stake
		T   uint64       // total stake in the system
		n   int          // spread over this many members
		p   uint64       // total used/n of stake being distributed (or active)
		exp float64      // expected average
	}{

		//the Ouroboras/Cdoe threshold threshold function makes it so that it doesn't matter if the
		//total stake is spread out over 2 members or over 2000. We can expect the
		//same absolute nr of identities per round that submit a block:
		{f: apd.New(999, -3), p: 1, n: 2, T: 2000, exp: 1.7},
		{f: apd.New(999, -3), p: 1, n: 20, T: 2000, exp: 4.8},
		{f: apd.New(999, -3), p: 1, n: 200, T: 2000, exp: 5.7},
		{f: apd.New(999, -3), p: 1, n: 2000, T: 2000, exp: 7.8},

		//the same holds true if the stake increases but the nr of members stays equal.
		//all stake is spread out over all members
		{f: apd.New(999, -3), p: 1, n: 20, T: 20, exp: 5.4},
		{f: apd.New(999, -3), p: 1, n: 20, T: 200, exp: 6.5},
		{f: apd.New(999, -3), p: 1, n: 20, T: 2000, exp: 4.8},
		{f: apd.New(999, -3), p: 1, n: 20, T: 20000, exp: 5.4},
		{f: apd.New(999, -3), p: 1, n: 20, T: math.MaxUint64, exp: 5.7},

		// and it still gives sensible results if the total stake is very large
		{f: apd.New(999, -3), p: 1, n: 20, T: math.MaxUint64, exp: 5.7},
		{f: apd.New(999, -3), p: 1, n: 200, T: math.MaxUint64, exp: 6.3},
		{f: apd.New(999, -3), p: 1, n: 2000, T: math.MaxUint64, exp: 6.5},
		// {f: apd.New(999, -3), n: 20000, T: math.MaxUint64}, // 7.8000

		// if only a smaller portion of the total stake in the system is used for
		// staking it becomes very hard for everyone to pass the threshold.
		{f: apd.New(999, -3), p: 1, n: 20, T: 2000, exp: 4.8},
		{f: apd.New(99999, -5), p: 2, n: 20, T: 2000, exp: 4.5},
		{f: apd.New(999999999999999999, -18), p: 16, n: 20, T: 2000, exp: 2.1},
		{f: apd.New(999999999999999999, -18), p: 32, n: 20, T: 2000, exp: 1.3},
	} {

		t.Run(fmt.Sprintf("p: %d, f: %s\t n: %d\t T: %d", tc.p, tc.f, tc.n, tc.T), func(t *testing.T) {
			r := rand.New(rand.NewSource(3))

			var total int
			for i := 0; i < k; i++ {
				dist := Dist(tc.n, tc.T, tc.p, r)
				passed := Filter(c, dist, tc.T, l, tc.f, r)
				t.Logf("\t%d", len(passed))
				total += len(passed)
			}

			t.Logf("average nr of members passed threshold per round: %.4f", float64(total)/float64(k))
			test.Equals(t, tc.exp, float64(total)/float64(k))
		})
	}
}
