package bls

import (
	"fmt"
	"testing"

	test "github.com/advanderveer/go-test"
	"github.com/dfinity/go-dfinity-crypto/bls"
)

//@TODO reference implementation: https://herumi.github.io/bls-wasm/bls-demo.html
//@TODO https://github.com/dfinity/random-beacon

func TestBasicSigning(t *testing.T) {
	var sk bls.SecretKey    //32 byte random bytes
	sk.SetByCSPRNG()        //Set by a Crypto Secure Pseudo Random Number Generator
	pk := sk.GetPublicKey() //64 bytes public
	msg := "hello, world"

	// proof-of-possession (PoP)
	// https://blog.dash.org/secret-sharing-and-threshold-signatures-with-bls-954d1587b5f
	sig := sk.Sign(msg)
	sigb := sig.Serialize()
	sig.Deserialize(sigb)
	test.Equals(t, true, sig.Verify(pk, msg))

	if sigb[0] == 0x01 {
		sigb[0] = 0x02
	} else {
		sigb[0] = 0x01
	}

	sig.Deserialize(sigb)
	test.Equals(t, false, sig.Verify(pk, msg))

	fmt.Printf("%d, %s\n", len(sk.GetLittleEndian()), sk.GetHexString())
	fmt.Printf("%d, %s\n", len(pk.Serialize()), pk.GetHexString())
}

// GROUP MEMBERS INDEPENDENTLY SIGN THE MESSAGE TO CREATE “SIGNATURE SHARES”. A
// THRESHOLD NUMBER ARE COMBINED TO CREATE THE OUTPUT SIGNATURE

// Given the seed data, any group member can sign a signature share
// Everyone can combine these shares into a valid signature once enough shares
// have been gathered.
// The group has a public key that can be used to verify this signature

// func (a Addr) ID() bls.ID {
// 	var fr bls.Fr
// 	fr.SetHashOf(a[:])
// 	var id bls.ID
// 	err := id.SetLittleEndian(fr.Serialize())
// 	if err != nil {
// 		// should not happen
// 		panic(err)
// 	}
//
// 	return id
// }

func TestThresholdSignatures(t *testing.T) {
	msg := "hello, world"

	var sk bls.SecretKey            //32 byte random bytes
	sk.SetByCSPRNG()                //Set by a Crypto Secure Pseudo Random Number Generator
	msk := sk.GetMasterSecretKey(3) //create threshold keys, min=3
	mpk := sk.GetPublicKey()

	sig1 := sk.Sign(msg)
	test.Equals(t, true, sig1.Verify(mpk, msg))
	fmt.Println(sig1.GetHexString())

	idTbl := []byte{5, 7, 8, 9, 10} //create shares
	n := len(idTbl)

	sks := make([]bls.SecretKey, n)
	sigs := make([]bls.Sign, n)
	ids := make([]bls.ID, n)
	for i := 0; i < n; i++ {
		err := ids[i].SetLittleEndian([]byte{idTbl[i], 0, 0, 0, 0, 0})
		test.Ok(t, err)

		err = sks[i].Set(msk, &ids[i])
		if err != nil {
			t.Error(err)
		}

		sigs[i] = *sks[i].Sign(msg)
	}

	var sig2 bls.Sign
	err := sig2.Recover(sigs[2:5], ids[2:5]) //with 3 or more of these we can recover the group signature
	test.Ok(t, err)
	test.Equals(t, sig1.Serialize(), sig2.Serialize())
	test.Equals(t, true, sig2.Verify(mpk, msg)) //verifies

	var sig3 bls.Sign
	err = sig3.Recover(sigs[0:3], ids[0:3]) //or another subset
	test.Ok(t, err)
	test.Equals(t, true, sig3.Verify(mpk, msg)) //verifies

}
