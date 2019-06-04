package ttl_test

import (
	"testing"
	"time"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/ttl"
	"github.com/advanderveer/go-test"
)

//tickets of various of strength 1,2,3 etc
var (
	ticketS1 = [slot.TicketSize]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS2 = [slot.TicketSize]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ticketS3 = [slot.TicketSize]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

func TestTimerExpire(t *testing.T) {
	netw := ttl.NewMemNetwork()

	w := netw.Endpoint()
	to := time.Second

	r1 := ttl.NewRound(netw.Endpoint(), 1)
	go r1.Start(to)

	test.Ok(t, w.Write(&ttl.Msg{Prio: ticketS2}))
	test.Ok(t, w.Write(&ttl.Msg{Prio: ticketS1}))

	r2 := <-r1.Done()
	go r2.Start(to)

	<-r2.Done()

	// test.Equals(t, ticketS2, r2.Seed())
}
