package bcast

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/advanderveer/go-test"
)

func TestServerServeClose(t *testing.T) {
	s, err := NewServer(os.Stderr, net.IPv4(127, 0, 0, 1), 0, StaticAddr(net.ParseIP("192.168.0.1")), nil)
	test.Ok(t, err)

	iip, eip, port, ca := s.Info()
	test.Assert(t, len(ca) > 200, "should have some cert data")
	test.Equals(t, "127.0.0.1", iip.String())
	test.Equals(t, "192.168.0.1", eip.String())
	test.Assert(t, port > 0, "should have a gotten a port assigned")

	done := make(chan struct{})
	go func() {
		err = s.Serve()
		test.Ok(t, err)
		done <- struct{}{}
	}()

	t.Run("closing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err = s.Close(ctx)
		test.Ok(t, err)

		t.Run("should be able to rebind on the old port", func(t *testing.T) {
			s, err := NewServer(os.Stderr, net.IPv4(127, 0, 0, 1), port, StaticAddr(net.ParseIP("127.0.0.1")), nil)
			test.Ok(t, err)
			test.Ok(t, s.Close(ctx))
		})
	})
}
