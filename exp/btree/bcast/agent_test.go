package bcast_test

import (
	"io/ioutil"
	"testing"

	"github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
)

var _ bcast.Broadcast = &bcast.Agent{}
var _ bcast.Broadcast = &bcast.Client{}
var _ bcast.Broadcast = &bcast.Handler{}

func TestAgentSetupAndClose(t *testing.T) {
	dir1, err := ioutil.TempDir("", "bcast_agent_")
	test.Ok(t, err)

	a1, err := bcast.NewAgent("127.0.0.1:0", bcast.NewPeers(5, 200, dir1))
	test.Ok(t, err)
	h1, p1, c1 := a1.Addr()
	test.Equals(t, "127.0.0.1", h1.String())
	test.Assert(t, p1 != 0, "should have port")
	test.Assert(t, len(c1) > 300, "should have some cert")
	defer a1.Close()

	dir2, err := ioutil.TempDir("", "bcast_agent_")
	test.Ok(t, err)
	a2, err := bcast.NewAgent("localhost:10000", bcast.NewPeers(5, 200, dir2))
	test.Ok(t, err)
	h2, p2, c2 := a2.Addr()
	test.Equals(t, "127.0.0.1", h2.String())
	test.Equals(t, uint16(10000), p2)
	test.Assert(t, len(c2) > 300, "should have some cert")

	t.Run("should be able to bind after close", func(t *testing.T) {
		test.Ok(t, a2.Close())
		_, err = bcast.NewAgent("localhost:10000", bcast.NewPeers(5, 200, dir1))
		test.Ok(t, err)
	})

	t.Run("join each other", func(t *testing.T) {
		test.Ok(t, a2.Join(a1.Addr()))
		test.Ok(t, a2.Join(a2.Addr()))
	})
}

func TestManualClientServer(t *testing.T) {
	dir1, err := ioutil.TempDir("", "bcast_agent_")
	test.Ok(t, err)
	dir2, err := ioutil.TempDir("", "bcast_agent_")
	test.Ok(t, err)

	p1 := bcast.NewPeers(5, 200, dir1)
	a1, _ := bcast.NewAgent("127.0.0.1:0", p1)
	p2 := bcast.NewPeers(5, 200, dir2)
	a2, _ := bcast.NewAgent("127.0.0.1:0", p2)
	test.Ok(t, a1.Join(a2.Addr()))
	test.Ok(t, a2.Join(a1.Addr()))

	test.Equals(t, 1, len(p1.Top()))
	test.Equals(t, 1, len(p2.Top()))

	// t.Run("client creation", func(t *testing.T) {
	// 	c2to1, err := bcast.NewClient(p2.Top()[0])
	// 	test.Ok(t, err)
	// 	c1to2, err := bcast.NewClient(p1.Top()[0])
	// 	test.Ok(t, err)
	//
	// 	test.Ok(t, c1to2.Close())
	// 	test.Ok(t, c2to1.Close())
	// })
}
