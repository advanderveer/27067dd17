package onl_test

import (
	"fmt"
	"testing"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/ssi"
	"github.com/advanderveer/go-test"
)

func TestTxOpHashing(t *testing.T) {
	w1 := &onl.Write{TxData: &ssi.TxData{ReadRows: make(ssi.KeySet), WriteRows: make(ssi.KeyChangeSet)}}
	test.Equals(t, "5b6fb58e", fmt.Sprintf("%.4x", w1.Hash()))

	w1.TimeCommit = 1
	test.Equals(t, "c416782a", fmt.Sprintf("%.4x", w1.Hash()))

	w1.TimeStart = 1
	test.Equals(t, "16be14a7", fmt.Sprintf("%.4x", w1.Hash()))

	w1.ReadRows.Add([]byte{0x01})
	test.Equals(t, "0aaa1111", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x02})
	test.Equals(t, "6960f8ef", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x03})
	test.Equals(t, "e899f492", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x03}) //shouldn't change anything
	test.Equals(t, "e899f492", fmt.Sprintf("%.4x", w1.Hash()))

	w1.Nonce[0] = 0x01
	test.Equals(t, "69ac6f45", fmt.Sprintf("%.4x", w1.Hash()))

	w1.PK[0] = 0x01
	test.Equals(t, "7c5ac829", fmt.Sprintf("%.4x", w1.Hash()))

	idn1 := onl.NewIdentity([]byte{0x01})
	t.Run("signature check", func(t *testing.T) {
		test.Equals(t, false, w1.VerifySignature())

		w1.PK = idn1.PK()  //add pk first
		idn1.SignWrite(w1) //then sign

		test.Equals(t, true, w1.VerifySignature())
	})

	t.Run("generate nonce", func(t *testing.T) {
		test.Ok(t, w1.GenerateNonce())
		test.Equals(t, false, w1.VerifySignature())
	})
}
