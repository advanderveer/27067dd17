package bcasthttp

import (
	"context"
	"net"
	"os"
	"testing"

	bcast "github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
)

var _ bcast.Writer = &Client{}

func TestClientToServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := NewHandler(os.Stderr, 0)
	s, err := NewServer(os.Stderr, 1, net.IPv4(127, 0, 0, 1), 0, StaticAddr(nil), h)
	test.Ok(t, err)

	go s.Serve()

	ip, _, port, ca := s.Info()
	cert, err := ParseCertificate(ca)
	test.Ok(t, err)
	c := NewClient(os.Stderr, ip, port, cert, 0)

	//write from client to server
	test.Ok(t, c.Write(&bcast.Msg{Foo: "bar"}))

	//should now read on the server
	msg, err := h.Read()
	test.Ok(t, err)
	test.Equals(t, "bar", msg.Foo)

	//close client
	test.Ok(t, c.Close())

	//writing another message should cause closed error
	test.Equals(t, ErrClientClosed, c.Write(&bcast.Msg{Foo: "bar"}))

	test.Ok(t, h.Close())
	test.Ok(t, s.Close(ctx))
}

func TestClientToNonExistingServer(t *testing.T) {
	c := NewClient(os.Stderr, net.IP{127, 0, 0, 1}, 11000, nil, 0)

	// write should return closed immediately
	test.Equals(t, ErrClientClosed, c.Write(&bcast.Msg{Foo: "bar"}))

	// closing should still be possible
	test.Ok(t, c.Close())
}
