package bls

import (
	"fmt"

	"github.com/dfinity/go-dfinity-crypto/bls"
)

func init() {
	err := bls.Init(int(bls.CurveFp254BNb))
	if err != nil {
		panic(fmt.Errorf("failed to init BLS library: %v", err))
	}
}
