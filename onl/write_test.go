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
	test.Equals(t, "374708ff", fmt.Sprintf("%.4x", w1.Hash()))

	w1.TimeCommit = 1
	test.Equals(t, "7c3ccd10", fmt.Sprintf("%.4x", w1.Hash()))

	w1.TimeStart = 1
	test.Equals(t, "532deabf", fmt.Sprintf("%.4x", w1.Hash()))

	w1.ReadRows.Add([]byte{0x01})
	test.Equals(t, "e1198b57", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x02})
	test.Equals(t, "a1804c32", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x03})
	test.Equals(t, "01da5699", fmt.Sprintf("%.4x", w1.Hash()))

	w1.WriteRows.Add([]byte{0x01}, []byte{0x03}) //shouldn't change anything
	test.Equals(t, "01da5699", fmt.Sprintf("%.4x", w1.Hash()))
}
