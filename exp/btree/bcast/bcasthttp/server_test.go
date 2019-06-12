package bcasthttp

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	bcast "github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
)

func TestServerServeClose(t *testing.T) {
	s, err := NewServer(os.Stderr, 1, net.IPv4(127, 0, 0, 1), 0, StaticAddr(net.ParseIP("192.168.0.1")), nil)
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
			s, err := NewServer(os.Stderr, 1, net.IPv4(127, 0, 0, 1), port, StaticAddr(net.ParseIP("127.0.0.1")), nil)
			test.Ok(t, err)
			test.Ok(t, s.Close(ctx))
		})
	})
}

func TestClosingOfOpenPushConnections(t *testing.T) {

	// setup handler that keeps reading messages
	h := NewHandler(os.Stderr, 10)
	go func() {
		for {
			h.Read()
		}
	}()

	s, err := NewServer(os.Stderr, 1, net.IPv4(127, 0, 0, 1), 0, StaticAddr(net.ParseIP("192.168.0.1")), h)
	test.Ok(t, err)
	done := make(chan struct{})
	go func() {
		s.Serve()
		done <- struct{}{}
	}()

	ip, _, port, ca := s.Info()
	cert, err := ParseCertificate(ca)
	test.Ok(t, err)

	//setup client that keeps pushing messages
	c := NewClient(os.Stderr, ip, port, cert, 0)
	go func() {
		for {
			err := c.Write(&bcast.Msg{Foo: "bar"})
			if err == ErrClientClosed {
				done <- struct{}{}
				return
			}
		}
	}()

	// close handler and give time for panics to occur
	test.Ok(t, h.Close())
	time.Sleep(time.Millisecond * 150)
	test.Ok(t, s.Close(context.Background()))
	<-done
	<-done

	// client should be closed also
	test.Equals(t, ErrClientClosed, c.Write(&bcast.Msg{Foo: "bar"}))
}

func TestMaxConns(t *testing.T) {
	h := NewHandler(os.Stderr, 10)
	s, err := NewServer(os.Stderr, 1, net.IPv4(127, 0, 0, 1), 0, StaticAddr(net.ParseIP("192.168.0.1")), h)
	test.Ok(t, err)
	defer s.Close(context.Background())
	go s.Serve()

	ip, _, port, ca := s.Info()
	cert, err := ParseCertificate(ca)
	test.Ok(t, err)

	c := NewClient(os.Stderr, ip, port, cert, 0)
	err = c.Write(&bcast.Msg{Foo: "bar"})
	test.Ok(t, err)

	// sync openend clients should timeout as the max amount of accepted
	// connections is reached after 1
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			c := NewClient(os.Stderr, ip, port, cert, time.Millisecond*100)
			err := c.Write(&bcast.Msg{Foo: "bar"})
			test.Equals(t, ErrClientClosed, err)
			test.Ok(t, c.Close())
		}(i)
	}

	wg.Wait()

	test.Ok(t, c.Close())
}
