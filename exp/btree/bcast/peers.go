package bcast

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/natefinch/atomic"
	"github.com/pkg/errors"
)

// Peers holds known peer endpoints that can be connected to. It is periodially
// cycled such that old peers are removed and a new list is persisted
type Peers struct {
	dir  string
	info map[string]*PeerInfo
	top  []string
	min  int
	max  int
	mu   sync.RWMutex
}

// PeerInfo holds the data about a specific peer
type PeerInfo struct {
	Port uint16 `json:"port"`
	CA   []byte `json:"ca"`

	//@TODO add a backoff that can be increased when peers are encountered repeatedly
	// in the selection loop.

	ip   net.IP
	opts string
}

// IP returns the ip of a peer
func (pi *PeerInfo) IP() net.IP { return pi.ip }

// NewPeers inits a new pool
func NewPeers(min, max int, dir string) (p *Peers) {
	if min > max {
		panic("min peers should be lower or equal to max peers")
	}

	p = &Peers{info: make(map[string]*PeerInfo), min: min, max: max, dir: dir}
	return
}

// Len returns the number of peers we're keeping
func (p *Peers) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.info)
}

// Get a peer by its addr
func (p *Peers) Get(addr string) (pi *PeerInfo) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pi = p.info[addr]
	return
}

// Put an host to the pool, overwriting existing info
func (p *Peers) Put(addr string, opts string) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.put(addr, opts)
}

func (p *Peers) put(addr string, opts string) (err error) {
	pi := &PeerInfo{Port: 443, opts: opts} //default peer options
	if opts != "" {
		err = json.Unmarshal([]byte(opts), pi)
		if err != nil {
			return fmt.Errorf("invalid peer configuration, must be valid json: %v", err)
		}
	}

	pi.ip = net.ParseIP(addr)
	if pi.ip == nil {
		return fmt.Errorf("ip for host must be an ipv4/v6 IP address")
	}

	p.info[pi.ip.String()] = pi
	if len(p.top) < p.min {

		// if there are less peers in the top peers then the min we need
		// append it right away to the top peers.
		p.top = append(p.top, addr)
	}

	return
}

func (p *Peers) sorted() (pis []*PeerInfo) {
	for _, pi := range p.info {
		pis = append(pis, pi)
	}

	sort.Slice(pis, func(i int, j int) bool {
		//@TODO sort by some sort of weight or score
		return pis[i].IP().String() > pis[j].IP().String()
	})

	return
}

// Top returns a snapshot of the top ranking peers at the time of calling
func (p *Peers) Top() (top []PeerInfo) {
	for _, addr := range p.top {
		pi := p.info[addr]
		if pi == nil {
			panic("found top peer info that is not known")
		}

		top = append(top, *pi)
	}

	return
}

// Cycle peer data with the on-disk file, determine new top ranking peers and
// trim old peers.
func (p *Peers) Cycle() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// create or open peers file
	path := filepath.Join(p.dir, "peers.list")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if os.IsExist(err) {
		f, err = os.Open(path)
	}

	if err != nil {
		return errors.Wrap(err, "failed to open or create peers file")
	}

	defer f.Close()

	// read and merge whatever is on disk
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		fields := strings.SplitN(scan.Text(), " ", 2)
		if len(fields) < 1 {
			return fmt.Errorf("invalid line in peers file, must be '<host>[ <json>]'")
		}

		var opts string
		if len(fields) > 1 {
			opts = fields[1]
		}

		err = p.put(string(fields[0]), opts)
		if err != nil {
			return err
		}
	}

	if err = scan.Err(); err != nil {
		return errors.Wrap(err, "failed to scan peers file")
	}

	// write peers to buf, if there are more peers then the
	// configured max we remove the lowest rating peers here.
	buf := bytes.NewBuffer(nil)
	p.top = []string{}
	for i, pi := range p.sorted() {

		// anything beyond the max will be deleted permanently
		if i > (p.max - 1) {
			delete(p.info, pi.IP().String())
			continue
		}

		// update top to contain the sorted peers up to the min amount configured
		if i < p.min {
			p.top = append(p.top, pi.IP().String())
		}

		if pi.opts != "" {
			fmt.Fprintf(buf, "%s %s\n", pi.IP(), pi.opts)
		} else {
			fmt.Fprintf(buf, "%s\n", pi.IP())
		}
	}

	// overwrite peers file atomically
	return atomic.WriteFile(path, buf)
}
