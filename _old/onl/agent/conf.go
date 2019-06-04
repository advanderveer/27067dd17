package agent

import (
	"io"
	"os"
	"time"

	"github.com/advanderveer/27067dd17/onl"
)

//Conf configures the agent
type Conf struct {
	//Logs will be written to the writer
	LogWriter io.Writer

	//The tcp address the agent will bind on
	Bind string

	//Maximum incoming broadcast connections
	MaxIncomingConn int

	//MaxMessageBuf is the maximum nr of messages the broadcast endpoint buffers
	MaxMessageBuf int

	//The time each agent will leave open the round for blocks to be sent in
	RoundTime time.Duration

	//The identity this agent will assume
	Identity *onl.Identity

	//genf is configured through StartWithStake
	genf func(kv *onl.KV)
}

//StartWithStake instructs the agent to start with a genesis block that encodes
//a certain amount of stake for the provided identities
func (c *Conf) StartWithStake(stake uint64, idns ...*onl.Identity) (err error) {
	c.genf = func(kv *onl.KV) {
		for _, idn := range idns {
			kv.CoinbaseTransfer(idn.PK(), stake)
			kv.DepositStake(idn.PK(), stake, idn.TokenPK())
		}
	}

	return
}

//DefaultConf returns sensible defaults
func DefaultConf() *Conf {
	return &Conf{
		LogWriter:       os.Stderr,
		Bind:            ":0",
		MaxIncomingConn: 10,
		MaxMessageBuf:   100,
		RoundTime:       time.Second,
		Identity:        onl.NewIdentity(nil),
	}
}
