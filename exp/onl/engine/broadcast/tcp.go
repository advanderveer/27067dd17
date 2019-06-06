package broadcast

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/advanderveer/27067dd17/onl"
	"github.com/advanderveer/27067dd17/onl/engine"
)

//TCP broadcast endpoint
type TCP struct {
	ln     net.Listener
	logs   *log.Logger
	peers  map[net.Addr]*tcppeer
	mu     sync.RWMutex
	cwg    sync.WaitGroup
	conns  chan net.Conn
	closed bool
	in     chan *engine.Msg
}

type tcppeer struct {
	conn net.Conn
	enc  *gob.Encoder
}

//NewTCP will start a new tcp endpoint, listening for incoming connections
func NewTCP(logw io.Writer, bind string, maxConn, maxBuf int) (bc *TCP, err error) {
	bc = &TCP{
		conns: make(chan net.Conn, maxConn),
		logs:  log.New(logw, "", 0),
		in:    make(chan *engine.Msg, maxBuf),
		peers: make(map[net.Addr]*tcppeer),
	}

	//listen on a random available port
	bc.ln, err = net.Listen("tcp", bind)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	//start handling connections
	go func() {
		defer close(bc.conns)

		for {
			conn, err := bc.ln.Accept()
			if err != nil {

				//still the recommended way of handling: @see https://github.com/golang/go/issues/4373
				if !strings.Contains(err.Error(), "use of closed network connection") {
					bc.logs.Printf("[ERRO] failed to accept broadcast tcp connection: %v", err)
				}

				return
			}

			//if curren buffer is full immediately close new connections
			if len(bc.conns) >= cap(bc.conns) {
				conn.Close()
				continue
			}

			//keep track of the connections so we can close them
			bc.conns <- conn

			//handle each connection concurrently
			go bc.handleConn(conn)
		}
	}()

	return
}

func (bc *TCP) handleConn(conn net.Conn) {
	bc.cwg.Add(1)
	defer bc.cwg.Done()

	//start decoding
	dec := gob.NewDecoder(conn)
	for {
		msg := &engine.Msg{}
		err := dec.Decode(msg)
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				bc.logs.Printf("[ERRO] failed to decode message from %s: %v", conn.RemoteAddr(), err)
			}

			return
		}

		//for sync messages we provide a return func for bi-directional sending
		if msg.Sync != nil {
			enc := gob.NewEncoder(conn)
			msg.Sync.SetWF(func(b *onl.Block) (err error) {
				err = enc.Encode(&engine.Msg{Block: b})
				if err != nil && err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
					bc.logs.Printf("[ERRO] failed to encode sync message to %s: %v", conn.RemoteAddr(), err)
				}

				return nil
			})
		}

		//send to incoming channel for consumer to read from
		bc.in <- msg
	}
}

//To will configure this broadcast endpoint to send any writes to these peers
func (bc *TCP) To(to time.Duration, peers ...net.Addr) (err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range peers {
		if _, ok := bc.peers[p]; ok {
			continue //peer already exists
		}

		//open conn
		conn, err := net.DialTimeout(p.Network(), p.String(), to)
		if err != nil {
			return fmt.Errorf("failed to dial peer: %v", err)
		}

		//keep conn info for later writing
		bc.peers[p] = &tcppeer{
			conn: conn,
			enc:  gob.NewEncoder(conn),
		}

		//handle incoming message from connecting peers
		go bc.handleConn(conn)

		//@TODO handle any incoming messages from the peers we connect to
	}

	return
}

// Addr returns the address this tcp endpoint is listening on
func (bc *TCP) Addr() (addr net.Addr) {
	return bc.ln.Addr()
}

//Read a message from the broadcast
func (bc *TCP) Read(msg *engine.Msg) (err error) {
	rmsg := <-bc.in
	if rmsg == nil {
		return io.EOF
	}

	*msg = *rmsg
	return
}

//Write a message to the broadcast
func (bc *TCP) Write(msg *engine.Msg) (err error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.closed {
		return ErrClosed
	}

	for _, p := range bc.peers {
		err = p.enc.Encode(msg)
		if err != nil {
			return fmt.Errorf("failed to encode msg for peer: %v", err)
		}
	}

	return
}

//Close the tcp broadcast
func (bc *TCP) Close() (err error) {
	err = bc.ln.Close()
	if err != nil {
		return fmt.Errorf("failed to close tcp listener: %v", err)
	}

	//shutdown open incoming conns
	for c := range bc.conns {
		err = c.Close()
		if err != nil {
			return fmt.Errorf("failed to close incoming tcp conn: %v", err)
		}
	}

	//shutdown outgoing connections
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, p := range bc.peers {
		err = p.conn.Close()
		if err != nil {
			return fmt.Errorf("failed to close tcp connection to peer: %v", err)
		}
	}

	bc.cwg.Wait() //wait for incoming conn's to end
	close(bc.in)  //read will now return EOF

	//mark as closed
	bc.closed = true
	return
}
