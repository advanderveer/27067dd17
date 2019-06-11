package bcast_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"

	"github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
)

func TestPoolCycling(t *testing.T) {
	dir, err := ioutil.TempDir("", "peers_")
	test.Ok(t, err)

	peers := bcast.NewPeers(5, 2000, dir)
	test.Ok(t, peers.Cycle())
	test.Equals(t, 0, peers.Len())

	t.Run("add manual", func(t *testing.T) {
		test.Ok(t, peers.Put("10.1.1.1", `{}`)) //will be overwritten by conf file
		test.Equals(t, 1, peers.Len())
		p1 := peers.Get("10.1.1.1")
		test.Equals(t, net.IPv4(10, 1, 1, 1), p1.IP())
		test.Equals(t, uint16(443), p1.Port)

		err = peers.Put("10.1.1.2", `{"port": 1100}`)
		test.Ok(t, err)
		test.Equals(t, 2, peers.Len())
		test.Equals(t, uint16(1100), peers.Get("10.1.1.2").Port)
	})

	t.Run("add by writing file", func(t *testing.T) {
		ioutil.WriteFile(filepath.Join(dir, "peers.list"), []byte(
			`2001:db8:85a3:0:0:8a2e:370:7334 {"port": 1200}
10.1.1.1`,
		), 0666)
		test.Ok(t, peers.Cycle())
		test.Equals(t, 3, peers.Len())
		test.Equals(t, uint16(443), peers.Get("10.1.1.1").Port)
		test.Equals(t, uint16(1200), peers.Get("2001:db8:85a3::8a2e:370:7334").Port) //more efficient notation
		test.Equals(t, uint16(1100), peers.Get("10.1.1.2").Port)

		data, _ := ioutil.ReadFile(filepath.Join(dir, "peers.list"))

		// should not contain the port 80 that is the default conf
		test.Equals(t, false, bytes.Contains(data, []byte("443")))

		// should have written the more efficient notation of the ipv6 addr
		test.Equals(t, true, bytes.Contains(data, []byte("2001:db8:85a3::8a2e:370:7334")))

		// top should be lower then the max because we don't know of more peers
		test.Equals(t, 3, len(peers.Top()))
	})

	t.Run("add many and cycle", func(t *testing.T) {
		for i := 0; i < 250; i++ {
			for j := 0; j < 10; j++ {
				test.Ok(t, peers.Put(fmt.Sprintf("13.0.%d.%d", j, i), ""))

				//very first put is still below min so we expect it to show up
				//in the top right away. (unsorted)
				if i == 0 && j == 0 {
					test.Equals(t, 4, len(peers.Top()))
				}
			}
		}

		test.Equals(t, 2503, peers.Len())
		test.Ok(t, peers.Cycle())           //arge cycle should be fine
		test.Equals(t, 2000, peers.Len())   //should be trimmed
		test.Equals(t, 5, len(peers.Top())) //should be the min nr of peers
	})

}
