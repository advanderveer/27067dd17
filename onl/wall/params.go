package wall

import (
	"github.com/advanderveer/27067dd17/vrf"
	"github.com/cockroachdb/apd"
)

// Params contains protocol parameters
type Params struct {

	//MaxDepositTTL describes the number of rounds a deposit can be locked for
	MaxDepositTTL uint64

	//DecimalContext context returns a context for arbitrary decimal calculations
	DecimalContext *apd.Context

	//RoundCoefficient configures the chance that someone draws a winning ticket
	//for a round if it controls 100% of the stake
	RoundCoefficient *apd.Decimal

	//GenesisTicket is the random vrf token that all other tokens will decent from
	GenesisTicket [vrf.Size]byte

	//GenesisIdentityRnd are the random bytes for the genesis signing identity
	GenesisIdentityRnd []byte
}

//DefaultParams returns sensible default params
func DefaultParams() *Params {
	return &Params{
		DecimalContext:   apd.BaseContext.WithPrecision(50),
		MaxDepositTTL:    100,
		RoundCoefficient: apd.New(999, -3),
		GenesisTicket: [vrf.Size]byte{
			0xd8, 0x81, 0xfe, 0x39, 0x87, 0xaa, 0xe6, 0xe4, 0x5e, 0x38, 0xfb, 0x7b, 0x26, 0x6a, 0x2f, 0xd0,
			0x0b, 0x35, 0x06, 0x01, 0x52, 0x89, 0x0d, 0x12, 0x32, 0x8b, 0xb0, 0xc0, 0xca, 0xe2, 0xb4, 0x01,
		},
	}
}
