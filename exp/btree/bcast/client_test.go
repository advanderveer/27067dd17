package bcast_test

import (
	"io/ioutil"
	"testing"

	"github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
)

func testAgent(t *testing.T) (a *bcast.Agent) {
	dir, err := ioutil.TempDir("", "bcast_agent_")
	test.Ok(t, err)

	p := bcast.NewPeers(5, 200, dir)
	a, err = bcast.NewAgent("127.0.0.1:0", p)
	test.Ok(t, err)
	return
}

func TestClientPushPull(t *testing.T) {
	a := testAgent(t)

	// create a client that is connected to the agent
	c, err := bcast.NewClient(a.Addr())
	test.Ok(t, err)

	// write to the agent should relay to any connected clients
	go func() {
		err = a.Write(&bcast.Msg{Foo: "bar"})
		test.Ok(t, err)
	}()

	// should now be able to read from the client
	msg, err := c.Read()
	test.Ok(t, err)
	test.Equals(t, "bar", msg.Foo)

	// client should also be able to write to server
	go func() {
		err = c.Write(&bcast.Msg{Foo: "foobar"})
		test.Ok(t, err)
	}()

	msg, err = a.Read()
	test.Ok(t, err)
	test.Equals(t, "foobar", msg.Foo)

	test.Ok(t, c.Close())
	test.Ok(t, a.Close())
}
