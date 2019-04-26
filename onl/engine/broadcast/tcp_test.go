package broadcast_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
	"github.com/advanderveer/27067dd17/onl/engine/broadcast"
	"github.com/advanderveer/go-test"
)

var _ engine.Broadcast = &broadcast.TCP{}

func TestTCPBroadcast(t *testing.T) {

	//setup tcp endpoints
	bc1, err := broadcast.NewTCP(os.Stderr, ":0", 10, 100)
	test.Ok(t, err)
	bc2, err := broadcast.NewTCP(os.Stderr, ":0", 10, 100)
	test.Ok(t, err)
	bc3, err := broadcast.NewTCP(os.Stderr, ":0", 10, 100)
	test.Ok(t, err)

	//ring topology
	bc1.To(time.Millisecond*10, bc2.Addr())
	bc2.To(time.Millisecond*10, bc3.Addr())
	bc3.To(time.Millisecond*10, bc1.Addr())

	//write and pass along message
	msg1 := &engine.Msg{Block: &onl.Block{Round: 1}}
	test.Ok(t, bc1.Write(msg1))

	msg2 := &engine.Msg{}
	test.Ok(t, bc2.Read(msg2))
	test.Equals(t, msg2, msg1)
	test.Ok(t, bc2.Write(msg2))

	msg3 := &engine.Msg{}
	test.Ok(t, bc3.Read(msg3))
	test.Equals(t, msg3, msg1)
	test.Ok(t, bc3.Write(msg3))

	msg4 := &engine.Msg{}
	test.Ok(t, bc1.Read(msg4))
	test.Equals(t, msg4, msg1)

	//close down
	test.Ok(t, bc1.Close())
	test.Ok(t, bc2.Close())
	test.Ok(t, bc3.Close())

	//test usage after shutdown
	test.Equals(t, io.EOF, bc1.Read(msg4))
	test.Equals(t, broadcast.ErrClosed, bc1.Write(msg4))
}

func TestMaxConnHandling(t *testing.T) {
	nConn := 5

	//the max connection that we're handling
	bc1, _ := broadcast.NewTCP(os.Stderr, ":0", nConn, 100)

	//saturate bc1
	for i := 0; i < nConn; i++ {
		bc, _ := broadcast.NewTCP(os.Stderr, ":0", nConn, 100)
		test.Ok(t, bc.To(time.Millisecond*10, bc1.Addr()))
	}

	//start another conn
	bc2, _ := broadcast.NewTCP(os.Stderr, ":0", nConn, 100)
	test.Ok(t, bc2.To(time.Millisecond*10, bc1.Addr()))

	//writing to the new connection should fail
	test.Assert(t, bc2.Write(&engine.Msg{}) != nil, "write should fail")
}
