package agent

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/27067dd17/onl/engine/clock"
)

//Agent represents a single identity in the network
type Agent struct {
	broadcast *broadcast.TCP
	clock     *clock.WallClock
	store     onl.Store
	clean     func()
	engine    *engine.Engine
}

//New allocates the agent
func New(cfg *Conf) (a *Agent, err error) {
	a = &Agent{}

	a.broadcast, err = broadcast.NewTCP(cfg.LogWriter, cfg.Bind, cfg.MaxIncomingConn, cfg.MaxMessageBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tcp broadcast: %v", err)
	}

	a.clock = clock.NewWallClock(cfg.RoundTime)

	a.store, a.clean = onl.TempBadgerStore()

	chain, _, err := onl.NewChain(a.store, a.clock.Round(), cfg.genf)
	if err != nil {
		return nil, fmt.Errorf("failed to initalize chain: %v", err)
	}

	a.engine = engine.New(cfg.LogWriter, a.broadcast, a.clock, cfg.Identity, chain)

	return
}

func (a *Agent) Join(peers ...net.Addr) (err error) {
	return a.broadcast.To(time.Second, peers...)
}

func (a *Agent) Addr() net.Addr {
	return a.broadcast.Addr()
}

func (a *Agent) Draw(w io.Writer) (err error) {
	return a.engine.Draw(w)
}

func (a *Agent) Close() (err error) {
	err = a.engine.Shutdown(context.Background())
	if err != nil {
		return err
	}

	a.clean()

	return
}
