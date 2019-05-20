package wall

import "github.com/cockroachdb/apd"

// Params contains protocol parameters
type Params struct {

	//MaxDepositTTL describes the number of rounds a deposit can be locked for
	MaxDepositTTL uint64

	//DecimalContext context returns a context for arbitrary decimal calculations
	DecimalContext *apd.Context

	//RoundCoefficient configures the chance that someone draws a winning ticket
	//for a round if it controls 100% of the stake
	RoundCoefficient *apd.Decimal
}

//DefaultParams returns sensible default params
func DefaultParams() *Params {
	return &Params{
		DecimalContext:   apd.BaseContext.WithPrecision(50),
		MaxDepositTTL:    100,
		RoundCoefficient: apd.New(999, -3),
	}
}
