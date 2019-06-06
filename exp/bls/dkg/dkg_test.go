package bks

import (
	"fmt"
	"testing"

	test "github.com/advanderveer/go-test"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	dkg "go.dedis.ch/kyber/v3/share/dkg/rabin"
)

// Secure Distributed Key Generation for Discrete-Log Based Cryptosystems:
// https://link.springer.com/content/pdf/10.1007/3-540-48910-X_21.pdf
//  "We assume that an adversary, A, can corrupt up to t of the n players in
//  the network, for any value of t < n/2 "

func TestBKS(t *testing.T) {
	suite := edwards25519.NewBlakeSHA256Ed25519() // cipher suite

	n := 5        //number of participants
	tn := n/2 + 1 //threshold number (@TODO can be n/2?)

	//step 1: init keys
	ltpks := make([]kyber.Point, n)  //long term public keys
	ltsks := make([]kyber.Scalar, n) //long term private keys
	for i := 0; i < n; i++ {
		ltsks[i] = suite.Scalar().Pick(suite.RandomStream())
		ltpks[i] = suite.Point().Mul(ltsks[i], nil)
	}

	//@TODO broadcast the long term public keys

	//step 2: init generator
	var err error
	gens := make([]*dkg.DistKeyGenerator, n)
	for i := 0; i < n; i++ {
		gens[i], err = dkg.NewDistKeyGenerator(suite, ltsks[i], ltpks, tn)
		test.Ok(t, err)
	}

	//step 3: run deals
	for i := 0; i < n; i++ {
		deals, err := gens[i].Deals()
		test.Ok(t, err)

		for j, dd := range deals {
			fmt.Println(i, "sends to:", j, ltpks[j], "deal:", dd)
		}

		_ = deals //number of deals can be large?
		// fmt.Println(len(deals))
	}

	_ = gens
	// var share *dkg.DistKeyShare
	// fmt.Println("abc")

}
